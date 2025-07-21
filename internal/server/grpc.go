package server

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/conf"
	"aresdata/internal/service"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server,
	videoRank *service.VideoRankService,
	videoService *service.VideoServiceService,
	productService *service.ProductServiceService,
	blogger *service.BloggerServiceService,
	logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterVideoRankServer(srv, videoRank)
	v1.RegisterVideoServiceServer(srv, videoService)
	v1.RegisterProductServiceServer(srv, productService)
	v1.RegisterBloggerServiceServer(srv, blogger)
	return srv
}
