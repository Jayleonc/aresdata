package task

import (
	"aresdata/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

// FetchVideoTrendHeadlessTask 负责使用无头浏览器拉取视频趋势
type FetchVideoTrendHeadlessTask struct {
	headlessUC *biz.HeadlessUsecase
	videoUC    *biz.VideoUsecase
	log        *log.Helper
}

// NewFetchVideoTrendHeadlessTask 构造任务实例
func NewFetchVideoTrendHeadlessTask(huc *biz.HeadlessUsecase, vuc *biz.VideoUsecase, logger log.Logger) *FetchVideoTrendHeadlessTask {
	return &FetchVideoTrendHeadlessTask{
		headlessUC: huc,
		videoUC:    vuc,
		log:        log.NewHelper(log.With(logger, "module", "task/fetch-video-trend-headless")),
	}
}

func (t *FetchVideoTrendHeadlessTask) Name() string {
	return FetchVideoTrendHeadless
}

func (t *FetchVideoTrendHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [无头浏览器-视频趋势] 拉取任务...")
	limit := 10 // 每次处理10个视频，因为无头浏览器较慢

	videos, err := t.videoUC.GetVideosNeedingTrendUpdate(ctx, limit)
	if err != nil {
		t.log.Errorf("获取待更新趋势视频列表失败: %v", err)
		return err
	}

	if len(videos) == 0 {
		t.log.Info("没有需要更新趋势的视频。")
		return nil
	}

	for _, v := range videos {
		_, err := t.headlessUC.FetchAndStoreVideoTrend(ctx, v.AwemeId, v.AwemePubTime.Format("20060102"))
		if err != nil {
			t.log.Errorf("为视频 %s 下发无头浏览器采集任务失败: %v", v.AwemeId, err)
		} else {
			t.log.Infof("已成功为视频 %s 下发无头浏览器采集任务", v.AwemeId)
		}
	}

	t.log.Info("[无头浏览器-视频趋势] 所有采集任务已下发完毕。")
	return nil
}
