package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"aresdata/internal/fetcher"
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type FetcherUsecase struct {
	repo    data.SourceDataRepo
	fetcher *fetcher.FeiguaFetcher
	log     *log.Helper
}

func NewFetcherUsecase(repo data.SourceDataRepo, fetcher *fetcher.FeiguaFetcher, logger log.Logger) *FetcherUsecase {
	return &FetcherUsecase{
		repo:    repo,
		fetcher: fetcher,
		log:     log.NewHelper(log.With(logger, "module", "usecase/fetcher")),
	}
}

// FetchAndStoreVideoRank 是一个具体的业务方法，负责采集视频榜单并存储
func (uc *FetcherUsecase) FetchAndStoreVideoRank(ctx context.Context, period, datecode string, pageIndex, pageSize int) (*v1.SourceData, error) {
	// 1. 调用 Fetcher 获取原始数据和请求元数据
	rawContent, meta, err := uc.fetcher.FetchVideoRank(ctx, period, datecode, pageIndex, pageSize)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to fetch video rank from feigua: %v", err)
		// 即使请求失败，也尝试记录请求上下文
		if meta != nil {
			failedData := &v1.SourceData{
				ProviderName:   "feigua",
				DataType:       "video_rank_" + period,
				EntityId:       fmt.Sprintf("%s_%s", period, datecode),
				Status:         -1, // 标记为错误
				FetchedAt:      time.Now().Format(time.RFC3339),
				Date:           datecode,
				RawContent:     err.Error(), // 内容字段记录错误信息
				RequestMethod:  meta.Method,
				RequestUrl:     meta.URL,
				RequestParams:  meta.Params,
				RequestHeaders: meta.Headers,
			}
			uc.repo.Save(ctx, failedData) // 忽略这里的错误，因为主流程已经失败
		}
		return nil, err
	}

	uc.log.WithContext(ctx).Infof("Successfully fetched data for period=%s, datecode=%s", period, datecode)

	// 2. 构造 SourceData 对象准备入库
	sourceData := &v1.SourceData{
		ProviderName:   "feigua",
		DataType:       "video_rank_" + period, // 例如: video_rank_day
		RawContent:     rawContent,
		EntityId:       fmt.Sprintf("%s_%s", period, datecode),
		Status:         0, // 初始状态为 未处理
		FetchedAt:      time.Now().Format(time.RFC3339),
		Date:           datecode,
		RequestMethod:  meta.Method,
		RequestUrl:     meta.URL,
		RequestParams:  meta.Params,
		RequestHeaders: meta.Headers,
	}

	// 3. 调用Repo存储到数据库
	return uc.repo.Save(ctx, sourceData)
}

// FetchAndStoreVideoTrend 采集并存储视频趋势数据
func (uc *FetcherUsecase) FetchAndStoreVideoTrend(ctx context.Context, awemeID string, awemePubTime time.Time) (*v1.SourceData, error) {
	// 接收 body 和 meta
	rawContent, meta, err := uc.fetcher.FetchVideoTrend(ctx, awemeID, awemePubTime)
	if err != nil {
		// 即便请求失败，我们也应该记录这次失败的请求，以便排查和重试
		if meta != nil {
			failedSourceData := &v1.SourceData{
				ProviderName:   "feigua",
				DataType:       "video_trend",
				EntityId:       awemeID,
				Status:         -1, // 标记为错误
				FetchedAt:      time.Now().Format(time.DateTime),
				RawContent:     err.Error(), // 内容字段记录错误信息
				RequestMethod:  meta.Method,
				RequestUrl:     meta.URL,
				RequestParams:  meta.Params,
				RequestHeaders: meta.Headers,
			}
			uc.repo.Save(ctx, failedSourceData) // 忽略这里的错误，因为主流程已经失败
		}
		return nil, err
	}

	// 成功则记录完整信息
	sourceData := &v1.SourceData{
		ProviderName:   "feigua",
		DataType:       "video_trend",
		RawContent:     rawContent,
		EntityId:       awemeID,
		Date:           time.Now().Format("20060102"),
		Status:         0,
		FetchedAt:      time.Now().Format(time.DateTime),
		RequestMethod:  meta.Method,
		RequestUrl:     meta.URL,
		RequestParams:  meta.Params,
		RequestHeaders: meta.Headers,
	}

	return uc.repo.Save(ctx, sourceData)
}

// FetchAndStoreVideoSummary 采集并存储视频总览数据
func (uc *FetcherUsecase) FetchAndStoreVideoSummary(ctx context.Context, awemeID, dateCode string) (*v1.SourceData, error) {
	rawContent, meta, err := uc.fetcher.FetchVideoSummary(ctx, awemeID, dateCode)
	dataType := "video_summary" // 定义数据类型
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to fetch video summary for awemeId %s: %v", awemeID, err)
		// 即使请求失败，也尝试记录请求上下文
		if meta != nil {
			failedSourceData := &v1.SourceData{
				ProviderName:   "feigua",
				DataType:       dataType,
				EntityId:       awemeID,
				Status:         -1, // 标记为错误
				FetchedAt:      time.Now().Format(time.RFC3339),
				Date:           dateCode,
				RawContent:     err.Error(), // 内容字段记录错误信息
				RequestMethod:  meta.Method,
				RequestUrl:     meta.URL,
				RequestParams:  meta.Params,
				RequestHeaders: meta.Headers,
			}
			// 尝试保存失败记录，忽略此处的错误因为主流程已经失败
			_, _ = uc.repo.Save(ctx, failedSourceData)
		}
		return nil, err
	}

	uc.log.WithContext(ctx).Infof("Successfully fetched video summary for awemeId=%s", awemeID)

	// 构造 SourceData 对象准备入库
	sourceData := &v1.SourceData{
		ProviderName:   "feigua",
		DataType:       dataType,
		RawContent:     rawContent,
		EntityId:       awemeID,
		Status:         0, // 初始状态为 未处理
		FetchedAt:      time.Now().Format(time.RFC3339),
		Date:           dateCode,
		RequestMethod:  meta.Method,
		RequestUrl:     meta.URL,
		RequestParams:  meta.Params,
		RequestHeaders: meta.Headers,
	}

	// 调用Repo存储到数据库
	return uc.repo.Save(ctx, sourceData)
}
