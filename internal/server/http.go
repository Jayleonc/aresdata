package server

import (
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/Jayleonc/aresdata/internal/service"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/rs/cors"
	nethttp "net/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server,
	videoRank *service.VideoRankService,
	videoService *service.VideoServiceService,
	productService *service.ProductServiceService,
	blogger *service.BloggerServiceService,
	videoTrend *service.VideoTrendServiceService,
	logger log.Logger) *http.Server {
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

	// CORS middleware
	opts = append(opts, http.Filter(func(handler nethttp.Handler) nethttp.Handler {
		return cors.New(cors.Options{
			AllowedOrigins:   []string{"*"}, // 生产环境请替换为具体域名
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		}).Handler(handler)
	}))

	srv := http.NewServer(opts...)

	v1.RegisterVideoRankHTTPServer(srv, videoRank)
	v1.RegisterVideoServiceHTTPServer(srv, videoService)
	v1.RegisterProductServiceHTTPServer(srv, productService)
	v1.RegisterBloggerServiceHTTPServer(srv, blogger)
	v1.RegisterVideoTrendServiceHTTPServer(srv, videoTrend)

	// 添加 OpenAPI 文档路由
	srv.Handle("/openapi.yaml", OpenAPIHandler("./openapi.yaml"))

	// 添加 Swagger UI 静态文件路由
	srv.Handle("/swagger-ui/", SwaggerUIHandler())

	return srv
}
