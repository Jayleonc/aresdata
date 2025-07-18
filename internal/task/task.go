package task

import "github.com/google/wire"

// ProviderSet is task providers.
// 我们在这里将所有独立的 Task 构造函数和集合函数打包在一起
var ProviderSet = wire.NewSet(
	NewFetchVideoRankTask,
	NewProcessVideoRankTask,
	NewFetchVideoTrendTask,
	NewProcessVideoTrendTask,
	NewFetchVideoSummaryTask,
	NewProcessVideoSummaryTask,
	// 这个函数是关键，它告诉 wire 如何构建 []Task
	NewTaskSet,
)

// NewTaskSet 负责将所有具体的任务实例聚合为一个 []Task 切片
func NewTaskSet(
	p1 *FetchVideoRankTask,
	p2 *ProcessVideoRankTask,
	p3 *FetchVideoTrendTask,
	p4 *ProcessVideoTrendTask,
	p5 *FetchVideoSummaryTask,
	p6 *ProcessVideoSummaryTask,
) []Task {
	return []Task{p1, p2, p3, p4, p5, p6}
}
