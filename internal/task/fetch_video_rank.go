package task

import (
	"aresdata/internal/biz"
	"context"
	"time"
)

type FetchVideoRankTask struct {
	uc *biz.FetcherUsecase
	// 未来可以注入 TaskLogRepo 来记录任务审计日志
}

func NewFetchVideoRankTask(uc *biz.FetcherUsecase) *FetchVideoRankTask {
	return &FetchVideoRankTask{uc: uc}
}

func (t *FetchVideoRankTask) Name() string {
	return FetchVideoRank
}

func (t *FetchVideoRankTask) Run(ctx context.Context, args ...string) error {
	// 默认采集昨天的日榜
	datecode := time.Now().Add(-24 * time.Hour).Format("20060102")
	_, err := t.uc.FetchAndStoreVideoRank(ctx, "day", datecode)
	return err
}
