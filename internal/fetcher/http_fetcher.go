package fetcher

import (
	"context"
	"fmt"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

// HttpFetcher implements the Fetcher interface for standard HTTP requests.
type HttpFetcher struct {
	log         *log.Helper
	cfg         *conf.DataSource
	accountPool *AccountPool
}

// NewHttpFetcher creates a new HttpFetcher.
func NewHttpFetcher(cfg *conf.DataSource, pool *AccountPool, logger log.Logger) *HttpFetcher {
	return &HttpFetcher{
		log:         log.NewHelper(log.With(logger, "module", fmt.Sprintf("fetcher/http/%s", cfg.Name))),
		cfg:         cfg,
		accountPool: pool,
	}
}

// GetConfig returns the datasource configuration for the fetcher.
func (h *HttpFetcher) FetchAndStoreVideoDetails(ctx context.Context, video *v1.VideoDTO) error {
	h.log.Infof("[HTTP] Fetching video details for: %s", video.AwemeId)
	// TODO: Implement actual HTTP fetch logic here
	// Simulate fetch and store for now
	return nil
}

// GetConfig returns the datasource configuration for the fetcher.
func (hf *HttpFetcher) GetConfig() *conf.DataSource {
	return hf.cfg
}

// CaptureSummary 只采集 summary 原始数据
func (hf *HttpFetcher) CaptureSummary(ctx context.Context, video *v1.VideoDTO) (string, error) {
	hf.log.Infof("[HTTP] CaptureSummary for: %s", video.AwemeId)
	// TODO: 实现实际 HTTP 请求采集 summary 数据
	return "{\"mock_summary\":true}", nil
}

// CaptureTrend 只采集 trend 原始数据
func (hf *HttpFetcher) CaptureTrend(ctx context.Context, video *v1.VideoDTO) (string, error) {
	hf.log.Infof("[HTTP] CaptureTrend for: %s", video.AwemeId)
	// TODO: 实现实际 HTTP 请求采集 trend 数据
	return "{\"mock_trend\":true}", nil
}
