package etl

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"aresdata/pkg/utils"

	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

// FeiguaVideoSummaryDTO 直接在Go中定义，用于解析JSON
type FeiguaVideoSummaryDTO struct {
	PlayCountStr       string `json:"PlayCountStr"`
	LikeCountStr       string `json:"LikeCountStr"`
	CommentCountStr    string `json:"CommentCountStr"`
	ShareCountStr      string `json:"ShareCountStr"`
	CollectCountStr    string `json:"CollectCountStr"`
	InteractionRateStr string `json:"InteractionRateStr"`
	Score              string `json:"Score"`
	LikeCommentRateStr string `json:"LikeCommentRateStr"`
	SalesGmvStr        string `json:"SalesGmvStr"`
	SalesCountStr      string `json:"SalesCountStr"`
	GoodsCountStr      string `json:"GoodsCountStr"`
	GPM                string `json:"GPM"`
	AwemeType          int32  `json:"AwemeType"`
}

// FeiguaVideoSummaryData 对应飞瓜响应的整体结构
// FeiguaVideoSummaryData 对应飞瓜响应的整体结构，包含状态和数据
// 通过嵌套 FeiguaBaseResponse 提供 Status/Msg/Code 字段
// Data 字段为业务数据
// Go 的 json 库会自动处理嵌套
type FeiguaVideoSummaryData struct {
	FeiguaBaseResponse
	Data FeiguaVideoSummaryDTO `json:"Data"`
}

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
	ds := &data.SourceData{
		ID:             rawData.Id,
		ProviderName:   rawData.ProviderName,
		DataType:       rawData.DataType,
		EntityId:       rawData.EntityId,
		Status:         rawData.Status,
		FetchedAt:      utils.ParseTimeRFC3339(rawData.FetchedAt),
		Date:           rawData.Date,
		RawContent:     rawData.RawContent,
		ProcessingLog:  rawData.ProcessingLog,
		Retries:        int(rawData.Retries),
		RequestMethod:  rawData.RequestMethod,
		RequestUrl:     rawData.RequestUrl,
		RequestParams:  rawData.RequestParams,
		RequestHeaders: rawData.RequestHeaders,
	}
	return p.ProcessData(ctx, ds)
}

// ProcessData 保留原有业务逻辑，接收*data.SourceData参数
func (p *VideoSummaryProcessor) ProcessData(ctx context.Context, rawData *data.SourceData) error {
	// 1. 一次性解析包含状态和数据的完整响应
	var resp FeiguaVideoSummaryData
	if err := json.Unmarshal([]byte(rawData.RawContent), &resp); err != nil {
		p.log.Errorf("failed to unmarshal video summary response for sourceID %d: %v", rawData.ID, err)
		return &ProcessError{Msg: "unmarshal video summary response failed", SourceID: rawData.ID, Err: err}
	}

	// 2. 预检API业务状态
	if !resp.Status {
		logMsg := fmt.Sprintf("API returned error status: Code=%d, Msg=%s", resp.Code, resp.Msg)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.ID, -1, logMsg)
	}

	summary := resp.Data
	now := time.Now()

	// 直接将DTO中的所有字段原样赋值给数据模型
	videoDim := &data.Video{
		AwemeId:            rawData.EntityId,
		PlayCountStr:       summary.PlayCountStr,
		LikeCountStr:       summary.LikeCountStr,
		CommentCountStr:    summary.CommentCountStr,
		ShareCountStr:      summary.ShareCountStr,
		CollectCountStr:    summary.CollectCountStr,
		InteractionRateStr: summary.InteractionRateStr,
		ScoreStr:           summary.Score,
		LikeCommentRateStr: summary.LikeCommentRateStr,
		SalesGmvStr:        summary.SalesGmvStr,
		SalesCountStr:      summary.SalesCountStr,
		GoodsCountStr:      summary.GoodsCountStr,
		GpmStr:             summary.GPM,
		AwemeType:          summary.AwemeType,
		SummaryUpdatedAt:   &now,
	}

	// 3. 调用 videoRepo.UpdateSummary 安全地只更新总览相关字段
	if err := p.videoRepo.UpdateFromSummary(ctx, videoDim); err != nil {
		return &ProcessError{Msg: "update video summary failed", SourceID: rawData.ID, Err: err}
	}

	return p.sourceDataRepo.UpdateStatus(ctx, rawData.ID, 1)
}
