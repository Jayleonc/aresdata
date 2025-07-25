//go:build wireinject
// +build wireinject

package main

import (
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/Jayleonc/aresdata/internal/fetcher"
	"github.com/Jayleonc/aresdata/internal/task"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// 依赖注入说明：task 层依赖具体 usecase（*fetcher.HttpUsecase, *fetcher.HeadlessUsecase），FetcherManager 仅用于 fetcher 初始化。
func wireApp(*conf.Bootstrap, *conf.Data, log.Logger) (*App, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		fetcher.ProviderSet,
		task.ProviderSet,
		newApp,
	))
}
