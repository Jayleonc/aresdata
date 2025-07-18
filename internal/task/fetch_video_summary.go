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
		t.log.WithContext(ctx).Errorf("failed to get videos needing summary update: %v", err)
		return err
	}

	if len(videos) == 0 {
		t.log.WithContext(ctx).Info("No videos need summary update.")
		return nil
	}

	t.log.WithContext(ctx).Infof("Start to fetch summary for %d videos", len(videos))

	for _, v := range videos {
		dateCode := v.AwemePubTime.Format("20060102")
		_, err := t.fetcherUC.FetchAndStoreVideoSummary(ctx, v.AwemeId, dateCode)
		if err != nil {
			t.log.WithContext(ctx).Errorf("failed to fetch and store summary for awemeId %s: %v", v.AwemeId, err)
		} else {
			t.log.WithContext(ctx).Infof("Successfully dispatched summary fetch for awemeId: %s", v.AwemeId)
		}
		time.Sleep(1 * time.Second)
	}

	t.log.WithContext(ctx).Info("All video summary fetching tasks have been dispatched.")
	return nil
}
