package fetcher

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/go-kratos/kratos/v2/log"
)

// HeadlessUsecase 是业务流程层（指挥官），负责编排采集和存储
type HeadlessUsecase struct {
	repo            data.SourceDataRepo
	headlessFetcher *HeadlessFetcher
	log             *log.Helper
}

func NewHeadlessUsecase(repo data.SourceDataRepo, headlessFetcher *HeadlessFetcher, logger log.Logger) *HeadlessUsecase {
	return &HeadlessUsecase{
		repo:            repo,
		headlessFetcher: headlessFetcher,
		log:             log.NewHelper(log.With(logger, "module", "usecase.headless")),
	}
}

// FetchAndStoreVideoDetails 负责一个视频详情的完整采集和存储流程
func (uc *HeadlessUsecase) FetchAndStoreVideoDetails(ctx context.Context, video *v1.VideoDTO) error {
	uc.log.Infof("Starting fetch and store details for video: %s", video.AwemeId)

	entryURL := fmt.Sprintf("https://www.douyin.com/video/%s", video.AwemeId)

	// 1. 调用 "士兵" 获取 Summary 数据
	summaryRaw, err := uc.headlessFetcher.CaptureSummary(ctx, entryURL)
	if err != nil {
		uc.log.Errorf("Failed to capture summary for video %s: %v", video.AwemeId, err)
		return err
	}

	// 2. 调用 "士兵" 获取 Trend 数据
	trendRaw, err := uc.headlessFetcher.CaptureTrend(ctx, entryURL)
	if err != nil {
		uc.log.Errorf("Failed to capture trend for video %s: %v", video.AwemeId, err)
		return err
	}

	// 3. 调用 "档案室" 存储 Summary 数据
	if summaryRaw != "" {
		_, err = uc.repo.Save(ctx, &v1.SourceData{
			ProviderName: "headless",
			DataType:     "video_summary",
			RawContent:   summaryRaw,
			EntityId:     video.AwemeId,
			FetchedAt:    time.Now().Format(time.RFC3339),
		})
		if err != nil {
			uc.log.Errorf("Failed to save summary for video %s: %v", video.AwemeId, err)
			// 即使一个失败，也尝试存另一个
		}
	} else {
		uc.log.Warnf("Summary data is empty for video %s", video.AwemeId)
	}

	// 4. 调用 "档案室" 存储 Trend 数据
	if trendRaw != "" {
		_, err = uc.repo.Save(ctx, &v1.SourceData{
			ProviderName: "headless",
			DataType:     "video_trend",
			RawContent:   trendRaw,
			EntityId:     video.AwemeId,
			FetchedAt:    time.Now().Format(time.RFC3339),
		})
		if err != nil {
			uc.log.Errorf("Failed to save trend for video %s: %v", video.AwemeId, err)
		}
	} else {
		uc.log.Warnf("Trend data is empty for video %s", video.AwemeId)
	}

	return nil // 返回 nil 表示流程完成，即使存储有部分失败
}
