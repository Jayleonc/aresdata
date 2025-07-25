package task

import (
	"github.com/google/wire"
)

// ProviderSet is task providers.
// 我们在这里将所有独立的 Task 构造函数和集合函数打包在一起
var ProviderSet = wire.NewSet(
	NewTaskSet,
	NewFetchVideoRankTask,
	NewFetchVideoTrendTask,
	NewFetchVideoDetailsHeadlessTask,
	NewRemedyVideoDetailsHeadlessTask,
)

// NewTaskSet 负责将所有具体的任务实例聚合为一个 []Task 切片
func NewTaskSet(
	p1 *FetchVideoRankTask,
	p3 *FetchVideoTrendTask,
	p7 *FetchVideoDetailsHeadlessTask,
	p8 *RemedyVideoDetailsHeadlessTask,
) []Task {
	return []Task{p1, p3, p7, p8}
}
