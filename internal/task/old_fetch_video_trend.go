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
	limit := 10 // 每次任务最多处理100个视频

	// 使用字符串数组代替结构体数组
	//videos := [][]string{
	//	{"7526922765706857767", "20250714"},
	//	{"7527338167700065563", "20250715"},
	//	{"7528437837746769179", "20250718"},
	//	{"7527219145831451938", "20250715"},
	//	{"7528372296273169679", "20250718"},
	//	{"7524784460735302946", "20250709"},
	//	{"7524567237513661754", "20250708"},
	//	{"7524322058580118842", "20250707"},
	//	{"7525294429016427791", "20250710"},
	//	{"7527215022746307875", "20250715"},
	//}

	// 1. 通过 VideoUsecase 获取需要更新趋势的视频列表
	videos, err := t.videoUC.GetVideosByTimeWindow(ctx, limit)
	if err != nil {
		t.log.Errorf("获取待更新趋势视频列表失败: %v", err)
		return err
	}

	if len(videos) == 0 {
		t.log.Info("没有需要更新趋势的视频。")
		return nil
	}

	for _, v := range videos {
		awemeID := v.AwemeId
		awemePubTime := v.AwemePubTime
		_, err := t.fetcherUC.FetchAndStoreVideoTrend(ctx, awemeID, awemePubTime)
		if err != nil {
			t.log.Errorf("为视频 %s 下发趋势采集任务失败: %v", awemeID, err)
		} else {
			t.log.Infof("已成功为视频 %s 下发趋势采集任务", awemeID)
		}
	}

	t.log.Info("[每日视频趋势] 所有采集任务已下发完毕。")
	return nil
}
