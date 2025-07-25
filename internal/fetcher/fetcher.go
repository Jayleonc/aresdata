package fetcher

import (
	"context"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/google/wire"
)

// Fetcher is the interface for all data source fetchers.
type Fetcher interface {
	GetConfig() *conf.DataSource
	// 只负责采集 Summary 原始数据
	CaptureSummary(ctx context.Context, video *v1.VideoDTO) (string, error)
	// 只负责采集 Trend 原始数据
	CaptureTrend(ctx context.Context, video *v1.VideoDTO) (string, error)
}

// ProviderSet is fetcher providers.
var ProviderSet = wire.NewSet(
	NewFetcherManager,
)

// 注意：Usecase 构造函数不应在此 ProviderSet 注册
