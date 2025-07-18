package task

import (
	"aresdata/internal/etl"
	"context"
)

type ProcessVideoSummaryTask struct {
	etl *etl.ETLUsecase
}

func NewProcessVideoSummaryTask(etl *etl.ETLUsecase) *ProcessVideoSummaryTask {
	return &ProcessVideoSummaryTask{etl: etl}
}

func (t *ProcessVideoSummaryTask) Name() string {
	return ProcessVideoSummary
}

func (t *ProcessVideoSummaryTask) Run(ctx context.Context, args ...string) error {
	// 调用通用的ETL Usecase，只处理 video_summary 类型的数据
	return t.etl.RunWithType(ctx, "video_summary")
}
