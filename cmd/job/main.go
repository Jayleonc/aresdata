package main

import (
	"aresdata/internal/conf"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
)

var (
	flagconf string
	mode     string // "cron" or "manual"
	period   string // for manual mode
	datecode string // for manual mode
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
	flag.StringVar(&mode, "mode", "cron", "run mode: cron or manual")
	flag.StringVar(&period, "period", "day", "data period for manual mode (day, week, month)")
	flag.StringVar(&datecode, "datecode", "", "date code for manual mode (e.g., 20250714)")
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(bc.Job, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// 根据 mode 参数决定执行逻辑
	switch mode {
	case "cron":
		runCronMode(app, logger)
	case "manual":
		runManualMode(app, logger)
	default:
		fmt.Printf("Invalid mode: %s. Please use 'cron' or 'manual'.\n", mode)
		os.Exit(1)
	}
}

// runCronMode 作为常驻服务运行
func runCronMode(app *CronService, logger log.Logger) {
	log.NewHelper(logger).Info("Starting in CRON mode...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Stop(ctx)
	log.NewHelper(logger).Info("CRON mode stopped.")
}

// runManualMode 执行一次性任务
func runManualMode(app *CronService, logger log.Logger) {
	log.NewHelper(logger).Info("Starting in MANUAL mode...")

	// 如果未指定datecode，则默认为昨天
	if datecode == "" {
		datecode = time.Now().Add(-24 * time.Hour).Format("20060102")
		log.NewHelper(logger).Infof("datecode not provided, using yesterday: %s", datecode)
	}

	log.NewHelper(logger).Infof("Executing job: FetchAndStoreVideoRank for period=%s, datecode=%s", period, datecode)

	// 直接调用 usecase 执行任务
	if _, err := app.uc.FetchAndStoreVideoRank(context.Background(), period, datecode); err != nil {
		log.NewHelper(logger).Errorf("Manual job failed: %v", err)
		os.Exit(1) // 以非0状态码退出表示失败
	} else {
		log.NewHelper(logger).Infof("Manual job finished successfully.")
	}
}
