package main

import (
	"aresdata/internal/biz"
	"aresdata/internal/conf"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
)

type CronService struct {
	log *log.Helper
	uc  *biz.FetcherUsecase
	c   *cron.Cron
	cfg *conf.Job
}

func NewCronService(uc *biz.FetcherUsecase, cfg *conf.Job, logger log.Logger) *CronService {
	return &CronService{
		log: log.NewHelper(log.With(logger, "module", "cron")),
		uc:  uc,
		c:   cron.New(cron.WithSeconds()),
		cfg: cfg,
	}
}

func (s *CronService) Start(ctx context.Context) {
	s.log.Info("Cron service starting")

	// 注册采集视频日榜任务
	_, err := s.c.AddFunc(s.cfg.FetchVideoRankCron, func() {
		s.log.Info("Executing job: FetchAndStoreVideoRank")
		// 获取昨天的日期
		yesterday := time.Now().Add(-24 * time.Hour).Format("20060102")
		if _, err := s.uc.FetchAndStoreVideoRank(context.Background(), "day", yesterday); err != nil {
			s.log.Errorf("Failed to execute FetchAndStoreVideoRank job: %v", err)
		} else {
			s.log.Info("Successfully executed FetchAndStoreVideoRank job")
		}
	})

	if err != nil {
		s.log.Fatalf("Failed to add cron job: %v", err)
	}

	s.c.Start()
	s.log.Infof("Cron job [FetchAndStoreVideoRank] scheduled with spec: %s", s.cfg.FetchVideoRankCron)
}

func (s *CronService) Stop(ctx context.Context) {
	s.log.Info("Cron service stopping")
	<-s.c.Stop().Done()
}
