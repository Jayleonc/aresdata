package fetcher

import (
	"aresdata/internal/conf"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// RequestMetadata 存储每次API请求的元数据
type RequestMetadata struct {
	Method  string
	URL     string
	Params  string // JSON string of params
	Headers string // JSON string of headers
}

// FeiguaFetcher 负责与飞瓜API进行交互
type FeiguaFetcher struct {
	log    *log.Helper
	conf   *conf.Data
	client *http.Client
}

// NewFeiguaFetcher 创建一个新的 FeiguaFetcher
func NewFeiguaFetcher(c *conf.Data, logger log.Logger) *FeiguaFetcher {
	// 创建我们的节流 transport
	throttledTransport := NewThrottledTransport(
		c.Feigua.ThrottleMinWaitMs,
		c.Feigua.ThrottleMaxWaitMs,
		logger,
	)

	return &FeiguaFetcher{
		log:  log.NewHelper(log.With(logger, "module", "fetcher/feigua")),
		conf: c,
		client: &http.Client{
			Timeout:   150 * time.Second,
			Transport: throttledTransport, // 核心：将客户端的 Transport 设置为我们自己的节流 Transport
		},
	}
}

// FetchVideoRank 采集带货视频榜单
func (f *FeiguaFetcher) FetchVideoRank(ctx context.Context, period, datecode string) (string, *RequestMetadata, error) {
	// 基于Python脚本构建请求
	apiEndpoint := f.conf.Feigua.BaseUrl + "/api/v3/awemerank/sellGoodsAwemeRank"

	params := url.Values{}
	params.Set("pageIndex", "1")
	params.Set("pageSize", "50")
	params.Set("period", period)
	params.Set("desc", "1")
	params.Set("datecode", datecode)
	params.Set("sort", "14")
	params.Set("rankType", "14")
	params.Set("priceRange", "0-500")
	params.Set("_", fmt.Sprintf("%d", time.Now().UnixMilli()))

	fullUrl := apiEndpoint + "?" + params.Encode()
	f.log.WithContext(ctx).Infof("Requesting URL: %s", fullUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 Headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", f.conf.Feigua.Cookie)
	req.Header.Set("Referer", f.conf.Feigua.BaseUrl+"/app/")
	req.Header.Set("Origin", f.conf.Feigua.BaseUrl)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	// 捕获元数据
	headersJson, _ := json.Marshal(req.Header)
	meta := &RequestMetadata{
		Method:  "GET",
		URL:     apiEndpoint, // 只存基础URL，完整URL可由参数重构
		Params:  params.Encode(),
		Headers: string(headersJson),
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", meta, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", meta, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), meta, nil
}

// FetchVideoTrend 采集单个视频的趋势数据
func (f *FeiguaFetcher) FetchVideoTrend(ctx context.Context, awemeID string, awemePubTime time.Time) (string, *RequestMetadata, error) {
	apiEndpoint := f.conf.Feigua.BaseUrl + "/api/v3/aweme/detail/detail/trends"

	// 手动拼接参数顺序，确保 awemeId, dateCode, period, type, _
	query := fmt.Sprintf("awemeId=%s&dateCode=%s&period=30&type=1&_=%d",
		awemeID,
		awemePubTime.Format("20060102"),
		time.Now().UnixMilli(),
	)
	fullUrl := apiEndpoint + "?" + query
	f.log.WithContext(ctx).Infof("Requesting Trend URL: %s", fullUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request for video trend: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", f.conf.Feigua.Cookie)
	req.Header.Set("Referer", f.conf.Feigua.BaseUrl+"/app/")
	req.Header.Set("Origin", f.conf.Feigua.BaseUrl)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	headersJson, _ := json.Marshal(req.Header)
	meta := &RequestMetadata{
		Method:  "GET",
		URL:     apiEndpoint,
		Params:  query,
		Headers: string(headersJson),
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", meta, fmt.Errorf("failed to execute request for video trend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", meta, fmt.Errorf("bad status code for video trend: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body for video trend: %w", err)
	}

	return string(body), meta, nil
}

// FetchVideoSummary 采集单个视频的总览数据
func (f *FeiguaFetcher) FetchVideoSummary(ctx context.Context, awemeID, dateCode string) (string, *RequestMetadata, error) {
	apiEndpoint := f.conf.Feigua.BaseUrl + "/api/v3/aweme/detail/detail/sumData"

	params := url.Values{}
	params.Set("awemeId", awemeID)
	params.Set("dateCode", dateCode) // 使用传入的 dateCode，通常是发布日期
	params.Set("_", fmt.Sprintf("%d", time.Now().UnixMilli()))

	fullUrl := apiEndpoint + "?" + params.Encode()
	f.log.WithContext(ctx).Infof("Requesting Summary URL: %s", fullUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request for video summary: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", f.conf.Feigua.Cookie)

	headersJson, _ := json.Marshal(req.Header)
	meta := &RequestMetadata{
		Method:  "GET",
		URL:     apiEndpoint,
		Params:  params.Encode(),
		Headers: string(headersJson),
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", meta, fmt.Errorf("failed to execute request for video summary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", meta, fmt.Errorf("bad status code for video summary: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body for video summary: %w", err)
	}

	return string(body), meta, nil
}
