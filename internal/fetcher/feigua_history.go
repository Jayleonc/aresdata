package fetcher

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/Jayleonc/aresdata/pkg/utils"
	"io"

	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// FeiguaFetcher 负责与飞瓜API进行交互
type FeiguaFetcher struct {
	log         *log.Helper
	cfg         *conf.DataSource
	client      *http.Client
	accountPool *AccountPool
}

func (f *FeiguaFetcher) GetConfig() *conf.DataSource {
	return f.cfg
}

// NewFeiguaFetcher 创建一个新的 FeiguaFetcher
func NewFeiguaFetcher(cfg *conf.DataSource, pool *AccountPool, logger log.Logger) *FeiguaFetcher {
	// 创建我们的节流 transport
	throttledTransport := NewThrottledTransport(
		int32(cfg.ThrottleMinWaitMs),
		int32(cfg.ThrottleMaxWaitMs),
		logger,
	)
	return &FeiguaFetcher{
		log:         log.NewHelper(log.With(logger, "module", fmt.Sprintf("fetcher/http/%s", cfg.Name))),
		cfg:         cfg,
		accountPool: pool,
		client: &http.Client{
			Timeout:   time.Duration(cfg.Timeout) * time.Second,
			Transport: throttledTransport, // 核心：将客户端的 Transport 设置为我们自己的节流 Transport
		},
	}
}

// FetchVideoRank 采集带货视频榜单
func (f *FeiguaFetcher) FetchVideoRank(ctx context.Context, period, datecode string, pageIndex, pageSize int) (string, *RequestMetadata, error) {
	// 基于Python脚本构建请求
	apiEndpoint := f.cfg.BaseUrl + "/api/v3/awemerank/sellGoodsAwemeRank"

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
	f.log.WithContext(ctx).Infof("Requesting URL: %s", fullUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

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

	// 设置 Headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Proxy-Connection", "keep-alive")

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

	// --- 新增：Gzip解压逻辑 ---
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", meta, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()
	default:
		// 如果没有压缩或者是不认识的压缩格式，直接使用原始响应体
		reader = resp.Body
	}
	// --- 解压逻辑结束 ---

	body, err := io.ReadAll(reader) // 注意：这里从新的 reader 读取
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body for video trend: %w", err)
	}

	return string(body), meta, nil

}

// FetchVideoTrend 采集单个视频的趋势数据
func (f *FeiguaFetcher) FetchVideoTrend(ctx context.Context, awemeID string, awemePubTime time.Time) (string, *RequestMetadata, error) {
	apiEndpoint := f.cfg.BaseUrl + "/api/v3/aweme/detail/newTrend"

	t := time.Now().UnixMilli()

	// 手动拼接参数顺序，确保 awemeId, dateCode, period, type, _
	query := fmt.Sprintf("awemeId=%s&dateCode=%s&period=30&type=1&_=%v",
		awemeID,
		awemePubTime.Format("20060102"),
		t,
	)
	fullUrl := apiEndpoint + "?" + query
	f.log.WithContext(ctx).Infof("Requesting Trend URL: %s", fullUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", fullUrl, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request for video trend: %w", err)
	}

	// 设置 Headers
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Proxy-Connection", "keep-alive")

	headersJson, _ := json.Marshal(req.Header)
	meta := &RequestMetadata{
		Method:  "GET",
		URL:     apiEndpoint,
		Params:  query,
		Headers: string(headersJson),
	}

	// --- 新增：打印 curl 命令用于调试 ---
	curlCmd := utils.RequestToCurl(req)
	f.log.WithContext(ctx).Infof("Generated CURL command:\n---\n%s\n---", curlCmd)
	// --- 新增代码结束 ---

	resp, err := f.client.Do(req)
	if err != nil {
		return "", meta, fmt.Errorf("failed to execute request for video trend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", meta, fmt.Errorf("bad status code for video trend: %d", resp.StatusCode)
	}

	// --- 新增：Gzip解压逻辑 ---
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", meta, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()
	default:
		// 如果没有压缩或者是不认识的压缩格式，直接使用原始响应体
		reader = resp.Body
	}
	// --- 解压逻辑结束 ---

	body, err := io.ReadAll(reader) // 注意：这里从新的 reader 读取
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body for video trend: %w", err)
	}

	return string(body), meta, nil
}
