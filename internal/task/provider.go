package task

import (
	"github.com/Jayleonc/aresdata/internal/fetcher"
)

// HeadlessTaskProvider 是一个依赖集合，专门为所有使用无头浏览器的 Task 提供服务。
// 它将零散的依赖项组合成一个有业务含义的整体。
type HeadlessTaskProvider struct {
	FetcherManager *fetcher.FetcherManager
	HeadlessUC     *fetcher.HeadlessUsecase
}

// NewHeadlessTaskProvider 创建一个新的 Provider 实例。
// 这个函数本身也是一个 Provider，会被 Wire 调用。
func NewHeadlessTaskProvider(fm *fetcher.FetcherManager, uc *fetcher.HeadlessUsecase) *HeadlessTaskProvider {
	return &HeadlessTaskProvider{
		FetcherManager: fm,
		HeadlessUC:     uc,
	}
}

// HttpTaskProvider 专门为 HTTP 任务服务
// 如果未来有其他HTTP任务共用的依赖，都加在这里
type HttpTaskProvider struct {
	HttpUC *fetcher.HttpUsecase
}

func NewHttpTaskProvider(uc *fetcher.HttpUsecase) *HttpTaskProvider {
	return &HttpTaskProvider{HttpUC: uc}
}
