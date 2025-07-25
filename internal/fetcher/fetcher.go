package fetcher

import (
	"context"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// RequestMetadata 存储用于日志和调试的API请求元数据。
type RequestMetadata struct {
	Method  string
	URL     string
	Params  string // 参数的JSON字符串
	Headers string // 请求头的JSON字符串
}

// Fetcher 是所有数据源采集“士兵”的统一接口。
type Fetcher interface {
	// GetConfig 返回该采集器的配置。
	GetConfig() *conf.DataSource

	// FetchVideoRank 获取视频榜单数据，主要由HTTP采集器使用。
	FetchVideoRank(ctx context.Context, period, datecode string, pageIndex, pageSize int) (string, *RequestMetadata, error)

	// CaptureVideoDetails 获取视频详情数据，主要由Headless采集器使用。
	// 返回值分别为：(摘要接口原始数据, 趋势接口原始数据, 错误)
	CaptureVideoDetails(ctx context.Context, entryURL string) (string, string, error)

	FetchVideoSummary(ctx context.Context, awemeID, dateCode string) (string, *RequestMetadata, error)
}

func NewHeadlessAccountPool(cfg *conf.DataSource, logger log.Logger) *AccountPool {
	// TODO: Replace with actual logic to get cookie paths from cfg and instantiate AccountPool
	cookiePaths := cfg.AccountPool // or whatever field holds paths
	pool, _ := NewAccountPool(cookiePaths, logger)
	return pool
}

// ProviderSet 是 fetcher 的依赖注入集合。
var ProviderSet = wire.NewSet(
	provideDataSources,
	NewFetcherManager,
	NewHttpUsecase,
	NewHeadlessUsecase,
	NewHeadlessAccountPool,
	NewHeadlessFetcher,
)

func provideDataSources(c *conf.Data) []*conf.DataSource {
	return c.GetDatasources()
}
