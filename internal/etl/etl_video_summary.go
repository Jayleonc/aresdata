package etl

import (
	"aresdata/pkg/utils"
	"context"
	"encoding/json"
	"strconv"
	"strings"

	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"github.com/go-kratos/kratos/v2/log"
)

// VideoSummaryProcessor 负责处理视频总览数据
type VideoSummaryProcessor struct {
	sourceDataRepo data.SourceDataRepo
	videoRepo      data.VideoRepo
	log            *log.Helper
}

func NewVideoSummaryProcessor(sdRepo data.SourceDataRepo, vRepo data.VideoRepo, logger log.Logger) *VideoSummaryProcessor {
	return &VideoSummaryProcessor{
		sourceDataRepo: sdRepo,
		videoRepo:      vRepo,
		log:            log.NewHelper(log.With(logger, "module", "etl/video-summary")),
	}
}

func (p *VideoSummaryProcessor) Process(ctx context.Context, rawData *v1.SourceData) error {
	// 1. 解析最外层结构
	var resp v1.FeiguaVideoSummaryData
	if err := json.Unmarshal([]byte(rawData.RawContent), &resp); err != nil {
		p.log.Errorf("failed to unmarshal video summary response for sourceID %d: %v", rawData.Id, err)
		return &ProcessError{Msg: "unmarshal video summary response failed", SourceID: rawData.Id, Err: err}
	}

	summary := resp.Data
	if summary == nil {
		p.log.Warnf("video summary data is nil for sourceID %d", rawData.Id)
		return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1) // 标记为已处理
	}

	// 2. 转换数据并准备更新 video 维度表
	// 我们使用一个辅助函数来安全地将 "1.2w" 这样的字符串转为 int64
	totalLikes, _ := utils.ParseCountStr(summary.LikeCountStr)
	totalComments, _ := utils.ParseCountStr(summary.CommentCountStr)
	totalShares, _ := utils.ParseCountStr(summary.ShareCountStr)
	totalCollects, _ := utils.ParseCountStr(summary.CollectCountStr)

	// 解析销量和销售额，单位统一为分
	salesVolume, _ := utils.ParseSalesCount(summary.SalesCountStr)
	salesGmv, _ := utils.ParseSalesGmv(summary.SalesGmvStr)

	videoDim := &data.Video{
		AwemeId:            rawData.EntityId,
		TotalLikes:         totalLikes,
		TotalComments:      totalComments,
		TotalShares:        totalShares,
		TotalCollects:      totalCollects,
		TotalSalesVolume:   salesVolume,
		TotalSalesGmv:      salesGmv,
		InteractionRateStr: summary.InteractionRateStr,
		GpmStr:             summary.Gpm,
	}

	// 3. 调用 videoRepo.Upsert 更新维度表
	if err := p.videoRepo.Upsert(ctx, videoDim); err != nil {
		return &ProcessError{Msg: "upsert video dimension from summary failed", SourceID: rawData.Id, Err: err}
	}

	// 4. 更新 source_data 状态
	return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
}
