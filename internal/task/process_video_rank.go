package task

import (
	"context"
	"github.com/Jayleonc/aresdata/internal/etl"
)

type ProcessVideoRankTask struct {
	etl *etl.ETLUsecase
}

func NewProcessVideoRankTask(etl *etl.ETLUsecase) *ProcessVideoRankTask {
	return &ProcessVideoRankTask{etl: etl}
}

func (t *ProcessVideoRankTask) Name() string {
	return ProcessVideoRank
}

func (t *ProcessVideoRankTask) Run(ctx context.Context, args ...string) error {
	// 处理所有 dataType 为 video_rank_day 的未处理数据
	return t.etl.RunWithType(ctx, "video_rank_day")
}
