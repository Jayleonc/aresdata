package biz

import (
	"aresdata/pkg/fetcher"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewFetcherUsecase, fetcher.NewFeiguaFetcher)
