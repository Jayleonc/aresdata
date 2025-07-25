package fetcher

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jayleonc/aresdata/internal/conf" // 确保路径正确
	"github.com/go-kratos/kratos/v2/log"
)

// HttpFetcher 为标准HTTP请求实现Fetcher接口。
type HttpFetcher struct {
	log         *log.Helper
	cfg         *conf.DataSource
	client      *http.Client
	accountPool *AccountPool
}

// NewHttpFetcher 创建一个新的HttpFetcher。
func NewHttpFetcher(cfg *conf.DataSource, pool *AccountPool, logger log.Logger) *HttpFetcher {
	// 此处可以添加您在旧代码中可能有的限流逻辑
	return &HttpFetcher{
		log:         log.NewHelper(log.With(logger, "module", fmt.Sprintf("fetcher/http/%s", cfg.Name))),
		cfg:         cfg,
		accountPool: pool,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

// GetConfig 返回采集器的数据源配置。
func (f *HttpFetcher) GetConfig() *conf.DataSource {
	return f.cfg
}

// FetchVideoRank 获取视频榜单数据。这是HttpFetcher的核心职责。
func (f *HttpFetcher) FetchVideoRank(ctx context.Context, period, datecode string, pageIndex, pageSize int) (string, *RequestMetadata, error) {
	apiEndpoint := f.cfg.BaseUrl + "/api/v3/awemerank/sellGoodsAwemeRank"

	// 此处完全复用 feigua_history.go 中的参数构建和请求逻辑
	params := url.Values{}
	params.Set("pageIndex", fmt.Sprintf("%d", pageIndex))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	params.Set("period", period)
	params.Set("desc", "1")
	params.Set("datecode", datecode)
	params.Set("sort", "14")
	params.Set("rankType", "14")
	params.Set("priceRange", "1-500") // 1 到 500 元，可以过滤掉那些薅羊毛之类的视频
	params.Set("_", fmt.Sprintf("%d", time.Now().UnixMilli()))

	fullUrl := apiEndpoint + "?" + params.Encode()
	f.log.WithContext(ctx).Infof("正在请求URL: %s", fullUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, nil)
	if err != nil {
		return "", nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 从账号池获取账号并设置Cookie
	account := f.accountPool.GetNextAccount()
	cookieStr := ""
	if account != nil {
		cookieStr = account.GetCookieHeader()
		req.Header.Set("Cookie", cookieStr)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) ...") // 使用真实的UA
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip") // 明确使用gzip

	headersJson, _ := json.Marshal(req.Header)
	meta := &RequestMetadata{
		Method:  "GET",
		URL:     apiEndpoint,
		Params:  params.Encode(),
		Headers: string(headersJson),
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", meta, fmt.Errorf("执行请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", meta, fmt.Errorf("错误的状态码: %d", resp.StatusCode)
	}

	// 处理Gzip压缩
	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", meta, fmt.Errorf("创建gzip读取器失败: %w", err)
		}
		defer reader.Close()
	} else {
		reader = resp.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", meta, fmt.Errorf("读取响应体失败: %w", err)
	}

	return string(body), meta, nil
}

// CaptureVideoDetails 不是HttpFetcher的职责，返回错误。
func (f *HttpFetcher) CaptureVideoDetails(ctx context.Context, entryURL string) (string, string, error) {
	return "", "", fmt.Errorf("CaptureVideoDetails 方法未在 HttpFetcher 中实现")
}

// FetchVideoSummary 采集单个视频的总览数据
func (f *HttpFetcher) FetchVideoSummary(ctx context.Context, awemeID, dateCode string) (string, *RequestMetadata, error) {
	apiEndpoint := f.cfg.BaseUrl + "/api/v3/aweme/detail/detail/sumData"

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
	account := f.accountPool.GetNextAccount()
	if account != nil && len(account.Cookies) > 0 {
		var sb strings.Builder
		for i, cookie := range account.Cookies {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(cookie.Name)
			sb.WriteString("=")
			sb.WriteString(cookie.Value)
		}
		req.Header.Set("Cookie", sb.String())
	}

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body for video summary: %w", err)
	}

	return string(body), meta, nil
}
