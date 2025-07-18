package task

import (
	"aresdata/internal/biz"
	"aresdata/internal/data"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type FetchVideoSummaryTask struct {
	fetcherUC *biz.FetcherUsecase
	videoRepo data.VideoRepo
	log       *log.Helper
}

func NewFetchVideoSummaryTask(fetcherUC *biz.FetcherUsecase, videoRepo data.VideoRepo, logger log.Logger) *FetchVideoSummaryTask {
	return &FetchVideoSummaryTask{
		fetcherUC: fetcherUC,
		videoRepo: videoRepo,
		log:       log.NewHelper(log.With(logger, "module", "task/fetch-video-summary")),
	}
}

func (t *FetchVideoSummaryTask) Name() string {
	return FetchVideoSummary
}

func (t *FetchVideoSummaryTask) Run(ctx context.Context, args ...string) error {
	videos, err := t.videoRepo.FindVideosNeedingSummaryUpdate(ctx, 100)
	if err != nil {
		t.log.WithContext(ctx).Errorf("获取待更新视频列表失败: %v", err)
		return err
	}

	if len(videos) == 0 {
		t.log.WithContext(ctx).Info("没有需要更新总览的视频。")
		return nil
	}

	t.log.WithContext(ctx).Infof("开始采集 %d 个视频的总览数据", len(videos))

	for _, v := range videos {
		dateCode := v.AwemePubTime.Format("20060102")
		_, err := t.fetcherUC.FetchAndStoreVideoSummary(ctx, v.AwemeId, dateCode)
		if err != nil {
			t.log.WithContext(ctx).Errorf("采集并入库 awemeId %s 总览数据失败: %v", v.AwemeId, err)
		} else {
			t.log.WithContext(ctx).Infof("已成功下发采集任务，awemeId: %s", v.AwemeId)
		}
		time.Sleep(1 * time.Second)
	}

	t.log.WithContext(ctx).Info("全部视频总览采集任务已下发完毕。")
	return nil
}
