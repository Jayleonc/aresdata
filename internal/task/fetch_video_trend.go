package task

import (
	"aresdata/internal/biz"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type FetchVideoTrendTask struct {
	fetcherUC *biz.FetcherUsecase
	rankUC    *biz.VideoRankUsecase // 依赖 VideoRankUsecase 获取ID
	log       *log.Helper
}

// NewFetchVideoTrendTask 构造函数现在注入两个 Usecase
func NewFetchVideoTrendTask(fetcherUC *biz.FetcherUsecase, rankUC *biz.VideoRankUsecase, logger log.Logger) *FetchVideoTrendTask {
	return &FetchVideoTrendTask{
		fetcherUC: fetcherUC,
		rankUC:    rankUC,
		log:       log.NewHelper(log.With(logger, "module", "task/fetch-video-trend")),
	}
}

func (t *FetchVideoTrendTask) Name() string {
	return FetchVideoTrend
}

func (t *FetchVideoTrendTask) Run(ctx context.Context, args ...string) error {
	// 1. 通过业务层获取需要追踪的 aweme_id 列表 (查询逻辑已内聚到 biz 和 data 层)
	awemeIDs, err := t.rankUC.GetTrackedAwemeIDs(ctx, 7) // 查询近7天
	if err != nil {
		t.log.WithContext(ctx).Errorf("failed to get tracked aweme ids: %v", err)
		return err
	}
	t.log.WithContext(ctx).Infof("Start to fetch trends for %d videos", len(awemeIDs))

	// 2. 依次采集并存储趋势
	for _, id := range awemeIDs {
		_, err := t.fetcherUC.FetchAndStoreVideoTrend(ctx, id)
		if err != nil {
			t.log.WithContext(ctx).Errorf("failed to fetch and store trend for awemeId %s: %v", id, err)
		} else {
			t.log.WithContext(ctx).Infof("Successfully dispatched trend fetch for awemeId: %s", id)
		}
		time.Sleep(1 * time.Second)
	}

	t.log.WithContext(ctx).Info("All trend fetching tasks have been dispatched.")
	return nil
}
