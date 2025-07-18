package task

import (
	"aresdata/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type FetchVideoRankTask struct {
	uc  *biz.FetcherUsecase
	log *log.Helper // 正确的日志字段类型
	// 未来可以注入 TaskLogRepo 来记录任务审计日志
}

func NewFetchVideoRankTask(uc *biz.FetcherUsecase, logger log.Logger) *FetchVideoRankTask {
	return &FetchVideoRankTask{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "task/fetch-video-rank")),
	}
}

func (t *FetchVideoRankTask) Name() string {
	return FetchVideoRank
}

func (t *FetchVideoRankTask) Run(ctx context.Context, args ...string) error {
	// 默认采集昨天的日榜
	datecode := time.Now().Add(-24 * time.Hour).Format("20060102")
	t.log.WithContext(ctx).Infof("开始采集日榜数据，日期: %s", datecode)
	_, err := t.uc.FetchAndStoreVideoRank(ctx, "day", datecode)
	if err != nil {
		t.log.WithContext(ctx).Errorf("采集日榜数据失败，日期: %s，错误: %v", datecode, err)
	} else {
		t.log.WithContext(ctx).Infof("日榜数据采集成功，日期: %s", datecode)
	}
	return err
}
