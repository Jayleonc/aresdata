package fetcher

import (
	"aresdata/internal/conf"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// FeiguaFetcher 负责与飞瓜API进行交互
type FeiguaFetcher struct {
	log    *log.Helper
	conf   *conf.Data
	client *http.Client
}

// NewFeiguaFetcher 创建一个新的 FeiguaFetcher
func NewFeiguaFetcher(c *conf.Data, logger log.Logger) *FeiguaFetcher {
	return &FeiguaFetcher{
		log:  log.NewHelper(log.With(logger, "module", "fetcher/feigua")),
		conf: c,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchVideoRank 采集带货视频榜单
func (f *FeiguaFetcher) FetchVideoRank(ctx context.Context, period, datecode string) (string, error) {
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
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 Headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", f.conf.Feigua.Cookie)
	req.Header.Set("Referer", f.conf.Feigua.BaseUrl+"/app/")
	req.Header.Set("Origin", f.conf.Feigua.BaseUrl)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}
