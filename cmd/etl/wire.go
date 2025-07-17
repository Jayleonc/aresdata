//go:build wireinject
// +build wireinject

package main

import (
	"aresdata/internal/conf"
	"aresdata/internal/data"
	"aresdata/internal/etl"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init etl application.
func wireApp(*conf.Job, *conf.Data, log.Logger) (ETLRunner, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		etl.ProviderSet,
		wire.Bind(new(ETLRunner), new(*etl.ETL)),
	))
}
