package task

import (
	"aresdata/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

// FetchVideoTrendTask 负责每日拉取视频趋势数据的任务
type FetchVideoTrendTask struct {
	fetcherUC *biz.FetcherUsecase
	videoUC   *biz.VideoUsecase // 依赖 VideoUsecase
	log       *log.Helper
}

// NewFetchVideoTrendTask 构造任务实例
func NewFetchVideoTrendTask(fetcherUC *biz.FetcherUsecase, videoUC *biz.VideoUsecase, logger log.Logger) *FetchVideoTrendTask {
	return &FetchVideoTrendTask{
		fetcherUC: fetcherUC,
		videoUC:   videoUC,
		log:       log.NewHelper(log.With(logger, "module", "task/fetch-video-trend")),
	}
}

func (t *FetchVideoTrendTask) Name() string {
	return FetchVideoTrend // 修正：返回正确的任务名称
}

// Run 任务的执行入口
func (t *FetchVideoTrendTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [每日视频趋势] 拉取任务...")
	limit := 150 // 每次任务最多处理100个视频

	// 1. 通过 VideoUsecase 获取需要更新趋势的视频列表
	videos, err := t.videoUC.GetVideosNeedingTrendUpdate(ctx, limit)
	if err != nil {
		t.log.Errorf("获取待更新趋势视频列表失败: %v", err)
		return err
	}

	if len(videos) == 0 {
		t.log.Info("没有需要更新趋势的视频。")
		return nil
	}

	t.log.Infof("发现 %d 个视频需要同步趋势数据", len(videos))

	// 2. 遍历ID，通过 FetcherUsecase 下发采集任务
	for _, v := range videos {
		// 修正：将视频的发布时间传递给 Usecase
		_, err := t.fetcherUC.FetchAndStoreVideoTrend(ctx, v.AwemeId, v.AwemePubTime)
		if err != nil {
			t.log.Errorf("为视频 %s 下发趋势采集任务失败: %v", v.AwemeId, err)
		} else {
			t.log.Infof("已成功为视频 %s 下发趋势采集任务", v.AwemeId)
		}
	}

	t.log.Info("[每日视频趋势] 所有采集任务已下发完毕。")
	return nil
}
