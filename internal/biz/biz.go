package biz

import (
	"aresdata/internal/fetcher"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewVideoRankUsecase,
	NewFetcherUsecase,
	NewVideoUsecase,
	NewProductUsecase,
	NewBloggerUsecase,
	NewVideoTrendUsecase, // 新增此行
	fetcher.NewFeiguaFetcher,
	NewHeadlessUsecase,
	fetcher.NewHeadlessFetcher,
)
