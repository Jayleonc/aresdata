package etl

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	v1 "aresdata/api/v1"
	"aresdata/internal/data"

	"github.com/go-kratos/kratos/v2/log"
)

// FeiguaVideoTrendItem 使用 json.Number 接收所有数值，增加所有字段
type FeiguaVideoTrendItem struct {
	DateCode           json.Number `json:"DateCode"`
	LikeCount          json.Number `json:"LikeCount"`
	LikeCountStr       string      `json:"LikeCountStr"`
	ShareCount         json.Number `json:"ShareCount"`
	ShareCountStr      string      `json:"ShareCountStr"`
	CommentCount       json.Number `json:"CommentCount"`
	CommentCountStr    string      `json:"CommentCountStr"`
	CollectCount       json.Number `json:"CollectCount"`
	CollectCountStr    string      `json:"CollectCountStr"`
	InteractionRate    json.Number `json:"InteractionRate"`
	InteractionRateStr string      `json:"InteractionRateStr"`
	SalesGmv           json.Number `json:"SalesGmv"`
	SalesGmvStr        string      `json:"SalesGmvStr"`
	SalesCount         json.Number `json:"SalesCount"`
	SalesCountStr      string      `json:"SalesCountStr"`
	Fans               json.Number `json:"Fans"`
	FansStr            string      `json:"FansStr"`
	GPM                json.Number `json:"GPM"`
	GPMStr             string      `json:"GPMStr"`
	IncLikeCount       json.Number `json:"IncLikeCount"`
	IncLikeCountStr    string      `json:"IncLikeCountStr"`
	IncShareCount      json.Number `json:"IncShareCount"`
	IncShareCountStr   string      `json:"IncShareCountStr"`
	IncCommentCount    json.Number `json:"IncCommentCount"`
	IncCommentCountStr string      `json:"IncCommentCountStr"`
	IncCollectCount    json.Number `json:"IncCollectCount"`
	IncCollectCountStr string      `json:"IncCollectCountStr"`
	IncSalesCount      json.Number `json:"IncSalesCount"`
	IncSalesCountStr   string      `json:"IncSalesCountStr"`
	IncSalesGmv        json.Number `json:"IncSalesGmv"`
	IncSalesGmvStr     string      `json:"IncSalesGmvStr"`
	IncFans            json.Number `json:"IncFans"`
	IncFansStr         string      `json:"IncFansStr"`
	ListTimeStr        string      `json:"ListTimeStr"`
	TimeStamp          json.Number `json:"TimeStamp"`
}

// FeiguaVideoTrendResponse 对应完整的趋势接口响应，包含状态
type FeiguaVideoTrendResponse struct {
	FeiguaBaseResponse
	Data []FeiguaVideoTrendItem `json:"Data"`
}

// VideoTrendProcessor 负责处理视频每日趋势数据
type VideoTrendProcessor struct {
	sourceDataRepo data.SourceDataRepo
	videoRepo      data.VideoRepo
	videoTrendRepo data.VideoTrendRepo
	log            *log.Helper
}

func NewVideoTrendProcessor(sdRepo data.SourceDataRepo, vRepo data.VideoRepo, vtRepo data.VideoTrendRepo, logger log.Logger) *VideoTrendProcessor {
	return &VideoTrendProcessor{
		sourceDataRepo: sdRepo,
		videoRepo:      vRepo,
		videoTrendRepo: vtRepo,
		log:            log.NewHelper(log.With(logger, "module", "etl/video-trend")),
	}
}

// Process 解析视频趋势原始数据，并存入 video_trends 表
func (p *VideoTrendProcessor) Process(ctx context.Context, rawData *v1.SourceData) error {
	var resp FeiguaVideoTrendResponse
	if err := json.Unmarshal([]byte(rawData.RawContent), &resp); err != nil {
		p.log.Errorf("failed to unmarshal video trend response for sourceID %d: %v", rawData.Id, err)
		return &ProcessError{Msg: "unmarshal video trend response failed", SourceID: rawData.Id, Err: err}
	}

	if !resp.Status {
		logMsg := fmt.Sprintf("API returned error status: Code=%d, Msg=%s", resp.Code, resp.Msg)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
	}

	if len(resp.Data) == 0 {
		p.log.Warnf("video trend data is empty for sourceID %d", rawData.Id)
		// 即使数据为空，也标记为成功，并更新时间戳，避免重复拉取
		_ = p.videoRepo.UpdateTrendTimestamp(ctx, rawData.EntityId)
		return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
	}

	awemeId := rawData.EntityId
	trendsToUpsert := make([]*data.VideoTrend, 0, len(resp.Data))
	for _, item := range resp.Data {
		dateCodeInt, _ := strconv.Atoi(item.DateCode.String())
		trend := &data.VideoTrend{
			AwemeId:            awemeId,
			DateCode:           dateCodeInt,
			LikeCount:          toInt64(item.LikeCount),
			LikeCountStr:       item.LikeCountStr,
			ShareCount:         toInt64(item.ShareCount),
			ShareCountStr:      item.ShareCountStr,
			CommentCount:       toInt64(item.CommentCount),
			CommentCountStr:    item.CommentCountStr,
			CollectCount:       toInt64(item.CollectCount),
			CollectCountStr:    item.CollectCountStr,
			InteractionRate:    toFloat64(item.InteractionRate),
			InteractionRateStr: item.InteractionRateStr,
			IncLikeCount:       toInt64(item.IncLikeCount),
			IncLikeCountStr:    item.IncLikeCountStr,
			IncShareCount:      toInt64(item.IncShareCount),
			IncShareCountStr:   item.IncShareCountStr,
			IncCommentCount:    toInt64(item.IncCommentCount),
			IncCommentCountStr: item.IncCommentCountStr,
			IncCollectCount:    toInt64(item.IncCollectCount),
			IncCollectCountStr: item.IncCollectCountStr,
			SalesCount:         toInt64(item.SalesCount),
			SalesCountStr:      item.SalesCountStr,
			SalesGmv:           toFloat64(item.SalesGmv),
			SalesGmvStr:        item.SalesGmvStr,
			Fans:               toInt64(item.Fans),
			FansStr:            item.FansStr,
			IncSalesCount:      toInt64(item.IncSalesCount),
			IncSalesCountStr:   item.IncSalesCountStr,
			IncSalesGmv:        toFloat64(item.IncSalesGmv),
			IncSalesGmvStr:     item.IncSalesGmvStr,
			IncFans:            toInt64(item.IncFans),
			IncFansStr:         item.IncFansStr,
			Gpm:                toFloat64(item.GPM),
			GpmStr:             item.GPMStr,
			ListTimeStr:        item.ListTimeStr,
			TimeStamp:          toInt64(item.TimeStamp),
		}
		trendsToUpsert = append(trendsToUpsert, trend)
	}

	if len(trendsToUpsert) > 0 {
		if err := p.videoTrendRepo.BatchUpsert(ctx, trendsToUpsert); err != nil {
			return &ProcessError{Msg: "batch upsert video trends failed", SourceID: rawData.Id, Err: err}
		}
	}

	if err := p.videoRepo.UpdateTrendTimestamp(ctx, awemeId); err != nil {
		p.log.Errorf("failed to update trend timestamp for awemeId %s: %v", awemeId, err)
	}

	return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
}
