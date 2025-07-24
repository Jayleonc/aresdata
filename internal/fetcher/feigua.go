package fetcher

import (
	"aresdata/internal/conf"
	"aresdata/pkg/utils"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
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
func (f *FeiguaFetcher) FetchVideoRank(ctx context.Context, period, datecode string, pageIndex, pageSize int) (string, *RequestMetadata, error) {
	// 基于Python脚本构建请求
	apiEndpoint := f.conf.Feigua.BaseUrl + "/api/v3/awemerank/sellGoodsAwemeRank"

	params := url.Values{}
	params.Set("pageIndex", fmt.Sprintf("%d", pageIndex))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))
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

	// --- 这部分代码现在是健壮的，无需修改 ---
	cookieStr, err := f.getCookieString()
	if err != nil {
		f.log.WithContext(ctx).Warnf("加载 Cookie 失败: %v", err)
	}
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}

	// 设置 Headers
	//req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	//req.Header.Set("Cookie", f.conf.Feigua.Cookie)
	//req.Header.Set("Referer", f.conf.Feigua.BaseUrl+"/app/")
	//req.Header.Set("Origin", f.conf.Feigua.BaseUrl)
	//req.Header.Set("Accept", "application/json, text/plain, */*")
	//req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	//req.Header.Set("Accept-Encoding", "gzip, deflate")
	//req.Header.Set("Connection", "keep-alive")

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	//req.Header.Set("Cookie", f.conf.Feigua.Cookie)
	//req.Header.Set("Referer", f.conf.Feigua.BaseUrl+"/app/")
	//req.Header.Set("Origin", f.conf.Feigua.BaseUrl)
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

	body, err := ioutil.ReadAll(reader) // 注意：这里从新的 reader 读取
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body for video trend: %w", err)
	}

	return string(body), meta, nil

}

// FetchVideoTrend 采集单个视频的趋势数据
func (f *FeiguaFetcher) FetchVideoTrend(ctx context.Context, awemeID string, awemePubTime time.Time) (string, *RequestMetadata, error) {
	apiEndpoint := f.conf.Feigua.BaseUrl + "/api/v3/aweme/detail/detail/trends"

	//// 自定义时间
	//fromDateCode := awemePubTime.Format("20060102")
	//toDateCode := time.Now().Format("20060102")
	//
	//// 构建新的查询参数，使用 period=0 & type=3 并指定起止日期
	//query := fmt.Sprintf("awemeId=%s&dateCode=%s&period=0&type=3&fromDateCode=%s&toDateCode=%s&_=%d",
	//	awemeID,
	//	fromDateCode, // dateCode 通常与 fromDateCode 保持一致
	//	fromDateCode,
	//	toDateCode,
	//	time.Now().UnixMilli(),
	//)
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	cookieStr, err := f.getCookieString()
	if err != nil {
		f.log.WithContext(ctx).Warnf("加载 Cookie 失败: %v", err)
	}
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)

	}
	//req.Header.Set("Referer", f.conf.Feigua.BaseUrl+"/app/")
	//req.Header.Set("Origin", f.conf.Feigua.BaseUrl)
	//req.Header.Set("Host", "118.178.59.83:8085")
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

	body, err := ioutil.ReadAll(reader) // 注意：这里从新的 reader 读取
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
	cookieStr, err := f.getCookieString()
	if err != nil {
		f.log.WithContext(ctx).Warnf("加载 Cookie 失败: %v", err)
	}
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", meta, fmt.Errorf("failed to read response body for video summary: %w", err)
	}

	return string(body), meta, nil
}

// cookiePair 定义了一个临时的结构体，用于解析JSON Cookie数组中的关键字段
type cookiePair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// getCookieString 封装了从配置中获取并格式化cookie的逻辑。
// 它会读取JSON格式的cookie，并将其转换为HTTP头部所需的 "key=value;" 字符串格式。
func (f *FeiguaFetcher) getCookieString() (string, error) {
	var cookieBytes []byte
	var err error

	// 步骤 1: 加载原始Cookie数据 (逻辑与之前一致)
	if f.conf.Feigua.CookieContent != "" {
		f.log.Info("从配置 `cookie_content` 中加载 Cookie...")
		cookieBytes = []byte(f.conf.Feigua.CookieContent)
	} else if f.conf.Feigua.CookiePath != "" {
		f.log.Infof("从文件 %s 中加载 Cookie...", f.conf.Feigua.CookiePath)
		cookieBytes, err = ioutil.ReadFile(f.conf.Feigua.CookiePath)
		if err != nil {
			return "", fmt.Errorf("读取 cookie 文件失败 %s: %w", f.conf.Feigua.CookiePath, err)
		}
	} else {
		f.log.Warn("未配置 Cookie 路径或内容，将以无身份状态访问")
		return "", nil
	}

	// 步骤 2: 解析JSON数据
	var cookies []cookiePair
	if err := json.Unmarshal(cookieBytes, &cookies); err != nil {
		// 增加一个兼容性分支：如果解析JSON失败，则假定它就是旧的字符串格式
		f.log.Warnf("解析 Cookie JSON 失败: %v。将尝试作为普通字符串处理。", err)
		return string(cookieBytes), nil
	}

	if len(cookies) == 0 {
		return "", fmt.Errorf("Cookie JSON 文件解析成功，但内容为空")
	}

	// 步骤 3: 将解析后的cookie转换为 "key=value;" 格式的字符串
	var cookieParts []string
	for _, c := range cookies {
		cookieParts = append(cookieParts, c.Name+"="+c.Value)
	}

	formattedCookie := strings.Join(cookieParts, "; ")
	f.log.Infof("成功将 %d 个 Cookie 格式化为 HTTP 请求头字符串", len(cookies))

	return formattedCookie, nil
}
