package fetcher

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/go-kratos/kratos/v2/log"
)

// HeadlessFetcher satisfies the Fetcher interface.
var _ Fetcher = &HeadlessFetcher{}

// HeadlessFetcher 是采集执行层（士兵），负责具体的浏览器操作。
// 它现在是高效的，使用单个浏览器实例来完成所有API的捕获。
type HeadlessFetcher struct {
	cfg         *conf.DataSource
	accountPool *AccountPool
	log         *log.Helper
}

// NewHeadlessFetcher 构造函数保持不变。
func NewHeadlessFetcher(cfg *conf.DataSource, pool *AccountPool, logger log.Logger) *HeadlessFetcher {
	return &HeadlessFetcher{
		cfg:         cfg,
		accountPool: pool,
		log:         log.NewHelper(log.With(logger, "module", "fetcher.headless")),
	}
}

func (f *HeadlessFetcher) GetConfig() *conf.DataSource {
	return f.cfg
}

// FetchVideoRank 【新增】明确此功能不由 HeadlessFetcher 实现，以满足接口要求
func (f *HeadlessFetcher) FetchVideoRank(ctx context.Context, period, datecode string, pageIndex, pageSize int) (string, *RequestMetadata, error) {
	return "", nil, fmt.Errorf("FetchVideoRank 方法未在 HeadlessFetcher 中实现")
}

// CaptureVideoDetails 是对外暴露的唯一采集方法，符合接口定义。
// 它负责编排整个无头浏览器采集流程：启动、伪装、导航、捕获、关闭。
func (f *HeadlessFetcher) CaptureVideoDetails(ctx context.Context, entryURL string) (string, string, error) {
	// 1. 准备工作：获取账号、构建浏览器选项
	account := f.accountPool.GetNextAccount()
	if account == nil {
		return "", "", fmt.Errorf("账号池为空，无可用账号")
	}
	f.log.Infof("使用账号 [%s] 准备采集任务，入口: %s", account.path, entryURL)
	opts := f.buildBrowserOptions()

	// 2. 初始化浏览器上下文
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(f.log.Infof))
	defer cancel()

	// 3. 设置一个总体的任务超时
	taskTimeoutCtx, cancel := context.WithTimeout(browserCtx, 2*time.Minute)
	defer cancel()

	// 4. 执行核心的采集逻辑
	summaryAPIPath := "/api/v3/aweme/detail/detail/sumData"
	trendAPIPath := "/api/v3/aweme/detail/newTrend"

	// 调用高效的多API采集器
	results, err := f.captureMultipleAPIs(taskTimeoutCtx, entryURL, account, summaryAPIPath, trendAPIPath)
	if err != nil {
		// 即便有错误，也尝试返回已经采集到的数据
		return results[summaryAPIPath], results[trendAPIPath], fmt.Errorf("采集过程发生错误: %w", err)
	}

	// 5. 成功返回结果
	summaryRaw := results[summaryAPIPath]
	trendRaw := results[trendAPIPath]

	if summaryRaw == "" && trendRaw == "" {
		return "", "", fmt.Errorf("采集失败，摘要和趋势数据均为空")
	}

	f.log.Infof("视频 [%s] 采集任务完成", entryURL)
	return summaryRaw, trendRaw, nil
}

// captureMultipleAPIs 是高效采集的核心，它只导航一次，捕获多个API。
func (f *HeadlessFetcher) captureMultipleAPIs(ctx context.Context, entryURL string, account *Account, targetPaths ...string) (map[string]string, error) {
	// 初始化一个map来存放结果，key是API路径，value是响应体
	results := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 根据目标API的数量设置WaitGroup
	targetsToFind := make(map[string]bool)
	for _, path := range targetPaths {
		targetsToFind[path] = true
	}
	wg.Add(len(targetsToFind))

	// 创建一个可以被提前取消的监听上下文
	listenCtx, stopListen := context.WithCancel(ctx)
	defer stopListen()

	// 设置监听器，这是整个采集过程的关键
	chromedp.ListenTarget(listenCtx, func(ev interface{}) {
		e, ok := ev.(*network.EventResponseReceived)
		if !ok {
			return
		}

		// 遍历我们需要寻找的所有目标API
		for path := range targetsToFind {
			if strings.Contains(e.Response.URL, path) {
				// 找到了一个！启动goroutine去获取响应体
				go func(reqID network.RequestID, foundPath string) {
					body, err := network.GetResponseBody(reqID).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Target))
					if err != nil {
						f.log.Errorf("获取API [%s] 响应体失败: %v", foundPath, err)
						wg.Done() // 即使失败也要Done，否则会死锁
						return
					}

					// 成功获取，存入结果map
					mu.Lock()
					if results[foundPath] == "" { // 避免重复写入
						results[foundPath] = string(body)
						f.log.Infof("成功捕获API: %s", foundPath)
						wg.Done()
					}
					mu.Unlock()
				}(e.RequestID, path)
				// 一旦匹配，就不需要再为这个URL检查其他路径了
				return
			}
		}
	})

	// 定义完整的浏览器行为序列
	actions := []chromedp.Action{
		// 启用网络监听
		network.Enable(),
		// 加载Cookie
		f.setCookiesAction(account.Cookies),
		// 注入JS，隐藏webdriver特征
		chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil),
		// 导航到目标页面
		chromedp.Navigate(entryURL),
	}

	// 执行浏览器行为
	if err := chromedp.Run(ctx, actions...); err != nil {
		return results, fmt.Errorf("执行浏览器导航和设置失败: %w", err)
	}

	// 等待所有API被捕获，或者等待任务超时
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		f.log.Info("所有目标API均已捕获。")
		return results, nil
	case <-ctx.Done():
		f.log.Warnf("任务超时，可能部分API未捕获。")
		return results, ctx.Err()
	}
}

// buildBrowserOptions 生成带伪装的浏览器启动参数，保持不变。
func (f *HeadlessFetcher) buildBrowserOptions() []chromedp.ExecAllocatorOption {
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	if f.cfg.Headless != nil && f.cfg.Headless.UserAgent != "" {
		userAgent = f.cfg.Headless.UserAgent
	}
	width := rand.Intn(1920-1366+1) + 1366
	height := rand.Intn(1080-768+1) + 768

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent(userAgent),
		chromedp.WindowSize(width, height),
		// 其他必要的flag...
	)

	if f.cfg.Proxy != "" {
		opts = append(opts, chromedp.ProxyServer(f.cfg.Proxy))
	}
	return opts
}

// setCookiesAction 返回一个可被chromedp.Run执行的Action，用于设置Cookie。
func (f *HeadlessFetcher) setCookiesAction(cookies []*http.Cookie) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		for _, cookie := range cookies {
			// 跳过已经过期的cookie
			if !cookie.Expires.IsZero() && cookie.Expires.Before(time.Now()) {
				continue
			}
			expr := cdp.TimeSinceEpoch(cookie.Expires)
			err := network.SetCookie(cookie.Name, cookie.Value).
				WithDomain(cookie.Domain).
				WithPath(cookie.Path).
				WithHTTPOnly(cookie.HttpOnly).
				WithSecure(cookie.Secure).
				WithExpires(&expr).
				Do(ctx)
			if err != nil {
				// 仅记录日志，不中断整个流程
				f.log.Warnf("设置cookie [%s] 失败: %v", cookie.Name, err)
			}
		}
		return nil
	})
}

func (f *HeadlessFetcher) FetchVideoSummary(ctx context.Context, awemeID, dateCode string) (string, *RequestMetadata, error) {
	//TODO implement me
	panic("implement me")
}
