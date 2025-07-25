package task

import (
	"context"
	"github.com/Jayleonc/aresdata/internal/etl"
)

type ProcessVideoTrendTask struct {
	etl *etl.ETLUsecase
}

func NewProcessVideoTrendTask(etl *etl.ETLUsecase) *ProcessVideoTrendTask {
	return &ProcessVideoTrendTask{etl: etl}
}

func (t *ProcessVideoTrendTask) Name() string {
	return ProcessVideoTrend
}

func (t *ProcessVideoTrendTask) Run(ctx context.Context, args ...string) error {
	// 调用通用的ETL Usecase，只处理 video_trend 类型的数据
	return t.etl.RunWithType(ctx, "video_trend")
}
