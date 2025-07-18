package etl

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	v1 "aresdata/api/v1"
	"aresdata/internal/data"

	"github.com/go-kratos/kratos/v2/log"
)

// FeiguaVideoTrendItem 对应趋势接口中Data数组的单个对象结构
type FeiguaVideoTrendItem struct {
	LikeCount     json.Number `json:"LikeCount"`
	CommentCount  json.Number `json:"CommentCount"`
	ShareCount    json.Number `json:"ShareCount"`
	CollectCount  json.Number `json:"CollectCount"`
	SalesGmv      json.Number `json:"SalesGmv"`
	SalesCount    json.Number `json:"SalesCount"`
	Fans          json.Number `json:"Fans"`
	DateCode      int64       `json:"DateCode"`
	SalesGmvStr   string      `json:"SalesGmvStr"`
	SalesCountStr string      `json:"SalesCountStr"`
	GPMStr        string      `json:"GPMStr"`
}

// FeiguaVideoTrendResponse 对应完整的趋势接口响应
type FeiguaVideoTrendResponse struct {
	Data []FeiguaVideoTrendItem `json:"Data"`
}

// VideoTrendProcessor 负责处理视频每日趋势数据
type VideoTrendProcessor struct {
	sourceDataRepo     data.SourceDataRepo
	videoRepo          data.VideoRepo
	videoTrendStatRepo data.VideoTrendStatRepo
	log                *log.Helper
}

func NewVideoTrendProcessor(sdRepo data.SourceDataRepo, vRepo data.VideoRepo, vtsRepo data.VideoTrendStatRepo, logger log.Logger) *VideoTrendProcessor {
	return &VideoTrendProcessor{
		sourceDataRepo:     sdRepo,
		videoRepo:          vRepo,
		videoTrendStatRepo: vtsRepo,
		log:                log.NewHelper(log.With(logger, "module", "etl/video-trend")),
	}
}

// Process 解析视频趋势原始数据，并存入 video_daily_stats 表
func (p *VideoTrendProcessor) Process(ctx context.Context, rawData *v1.SourceData) error {
	var resp FeiguaVideoTrendResponse
	if err := json.Unmarshal([]byte(rawData.RawContent), &resp); err != nil {
		p.log.Errorf("failed to unmarshal video trend response for sourceID %d: %v", rawData.Id, err)
		return &ProcessError{Msg: "unmarshal video trend response failed", SourceID: rawData.Id, Err: err}
	}

	if len(resp.Data) == 0 {
		p.log.Warnf("video trend data is empty for sourceID %d", rawData.Id)
		return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
	}

	// 1. 更新 Video 维度表（只更新总览字段）
	latestTrend := resp.Data[len(resp.Data)-1]
	videoDim := &data.Video{
		AwemeId:            rawData.EntityId,
		TotalLikes:         toInt64(latestTrend.LikeCount),
		TotalComments:      toInt64(latestTrend.CommentCount),
		TotalShares:        toInt64(latestTrend.ShareCount),
		TotalCollects:      toInt64(latestTrend.CollectCount),
		TotalSalesGmv:      toInt64(latestTrend.SalesGmv),
		TotalSalesVolume:   toInt64(latestTrend.SalesCount),
		InteractionRateStr: "", // 需要根据实际数据赋值，如有字段请补充
		GpmStr:             latestTrend.GPMStr,
	}
	if err := p.videoRepo.Upsert(ctx, videoDim); err != nil {
		return &ProcessError{Msg: "upsert video dimension failed", SourceID: rawData.Id, Err: err}
	}

	// 2. 批量 upsert 每日趋势快照
	var dailyStatsToUpsert []*data.VideoTrendStat
	awemeId := rawData.EntityId
	for _, item := range resp.Data {
		date, err := time.Parse("20060102", strconv.FormatInt(item.DateCode, 10))
		if err != nil {
			p.log.Errorf("failed to parse datecode %d for awemeId %s: %v", item.DateCode, awemeId, err)
			continue
		}
		salesGmv, _ := item.SalesGmv.Float64()
		stat := &data.VideoTrendStat{
			AwemeId:           awemeId,
			Date:              date,
			TotalLikes:        toInt64(item.LikeCount),
			TotalComments:     toInt64(item.CommentCount),
			TotalShares:       toInt64(item.ShareCount),
			TotalCollects:     toInt64(item.CollectCount),
			TotalSalesGmv:     salesGmv,
			TotalSalesVolume:  toInt64(item.SalesCount),
			BloggerFansAtDate: toInt64(item.Fans),
		}
		dailyStatsToUpsert = append(dailyStatsToUpsert, stat)
	}
	if len(dailyStatsToUpsert) > 0 {
		if err := p.videoTrendStatRepo.BatchUpsert(ctx, dailyStatsToUpsert); err != nil {
			return &ProcessError{Msg: "batch upsert video trend stats failed", SourceID: rawData.Id, Err: err}
		}
	}

	return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
}
