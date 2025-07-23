package task

import (
	"context"
)

const (
	FetchVideoRank          = "fetch:video_rank"
	ProcessVideoRank        = "process:video_rank"
	FetchVideoTrend         = "fetch:video_trend"
	ProcessVideoTrend       = "process:video_trend"
	FetchVideoSummary       = "fetch:video_summary"
	ProcessVideoSummary     = "process:video_summary"
	FetchVideoTrendHeadless = "fetch:video_trend_headless"
)

// Task 定义了所有可执行任务的标准接口
type Task interface {
	// Name 返回任务的唯一名称，用于注册和调用
	Name() string
	// Run 执行任务的具体逻辑
	// args 用于接收来自调度器或其他任务的动态参数
	Run(ctx context.Context, args ...string) error
}
