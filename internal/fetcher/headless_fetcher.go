package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/go-kratos/kratos/v2/log"
	"math/rand"
)

// HeadlessFetcher 是采集执行层（士兵），负责具体的浏览器操作
type HeadlessFetcher struct {
	cfg         *conf.DataSource
	accountPool *AccountPool
	log         *log.Helper
}

func NewHeadlessFetcher(cfg *conf.DataSource, pool *AccountPool, logger log.Logger) *HeadlessFetcher {
	return &HeadlessFetcher{
		cfg:         cfg,
		accountPool: pool,
		log:         log.NewHelper(log.With(logger, "module", "fetcher.headless")),
	}
}

// CaptureSummary 负责启动一个一次性的浏览器实例，只为了采集 Summary 数据
func (f *HeadlessFetcher) CaptureSummary(ctx context.Context, entryURL string) (string, error) {
	return f.captureSingleAPI(ctx, entryURL, "/api/v3/aweme/detail/detail/sumData")
}

// CaptureTrend 负责启动一个一次性的浏览器实例，只为了采集 Trend 数据
func (f *HeadlessFetcher) CaptureTrend(ctx context.Context, entryURL string) (string, error) {
	return f.captureSingleAPI(ctx, entryURL, "/api/v3/aweme/detail/newTrend")
}

// captureSingleAPI 是底层的、私有的采集方法，封装了完整的浏览器生命周期
func (f *HeadlessFetcher) captureSingleAPI(ctx context.Context, entryURL, apiPath string) (string, error) {
	// 1. 获取一个新账号
	account := f.accountPool.GetNextAccount()
	if account == nil {
		return "", fmt.Errorf("no available accounts")
	}
	f.log.Infof("Using account from %s to capture API [%s]", account.path, apiPath)

	// 2. 构建浏览器伪装参数
	opts := f.buildBrowserOptions()
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// 3. 启动一个带超时的浏览器实例
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(f.log.Infof))
	defer cancel()

	ctxTimeout, cancel := context.WithTimeout(taskCtx, 2*time.Minute)
	defer cancel()

	// 4. 加载Cookie并注入JS伪装
	if err := f.setCookies(ctxTimeout, account.Cookies); err != nil {
		return "", fmt.Errorf("failed to set cookies: %w", err)
	}
	if err := chromedp.Run(ctxTimeout, chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil)); err != nil {
		return "", fmt.Errorf("failed to inject navigator.webdriver: %w", err)
	}

	// 5. 监听并捕获唯一的API
	apiResult, err := f.captureAPI(ctxTimeout, entryURL, apiPath)
	if err != nil {
		return "", err
	}

	return apiResult, nil
}

// captureAPI 是真正执行监听和捕获的逻辑
func (f *HeadlessFetcher) captureAPI(ctx context.Context, entryURL string, targetAPIPath string) (string, error) {
	var result string
	var wg sync.WaitGroup
	wg.Add(1)

	listenCtx, cancelListen := context.WithCancel(ctx)
	defer cancelListen()

	chromedp.ListenTarget(listenCtx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			if strings.Contains(ev.Response.URL, targetAPIPath) {
				// 使用goroutine以非阻塞方式获取响应体
				go func(reqID network.RequestID) {
					bodyBytes, err := network.GetResponseBody(reqID).Do(cdp.WithExecutor(listenCtx, chromedp.FromContext(listenCtx).Target))
					if err == nil {
						result = string(bodyBytes)
						f.log.Infof("Successfully captured API response: %s", targetAPIPath)
					} else {
						f.log.Errorf("Failed to get response body for API [%s]: %v", targetAPIPath, err)
					}
					wg.Done()
				}(ev.RequestID)
			}
		}
	})

	// 执行导航
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(entryURL), chromedp.Sleep(2*time.Second)); err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	// 等待API被捕获或超时
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		return result, nil
	case <-ctx.Done():
		if result != "" {
			return result, nil
		}
		return "", fmt.Errorf("task timed out while waiting for API: %s", targetAPIPath)
	}
}

// buildBrowserOptions 生成带伪装的浏览器启动参数
func (f *HeadlessFetcher) buildBrowserOptions() []chromedp.ExecAllocatorOption {
	userAgent := f.cfg.Headless.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	}
	width := rand.Intn(1920-1366+1) + 1366
	height := rand.Intn(1080-768+1) + 768

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent(userAgent),
		chromedp.WindowSize(width, height),
	)

	if f.cfg.Proxy != "" {
		opts = append(opts, chromedp.ProxyServer(f.cfg.Proxy))
	}
	return opts
}

// setCookies 辅助函数，用于设置cookies
func (f *HeadlessFetcher) setCookies(ctx context.Context, cookies []*http.Cookie) error {
	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		for _, cookie := range cookies {
			expr := cdp.TimeSinceEpoch(cookie.Expires)
			err := network.SetCookie(cookie.Name, cookie.Value).
				WithDomain(cookie.Domain).
				WithPath(cookie.Path).
				WithHTTPOnly(cookie.HttpOnly).
				WithSecure(cookie.Secure).
				WithExpires(&expr).
				Do(ctx)
			if err != nil {
				return fmt.Errorf("could not set cookie %s: %v", cookie.Name, err)
			}
		}
		return nil
	}))
}
