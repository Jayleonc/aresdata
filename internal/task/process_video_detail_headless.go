// internal/task/process_video_detail_headless.go

package task

import (
	"context"
	"github.com/Jayleonc/aresdata/internal/etl"
	"github.com/go-kratos/kratos/v2/log"
)

// ProcessVideoDetailHeadlessTask 负责处理所有由无头浏览器采集的详情数据
type ProcessVideoDetailHeadlessTask struct {
	etl *etl.ETLUsecase
	log *log.Helper
}

// NewProcessVideoDetailHeadlessTask .
func NewProcessVideoDetailHeadlessTask(etl *etl.ETLUsecase, logger log.Logger) *ProcessVideoDetailHeadlessTask {
	return &ProcessVideoDetailHeadlessTask{
		etl: etl,
		log: log.NewHelper(log.With(logger, "module", "task.process_video_detail_headless")),
	}
}

func (t *ProcessVideoDetailHeadlessTask) Name() string {
	return ProcessVideoDetailHeadless
}

func (t *ProcessVideoDetailHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.WithContext(ctx).Info("开始执行 [ETL-无头浏览器视频详情] 任务...")

	// 1. 首先处理 video_summary_headless 类型的数据
	t.log.WithContext(ctx).Info("正在处理 video_summary_headless 数据...")
	if err := t.etl.RunWithType(ctx, "video_summary_headless"); err != nil {
		// 记录错误，但继续执行下一个任务，确保数据处理的完整性
		t.log.WithContext(ctx).Errorf("处理 video_summary_headless 数据失败: %v", err)
	} else {
		t.log.WithContext(ctx).Info("video_summary_headless 数据处理完成。")
	}

	// 2. 接着处理 video_trend_headless 类型的数据
	t.log.WithContext(ctx).Info("正在处理 video_trend_headless 数据...")
	if err := t.etl.RunWithType(ctx, "video_trend_headless"); err != nil {
		t.log.WithContext(ctx).Errorf("处理 video_trend_headless 数据失败: %v", err)
		return err // 如果趋势处理也失败，则返回错误
	} else {
		t.log.WithContext(ctx).Info("video_trend_headless 数据处理完成。")
	}

	t.log.WithContext(ctx).Info("[ETL-无头浏览器视频详情] 任务执行完毕。")
	return nil
}
