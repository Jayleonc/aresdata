package server

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/conf"
	"aresdata/internal/service"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, fetcher *service.FetcherService, videoRank *service.VideoRankService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterFetcherHTTPServer(srv, fetcher)
	v1.RegisterVideoRankHTTPServer(srv, videoRank)

	// 添加 OpenAPI 文档路由
	srv.Handle("/openapi.yaml", OpenAPIHandler("./openapi.yaml"))

	// 添加 Swagger UI 静态文件路由
	srv.Handle("/swagger-ui/", SwaggerUIHandler())

	return srv
}
