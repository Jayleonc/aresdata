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
		t.log.WithContext(ctx).Errorf("获取需要追踪的 aweme_id 列表失败: %v", err)
		return err
	}
	t.log.WithContext(ctx).Infof("开始采集 %d 个视频的趋势数据", len(awemeIDs))

	// 2. 依次采集并存储趋势
	for _, id := range awemeIDs {
		_, err := t.fetcherUC.FetchAndStoreVideoTrend(ctx, id)
		if err != nil {
			t.log.WithContext(ctx).Errorf("采集并入库 awemeId %s 趋势数据失败: %v", id, err)
		} else {
			t.log.WithContext(ctx).Infof("已成功下发采集趋势任务，awemeId: %s", id)
		}
		time.Sleep(1 * time.Second)
	}

	t.log.WithContext(ctx).Info("全部视频趋势采集任务已下发完毕。")
	return nil
}
