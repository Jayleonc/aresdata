//go:build wireinject
// +build wireinject

package main

import (
	"aresdata/internal/biz"
	"aresdata/internal/conf"
	"aresdata/internal/data"
	"aresdata/internal/etl"
	"aresdata/internal/task"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Bootstrap, *conf.Data, log.Logger) (*App, func(), error) {
	panic(wire.Build(
		data.ProviderSet, // 提供 Repos
		biz.ProviderSet,  // 提供 Usecases，它依赖 Repos
		etl.ProviderSet,
		task.ProviderSet, // 提供 Tasks，它依赖 Usecases
		newApp,
	))
}
