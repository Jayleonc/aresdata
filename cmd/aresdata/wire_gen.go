// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/Jayleonc/aresdata/internal/biz"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/Jayleonc/aresdata/internal/server"
	"github.com/Jayleonc/aresdata/internal/service"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
)

import (
	_ "go.uber.org/automaxprocs"
)

// Injectors from wire.go:

// wireApp init kratos application.
func wireApp(confServer *conf.Server, confData *conf.Data, logger log.Logger) (*kratos.App, func(), error) {
	cmdable := data.NewRedisClient(confData)
	dataData, cleanup, err := data.NewData(confData, cmdable, logger)
	if err != nil {
		return nil, nil, err
	}
	videoRankRepo := data.NewVideoRankRepo(dataData)
	videoRankUsecase := biz.NewVideoRankUsecase(videoRankRepo)
	videoRankService := service.NewVideoRankService(videoRankUsecase)
	videoRepo := data.NewVideoRepo(dataData)
	sourceDataRepo := data.NewSourceDataRepo(dataData, logger)
	videoUsecase := biz.NewVideoUsecase(videoRepo, sourceDataRepo)
	videoServiceService := service.NewVideoServiceService(videoUsecase)
	productRepo := data.NewProductRepo(dataData)
	productUsecase := biz.NewProductUsecase(productRepo)
	productServiceService := service.NewProductServiceService(productUsecase)
	bloggerRepo := data.NewBloggerRepo(dataData)
	bloggerUsecase := biz.NewBloggerUsecase(bloggerRepo)
	bloggerServiceService := service.NewBloggerServiceService(bloggerUsecase)
	videoTrendRepo := data.NewVideoTrendRepo(dataData)
	videoTrendUsecase := biz.NewVideoTrendUsecase(videoTrendRepo)
	videoTrendServiceService := service.NewVideoTrendServiceService(videoTrendUsecase)
	grpcServer := server.NewGRPCServer(confServer, videoRankService, videoServiceService, productServiceService, bloggerServiceService, videoTrendServiceService, logger)
	httpServer := server.NewHTTPServer(confServer, videoRankService, videoServiceService, productServiceService, bloggerServiceService, videoTrendServiceService, logger)
	app := newApp(logger, grpcServer, httpServer)
	return app, func() {
		cleanup()
	}, nil
}
