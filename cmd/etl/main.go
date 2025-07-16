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
	dataType string // for manual mode: which data_type to process
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
	flag.StringVar(&mode, "mode", "cron", "run mode: cron or manual")
	flag.StringVar(&dataType, "data_type", "", "data type to process in manual mode (e.g., video_rank_day)")
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

	etl, cleanup, err := wireApp(bc.Job, bc.Data, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	switch mode {
	case "cron":
		runCronMode(etl, logger)
	case "manual":
		runManualMode(etl, logger)
	default:
		fmt.Printf("Invalid mode: %s. Please use 'cron' or 'manual'.\n", mode)
		os.Exit(1)
	}
}

// runCronMode runs as a long-running service
func runCronMode(etl ETLRunner, logger log.Logger) {
	log.NewHelper(logger).Info("ETL Starting in CRON mode...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			// 第二个参数需要处理
			if err := etl.Run(ctx, ""); err != nil {
				log.NewHelper(logger).Errorf("ETL Run error: %v", err)
			}
			time.Sleep(60 * time.Second) // 可根据实际需求调整调度周期
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.NewHelper(logger).Info("CRON mode stopped.")
}

// runManualMode runs a single ETL task
func runManualMode(etl ETLRunner, logger log.Logger) {
	log.NewHelper(logger).Info("ETL Starting in MANUAL mode...")
	if dataType == "" {
		log.NewHelper(logger).Info("No data_type specified, will process all unprocessed types.")
	}
	if err := etl.Run(context.Background(), dataType); err != nil {
		log.NewHelper(logger).Errorf("Manual ETL failed: %v", err)
		os.Exit(1)
	} else {
		log.NewHelper(logger).Infof("Manual ETL finished successfully.")
	}
}

type ETLRunner interface {
	Run(ctx context.Context, dataType string) error
}
