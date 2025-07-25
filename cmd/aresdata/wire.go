//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/Jayleonc/aresdata/internal/biz"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/Jayleonc/aresdata/internal/server"
	"github.com/Jayleonc/aresdata/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
