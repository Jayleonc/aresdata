package main

import (
	"context"
	"flag"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/Jayleonc/aresdata/internal/
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
	"os"
)

var (
	flagconf string
	taskName string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path")
	flag.StringVar(&taskName, "task", "", "run a single task manually by its name")
}

// App 结构体聚合了所有依赖
type App struct {
	logger log.Logger
	tasks  map[string]task.Task
	cron   *cron.Cron
}

func newApp(logger log.Logger, tasks []task.Task) *App {
	app := &App{
		logger: logger,
		tasks:  make(map[string]task.Task),
		cron:   cron.New(cron.WithSeconds()),
	}
	for _, t := range tasks {
		app.tasks[t.Name()] = t
	}
	return app
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout), "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)
	c := config.New(config.WithSource(file.NewSource(flagconf)))
	defer c.Close()
	if err := c.Load(); err != nil {
		panic(err)
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(&bc, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if taskName != "" {
		log.NewHelper(logger).Infof("Running single task manually: %s", taskName)
		if t, ok := app.tasks[taskName]; ok {
			if err := t.Run(context.Background()); err != nil {
				log.NewHelper(logger).Errorf("Task %s failed: %v", taskName, err)
			}
		} else {
			log.NewHelper(logger).Errorf("Task %s not found", taskName)
		}
	} else {
		log.NewHelper(logger).Info("Starting cron scheduler mode...")
		app.cron.AddFunc("0 0 2 * * *", func() {
			log.NewHelper(logger).Info("Cron triggered for task: fetch:video_rank")
			fetchTask := app.tasks[task.FetchVideoRank]
			if err := fetchTask.Run(context.Background()); err == nil {
				log.NewHelper(logger).Info("Fetch task succeeded, triggering ETL task: etl:video_rank")
				etlTask := app.tasks[task.ProcessVideoRank]
				if err_etl := etlTask.Run(context.Background()); err_etl != nil {
					log.NewHelper(logger).Errorf("ETL task %s failed: %v", etlTask.Name(), err_etl)
				}
			} else {
				log.NewHelper(logger).Errorf("Fetch task %s failed: %v", fetchTask.Name(), err)
			}
		})
		app.cron.AddFunc("0 0 3 * * *", func() {
			log.NewHelper(logger).Info("Cron triggered for task: fetch:video_trend")
			fetchTask := app.tasks[task.FetchVideoTrend]
			if err := fetchTask.Run(context.Background()); err == nil {
				log.NewHelper(logger).Info("Fetch trend task succeeded, triggering ETL task: etl:video_trend")
				etlTask := app.tasks[task.ProcessVideoTrend]
				if err_etl := etlTask.Run(context.Background()); err_etl != nil {
					log.NewHelper(logger).Errorf("ETL task %s failed: %v", etlTask.Name(), err_etl)
				}
			} else {
				log.NewHelper(logger).Errorf("Fetch task %s failed: %v", fetchTask.Name(), err)
			}
		})

		// 每天4点执行，采集和处理视频总览数据
		app.cron.AddFunc("0 0 4 * * *", func() {
			log.NewHelper(logger).Info("Cron triggered for task: fetch:video_summary")
			fetchTask := app.tasks[task.FetchVideoSummary]
			if err := fetchTask.Run(context.Background()); err == nil {
				log.NewHelper(logger).Info("Fetch summary task succeeded, triggering ETL task: process:video_summary")
				etlTask := app.tasks[task.ProcessVideoSummary]
				if err_etl := etlTask.Run(context.Background()); err_etl != nil {
					log.NewHelper(logger).Errorf("ETL task %s failed: %v", etlTask.Name(), err_etl)
				}
			} else {
				log.NewHelper(logger).Errorf("Fetch task %s failed: %v", fetchTask.Name(), err)
			}
		})
		app.cron.Start()
		select {}
	}
}
