// internal/etl/etl_video_detail.go

package etl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"strconv"
	"time"

	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/Jayleonc/aresdata/pkg/utils"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// VideoDetailProcessor 负责处理所有视频详情相关的数据，包括 summary 和 trend
type VideoDetailProcessor struct {
	log            *log.Helper
	sourceDataRepo data.SourceDataRepo
	videoRepo      data.VideoRepo
	bloggerRepo    data.BloggerRepo
	videoTrendRepo data.VideoTrendRepo
}

// NewVideoDetailProcessor .
func NewVideoDetailProcessor(
	logger log.Logger,
	sourceDataRepo data.SourceDataRepo,
	videoRepo data.VideoRepo,
	bloggerRepo data.BloggerRepo,
	videoTrendRepo data.VideoTrendRepo,
) *VideoDetailProcessor {
	return &VideoDetailProcessor{
		log:            log.NewHelper(log.With(logger, "module", "processor/video_detail")),
		sourceDataRepo: sourceDataRepo,
		videoRepo:      videoRepo,
		bloggerRepo:    bloggerRepo,
		videoTrendRepo: videoTrendRepo,
	}
}

// Process 实现了 Processor 接口，负责DTO到DO的转换和任务分发
// Process implements the Processor interface, routing data by type.
func (p *VideoDetailProcessor) Process(ctx context.Context, rawData *v1.SourceData) error {
	switch rawData.DataType {
	case "video_summary_headless":
		return p.processSummary(ctx, rawData)
	case "video_trend_headless":
		return p.processTrend(ctx, rawData)
	default:
		logMsg := fmt.Sprintf("未知的视频详情数据类型: %s", rawData.DataType)
		p.log.Warn(logMsg)
		// Assuming v1.SourceData and data.SourceData are compatible enough for this call
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
	}
}

// processSummary handles video overview data.
func (p *VideoDetailProcessor) processSummary(ctx context.Context, rawData *v1.SourceData) error {
	// 1. Unmarshal the JSON payload first to get the real-time like count.
	var resp FeiguaVideoSummaryData
	if err := json.Unmarshal([]byte(rawData.RawContent), &resp); err != nil {
		return &ProcessError{Msg: "unmarshal video summary response failed", SourceID: rawData.Id, Err: err}
	}

	if !resp.Status {
		logMsg := fmt.Sprintf("API(summary)返回错误: Code=%d, Msg=%s", resp.Code, resp.Msg)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
	}

	// 2. Get the blogger info to fetch the fan count.
	// We need to get the video record first to find the blogger_id.
	video, err := p.videoRepo.Get(ctx, rawData.EntityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logMsg := fmt.Sprintf("数据不一致(summary): videos 表中未找到 AwemeId %s", rawData.EntityId)
			return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
		}
		return &ProcessError{Msg: "failed to get video by awemeId", SourceID: rawData.Id, Err: err}
	}

	blogger, err := p.bloggerRepo.Get(ctx, video.BloggerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logMsg := fmt.Sprintf("数据不一致(summary): bloggers 表中未找到 BloggerId %d", video.BloggerId)
			return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
		}
		return &ProcessError{Msg: "failed to get blogger by id", SourceID: rawData.Id, Err: err}
	}

	// 3. Perform the CORRECT filtering logic.
	likeCount := utils.ParseUnitStrToInt64(resp.Data.LikeCountStr)
	fansCount := blogger.BloggerFansNum

	if likeCount <= 50 && fansCount <= 200 {
		logMsg := fmt.Sprintf("过滤条件触发(summary): 视频实时点赞数 (%d) 且博主粉丝数 (%d) 均不满足要求。", likeCount, fansCount)
		p.log.Infof(logMsg)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, 2, logMsg)
	}

	// 4. If not filtered, proceed to update the video dimension table.
	summary := resp.Data
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
		SummaryUpdatedAt:   utils.TimeToPtr(time.Now()),
	}

	if err := p.videoRepo.UpdateFromSummary(ctx, videoDim); err != nil {
		return &ProcessError{Msg: "update video summary failed", SourceID: rawData.Id, Err: err}
	}

	return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
}

// processTrend handles video trend data.
func (p *VideoDetailProcessor) processTrend(ctx context.Context, rawData *v1.SourceData) error {
	// --- 核心修正：独立的过滤数据获取逻辑 ---
	// 关键注释：处理趋势数据时，过滤条件所需的数据（点赞数、粉丝数）必须独立获取，
	// 绝不能依赖 processSummary 的执行结果，因为二者执行顺序不确定。

	// 1. 获取粉丝数 (Fans Count)
	//    此数据流： aweme_id -> videos 表 -> blogger_id -> bloggers 表 -> fans_count
	video, err := p.videoRepo.Get(ctx, rawData.EntityId)
	if err != nil {
		// 如果视频记录本身不存在，则无法继续，记录错误并跳过
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logMsg := fmt.Sprintf("数据前置条件不足(trend): videos 表中未找到 AwemeId %s", rawData.EntityId)
			return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
		}
		return &ProcessError{Msg: "failed to get video by awemeId for filtering", SourceID: rawData.Id, Err: err}
	}
	blogger, err := p.bloggerRepo.Get(ctx, video.BloggerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logMsg := fmt.Sprintf("数据前置条件不足(trend): bloggers 表中未找到 BloggerId %d", video.BloggerId)
			return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
		}
		return &ProcessError{Msg: "failed to get blogger by id for filtering", SourceID: rawData.Id, Err: err}
	}
	fansCount := blogger.BloggerFansNum

	// 2. 获取点赞数 (Like Count)
	//    此数据流： aweme_id -> source_data 表 (type=summary) -> raw_content -> like_count_str
	summarySourceData, err := p.sourceDataRepo.FindLatestByTypeAndEntityID(ctx, "video_summary_headless", rawData.EntityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logMsg := fmt.Sprintf("数据前置条件不足(trend): source_data 表中未找到 AwemeId %s 对应的 summary 数据", rawData.EntityId)
			// 注意：这里我们选择跳过而不是报错，因为可能summary数据确实还没采集到
			return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, 2, logMsg)
		}
		return &ProcessError{Msg: "failed to find summary source data for filtering", SourceID: rawData.Id, Err: err}
	}
	// 解析 summary 的 JSON 以提取 LikeCountStr
	var summaryPayload FeiguaVideoSummaryData
	if err := json.Unmarshal([]byte(summarySourceData.RawContent), &summaryPayload); err != nil {
		return &ProcessError{Msg: "unmarshal summary source data for filtering failed", SourceID: rawData.Id, Err: err}
	}
	likeCount := utils.ParseUnitStrToInt64(summaryPayload.Data.LikeCountStr)

	// 3. 执行过滤判断
	if likeCount <= 50 && fansCount <= 200 {
		logMsg := fmt.Sprintf("过滤条件触发(trend): 视频点赞数 (%d) 且博主粉丝数 (%d) 均不满足要求。", likeCount, fansCount)
		p.log.Infof(logMsg)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, 2, logMsg)
	}
	// --- 修正结束 ---

	// 4. 解析趋势数据API响应
	var resp FeiguaVideoTrendData
	if err := json.Unmarshal([]byte(rawData.RawContent), &resp); err != nil {
		return &ProcessError{Msg: "unmarshal video trend response failed", SourceID: rawData.Id, Err: err}
	}
	if !resp.Status {
		logMsg := fmt.Sprintf("API(trend)返回错误: Code=%d, Msg=%s", resp.Code, resp.Msg)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
	}
	if len(resp.Data) == 0 {
		_ = p.videoRepo.UpdateTrendTimestamp(ctx, rawData.EntityId)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, 1, "API返回的趋势数据为空")
	}

	// 5. 转换数据并调用 BatchOverwrite
	// 5. 转换数据并调用 BatchOverwrite
	trendsToOverwrite := make([]*data.VideoTrend, 0, len(resp.Data))
	for _, item := range resp.Data {
		dateCodeInt, _ := strconv.Atoi(item.DateCode.String())
		trend := &data.VideoTrend{
			AwemeId:            rawData.EntityId,
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
		trendsToOverwrite = append(trendsToOverwrite, trend)
	}

	if err := p.videoTrendRepo.BatchOverwrite(ctx, trendsToOverwrite); err != nil {
		return &ProcessError{Msg: "batch overwrite video trends failed", SourceID: rawData.Id, Err: err}
	}

	if err := p.videoRepo.UpdateTrendTimestamp(ctx, video.AwemeId); err != nil {
		p.log.Warnf("更新 video trend_updated_at 失败 (VideoID: %s): %v", video.AwemeId, err)
	}

	return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
}

// FeiguaVideoSummaryData ...
type FeiguaVideoSummaryData struct {
	FeiguaBaseResponse
	Data FeiguaVideoSummaryDTO `json:"data"`
}

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

// FeiguaVideoTrendDataItem ...
type FeiguaVideoTrendDataItem struct {
	RecordTime      string `json:"record_time"`
	PlayCountStr    string `json:"play_count_str"`
	LikeCountStr    string `json:"like_count_str"`
	CommentCountStr string `json:"comment_count_str"`
	ShareCountStr   string `json:"share_count_str"`
	CollectCountStr string `json:"collect_count_str"`
}

// FeiguaVideoTrendData ...
type FeiguaVideoTrendData struct {
	FeiguaBaseResponse
	Data []*FeiguaVideoTrendItem `json:"data"`
}

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
