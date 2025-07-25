// internal/task/remedy_video_details_headless.go
package task

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/Jayleonc/aresdata/internal/fetcher"
	"github.com/go-kratos/kratos/v2/log"
)

// RemedyVideoDetailsHeadlessTask 负责修复部分采集失败的视频详情数据
type RemedyVideoDetailsHeadlessTask struct {
	log            *log.Helper
	videoRepo      data.VideoRepo
	fetcherManager *fetcher.FetcherManager
	dataSourceName string
	headlessUC     *fetcher.HeadlessUsecase // 新增，负责采集和存储
}

// NewRemedyVideoDetailsHeadlessTask 创建一个新的修复任务实例
func NewRemedyVideoDetailsHeadlessTask(
	logger log.Logger,
	videoRepo data.VideoRepo,
	fetcherManager *fetcher.FetcherManager,
	headlessUC *fetcher.HeadlessUsecase, // 新增参数
) *RemedyVideoDetailsHeadlessTask {
	return &RemedyVideoDetailsHeadlessTask{
		log:            log.NewHelper(log.With(logger, "module", "task.remedy_video_details_headless")),
		videoRepo:      videoRepo,
		fetcherManager: fetcherManager,
		dataSourceName: "your_headless_data_source_name_here",
		headlessUC:     headlessUC, // 注入 usecase
	}
}

// Name 返回任务的名称
func (t *RemedyVideoDetailsHeadlessTask) Name() string {
	return RemedyVideoDetailsHeadless
}

// Run 执行任务的核心逻辑
func (t *RemedyVideoDetailsHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [无头浏览器-视频详情修复] 任务...")

	// 1. 获取采集器实例
	// 对于修复任务，我们通常针对一个主要的数据源进行
	const primaryDatasource = "feigua_headless_primary"
	const (
		hoursAgo = 24
		limit    = 10
	)
	videos, err := t.videoRepo.FindPartiallyCollectedVideos(ctx, hoursAgo, limit)
	if err != nil {
		t.log.Errorf("Failed to find partially collected videos: %v", err)
		return err
	}

	for _, video := range videos {
		// 将 data.VideoForCollection 转换为 v1.VideoDTO
		videoDTO := &v1.VideoDTO{
			AwemeId:        video.AwemeId,
			AwemePubTime:   video.AwemePubTime.Format("2006-01-02 15:04:05"),
			AwemeDetailUrl: video.AwemeDetailUrl,
		}

		if err := t.headlessUC.FetchAndStoreVideoDetails(ctx, videoDTO); err != nil {
			t.log.Errorf("Failed to fetch and store video details for video %s from source %s: %v", video.AwemeId, t.dataSourceName, err)
		} else {
			t.log.Infof("Successfully remedied video details for video %s from source %s", video.AwemeId, t.dataSourceName)
		}
	}

	t.log.Info("[无头浏览器-视频详情修复] 所有修复任务已处理完毕。")
	return nil
}

// newChromedpLogger 创建一个日志函数，该函数会过滤掉特定于chromedp的、已知的无害错误日志。
func (t *RemedyVideoDetailsHeadlessTask) newChromedpLogger() func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if strings.Contains(msg, "initiatorIPAddressSpace") {
			return
		}
		t.log.Info(msg)
	}
}
