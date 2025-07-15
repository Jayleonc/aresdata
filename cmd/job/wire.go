//go:build wireinject
// +build wireinject

package main

import (
	"aresdata/internal/biz"
	"aresdata/internal/conf"
	"aresdata/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init job application.
func wireApp(*conf.Job, *conf.Data, log.Logger) (*CronService, func(), error) {
	panic(wire.Build(
		data.ProviderSet,
		biz.ProviderSet,
		NewCronService,
	))
}
