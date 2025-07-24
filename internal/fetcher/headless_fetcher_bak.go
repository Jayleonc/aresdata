// internal/fetcher/headless_fetcher.go
package fetcher

//
//import (
//	"aresdata/internal/conf"
//	"context"
//	"encoding/json"
//	"fmt"
//	"github.com/chromedp/cdproto/cdp"
//	"io/ioutil"
//	"strings"
//	"time"
//
//	"github.com/chromedp/cdproto/network"
//	"github.com/chromedp/chromedp"
//	"github.com/go-kratos/kratos/v2/log"
//)
//
//// HeadlessRequestMetadata 存储由无头浏览器发出的请求的元数据
//type HeadlessRequestMetadata struct {
//	Method string `json:"method"`
//	URL    string `json:"url"`
//	Params string `json:"params"`
//}
//
//// HeadlessFetcher 使用 chromedp 实现无头浏览器采集
//type HeadlessFetcher struct {
//	log *log.Helper
//	cfg *conf.Feigua
//}
//
//// NewHeadlessFetcher 创建一个新的 HeadlessFetcher 实例
//func NewHeadlessFetcher(c *conf.Data, logger log.Logger) *HeadlessFetcher {
//	return &HeadlessFetcher{
//		log: log.NewHelper(logger),
//		cfg: c.GetFeigua(),
//	}
//}
//
//// FetchTrend 先访问入口页面，执行JS生成签名URL，再导航并拦截API响应
//func (f *HeadlessFetcher) FetchTrend(ctx context.Context, awemeID string, dateCode string) (string, *HeadlessRequestMetadata, error) {
//	f.log.Infof("开始为视频 %s 执行【无头浏览器方案】采集...", awemeID)
//
//	// --- 1. 定义入口页面URL和元数据 ---
//	entryPageURL := f.cfg.BaseUrl + "/app/#/"
//	f.log.Infof("步骤1: 导航至入口页面: %s", entryPageURL)
//
//	apiEndpoint := f.cfg.BaseUrl + "/api/v3/aweme/detail/detail/trends"
//	query := fmt.Sprintf("awemeId=%s&dateCode=%s&period=30&type=1", awemeID, dateCode)
//	meta := &HeadlessRequestMetadata{
//		Method: "GET",
//		URL:    apiEndpoint,
//		Params: query,
//	}
//
//	// 设置 Allocator 选项
//	opts := []chromedp.ExecAllocatorOption{
//		chromedp.NoFirstRun,
//		chromedp.NoDefaultBrowserCheck,
//		chromedp.Flag("headless", f.cfg.Headless),
//		chromedp.UserAgent(f.cfg.UserAgent),
//	}
//	if f.cfg.Proxy != "" {
//		opts = append(opts, chromedp.ProxyServer(f.cfg.Proxy))
//	}
//	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
//	defer cancel()
//
//	bopts := []chromedp.BrowserOption{
//		chromedp.WithBrowserErrorf(func(s string, a ...interface{}) {}),
//	}
//	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithBrowserOption(bopts...), chromedp.WithLogf(f.log.Infof))
//	taskCtx, cancel = context.WithTimeout(taskCtx, time.Duration(f.cfg.Timeout)*time.Second)
//	defer cancel()
//
//	// --- 2. 设置网络请求监听 ---
//	responseChan := make(chan string, 1)
//	listenCtx, cancelListen := context.WithCancel(taskCtx)
//	defer cancelListen()
//	chromedp.ListenTarget(listenCtx, func(ev interface{}) {
//		go func() {
//			respEvent, ok := ev.(*network.EventResponseReceived)
//			if !ok || !strings.Contains(respEvent.Response.URL, "/api/v3/aweme/detail/detail/trends") {
//				return
//			}
//			f.log.Infof("已拦截到目标API的响应: %s", respEvent.Response.URL)
//			body, err := network.GetResponseBody(respEvent.RequestID).Do(cdp.WithExecutor(listenCtx, chromedp.FromContext(listenCtx).Target))
//			if err != nil {
//				f.log.Errorf("获取响应体失败: %v", err)
//				return
//			}
//			select {
//			case responseChan <- string(body):
//				f.log.Info("已成功获取响应体并发送至channel")
//			default:
//			}
//			cancelListen()
//		}()
//	})
//
//	// --- 3. 执行终极任务流程 ---
//	var signedURL string
//	err := chromedp.Run(taskCtx,
//		f.loadCookiesAction(),
//		chromedp.Navigate(entryPageURL),
//		chromedp.Evaluate(fmt.Sprintf(`window.getSign('%s')`, awemeID), &signedURL),
//		chromedp.ActionFunc(func(ctx context.Context) error {
//			if signedURL == "" {
//				return fmt.Errorf("通过JS获取签名URL失败，返回为空")
//			}
//			f.log.Infof("步骤4: 已通过JS获取到签名URL，准备导航: %s", signedURL)
//			return chromedp.Navigate(signedURL).Do(ctx)
//		}),
//	)
//
//	if err != nil {
//		return "", meta, fmt.Errorf("chromedp.Run 执行失败: %w", err)
//	}
//
//	// --- 4. 等待channel返回结果或超时 ---
//	select {
//	case <-taskCtx.Done():
//		return "", meta, fmt.Errorf("任务被取消或超时 (awemeID: %s)", awemeID)
//	case rawContent := <-responseChan:
//		f.log.Infof("已成功采集视频 %s 的完整趋势数据", awemeID)
//		return rawContent, meta, nil
//	}
//}
//
//// loadCookiesAction 是一个辅助函数，用于加载 Cookies (代码不变)
//func (f *HeadlessFetcher) loadCookiesAction() chromedp.Action {
//	return chromedp.ActionFunc(func(ctx context.Context) error {
//		var cookiesBytes []byte
//		var err error
//
//		if f.cfg.CookiePath != "" {
//			cookiesBytes, err = ioutil.ReadFile(f.cfg.CookiePath)
//			if err != nil {
//				return fmt.Errorf("读取 cookie 文件失败 %s: %w", f.cfg.CookiePath, err)
//			}
//			f.log.Infof("从文件 %s 中加载 Cookies...", f.cfg.CookiePath)
//		} else {
//			f.log.Warn("未配置 Cookie 路径，将以无身份状态访问")
//			return nil
//		}
//
//		processedCookiesStr := strings.ReplaceAll(string(cookiesBytes), `"sameSite": "unspecified"`, `"sameSite": "Lax"`)
//		cookiesBytes = []byte(processedCookiesStr)
//
//		cookies := []*network.CookieParam{}
//		if err := json.Unmarshal(cookiesBytes, &cookies); err != nil {
//			return fmt.Errorf("解析 cookie JSON 失败: %w", err)
//		}
//
//		f.log.Infof("成功解析 %d 个 Cookies，准备设置...", len(cookies))
//		return network.SetCookies(cookies).Do(ctx)
//	})
//}
