package biz

import (
	"aresdata/internal/fetcher"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewVideoRankUsecase,
	NewFetcherUsecase,
	fetcher.NewFeiguaFetcher,
)
