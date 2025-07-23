package biz

import (
	"context"
	"time"

	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"aresdata/internal/fetcher"

	"github.com/go-kratos/kratos/v2/log"
)

// HeadlessUsecase 封装了使用无头浏览器进行采集的业务逻辑
type HeadlessUsecase struct {
	repo            data.SourceDataRepo
	headlessFetcher *fetcher.HeadlessFetcher
	log             *log.Helper
}

// NewHeadlessUsecase 创建 HeadlessUsecase
func NewHeadlessUsecase(repo data.SourceDataRepo, hf *fetcher.HeadlessFetcher, logger log.Logger) *HeadlessUsecase {
	return &HeadlessUsecase{
		repo:            repo,
		headlessFetcher: hf,
		log:             log.NewHelper(log.With(logger, "module", "usecase/headless")),
	}
}

// FetchAndStoreVideoTrend 使用无头浏览器采集并存储视频趋势数据
func (uc *HeadlessUsecase) FetchAndStoreVideoTrend(ctx context.Context, awemeID, dateCode string) (*v1.SourceData, error) {
	rawContent, meta, err := uc.headlessFetcher.FetchTrend(ctx, awemeID, dateCode)
	dataType := "video_trend_headless" // 使用新的数据类型以作区分

	if err != nil {
		uc.log.Errorf("使用无头浏览器采集趋势失败 (awemeID: %s): %v", awemeID, err)
		failedSourceData := &v1.SourceData{
			ProviderName: "headless_chrome",
			DataType:     dataType,
			EntityId:     awemeID,
			Date:         dateCode,
			Status:       -1, // 标记为错误
			FetchedAt:    time.Now().Format(time.RFC3339),
			RawContent:   err.Error(),
		}
		_, _ = uc.repo.Save(ctx, failedSourceData)
		return nil, err
	}

	sourceData := &v1.SourceData{
		ProviderName:  "headless_chrome",
		DataType:      dataType,
		RawContent:    rawContent,
		Date:          dateCode,
		EntityId:      awemeID,
		Status:        0, // 初始状态为 未处理
		FetchedAt:     time.Now().Format(time.RFC3339),
		RequestMethod: meta.Method,
		RequestUrl:    meta.URL,
		RequestParams: meta.Params,
	}

	return uc.repo.Save(ctx, sourceData)
}
