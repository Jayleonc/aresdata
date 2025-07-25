package task

import (
	"context"
	"github.com/Jayleonc/aresdata/internal/fetcher"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type FetchVideoRankTask struct {
	uc  *fetcher.HttpUsecase
	log *log.Helper // 正确的日志字段类型
	// 未来可以注入 TaskLogRepo 来记录任务审计日志
}

func NewFetchVideoRankTask(uc *fetcher.HttpUsecase, logger log.Logger) *FetchVideoRankTask {
	return &FetchVideoRankTask{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "task/fetch-video-rank")),
	}
}

func (t *FetchVideoRankTask) Name() string {
	return FetchVideoRank
}

func (t *FetchVideoRankTask) Run(ctx context.Context, args ...string) error {
	// 默认采集前天的数据，例如今天是22号，则采集20号的数据
	datecode := time.Now().AddDate(0, 0, -2).Format("20060102")
	t.log.WithContext(ctx).Infof("开始采集日榜数据，日期: %s", datecode)
	// 定义分批次采集的参数
	totalBatches := 10 // 采集20次
	pageSize := 50     // 每次50条，最多只能拿50条，写100也是50条
	var finalErr error

	for i := 1; i <= totalBatches; i++ {
		pageIndex := i
		t.log.WithContext(ctx).Infof("正在采集第 %d/%d 批数据...", pageIndex, totalBatches)

		_, err := t.uc.FetchAndStoreVideoRank(ctx, "day", datecode, pageIndex, pageSize)
		if err != nil {
			t.log.WithContext(ctx).Errorf("采集日榜数据失败，日期: %s，批次: %d，错误: %v", datecode, pageIndex, err)
			finalErr = err // 记录遇到的最后一个错误
		} else {
			t.log.WithContext(ctx).Infof("第 %d/%d 批数据采集任务已成功下发", pageIndex, totalBatches)
		}

		// 加上适当的延时，防止请求过于频繁
		time.Sleep(2 * time.Second)
	}

	if finalErr != nil {
		t.log.WithContext(ctx).Errorf("分批次采集任务完成，但过程中存在错误。")
		return finalErr
	}

	t.log.WithContext(ctx).Infof("所有批次日榜数据采集任务均已成功下发，日期: %s", datecode)
	return nil
}
