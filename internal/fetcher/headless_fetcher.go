// internal/fetcher/headless_fetcher.go
package fetcher

import (
	"aresdata/internal/conf"
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type HeadlessFetcher struct {
	log *log.Helper
	cfg *conf.Feigua
}

func NewHeadlessFetcher(c *conf.Data, logger log.Logger) *HeadlessFetcher {
	return &HeadlessFetcher{
		log: log.NewHelper(logger),
		cfg: c.GetFeigua(),
	}
}

// CaptureAPIs 在一个给定的标签页(ctx)中，导航到entryURL，并监听捕获所有指定的targetAPIPaths。
// 【最终稳定版】: 使用单一监听器关联两个事件，彻底解决并发竞争问题。
func (f *HeadlessFetcher) CaptureAPIs(ctx context.Context, entryURL string, targetAPIPaths []string) (map[string]string, error) {
	f.log.Infof("启动API捕获任务. 入口URL: %s, 目标API数量: %d", entryURL, len(targetAPIPaths))

	results := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(targetAPIPaths))

	// a-t-com/aresdata 请求ID与API路径的映射，用于关联事件
	requestMap := make(map[network.RequestID]string)
	var requestMapMu sync.Mutex

	// 确保监听器在函数返回时被正确取消
	listenCtx, cancelListen := context.WithCancel(ctx)
	defer cancelListen()

	// --- 【核心】只启动一个监听器 ---
	chromedp.ListenTarget(listenCtx, func(ev interface{}) {
		// 分发事件到不同的处理器
		switch ev := ev.(type) {

		// 步骤1：当收到响应头时，进行标记
		case *network.EventResponseReceived:
			for _, apiPath := range targetAPIPaths {
				if strings.Contains(ev.Response.URL, apiPath) {
					requestMapMu.Lock()
					// 标记这个请求ID是我们感兴趣的
					requestMap[ev.RequestID] = apiPath
					requestMapMu.Unlock()
					f.log.Infof("已标记目标API请求: %s (ID: %s)", apiPath, ev.RequestID)
				}
			}

		// 步骤2：当请求完全加载后，进行处理
		case *network.EventLoadingFinished:
			requestMapMu.Lock()
			// 检查这个完成的请求是否在我们标记的列表里
			apiPath, ok := requestMap[ev.RequestID]
			requestMapMu.Unlock()

			if ok {
				// 如果是，启动goroutine去获取响应体
				go func(reqID network.RequestID, path string) {
					// 由于此事件在数据完全加载后触发，所以GetResponseBody是100%安全的
					bodyBytes, err := network.GetResponseBody(reqID).Do(cdp.WithExecutor(listenCtx, chromedp.FromContext(listenCtx).Target))
					if err != nil {
						f.log.Errorf("获取API [%s] 响应体失败: %v", path, err)
						wg.Done() // 即使失败也要标记完成，防止死锁
						return
					}

					// 使用互斥锁安全地写入结果
					mu.Lock()
					if _, exists := results[path]; !exists {
						results[path] = string(bodyBytes)
						f.log.Infof("成功捕获并获取API响应: %s", path)
						wg.Done()
					}
					mu.Unlock()

					// 清理已处理的请求，防止重复处理（可选但推荐）
					requestMapMu.Lock()
					delete(requestMap, reqID)
					requestMapMu.Unlock()

				}(ev.RequestID, apiPath)
			}
		}
	})

	// --- 后续的导航和等待逻辑保持不变 ---
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(entryURL),
		chromedp.Sleep(time.Duration(f.cfg.ThrottleStartWaitMs)*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp导航失败: %w", err)
	}

	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		f.log.Info("所有目标API均已成功捕获。")
		return results, nil
	case <-ctx.Done():
		mu.Lock()
		defer mu.Unlock()
		if len(results) < len(targetAPIPaths) {
			return results, fmt.Errorf("任务超时，仅捕获了 %d/%d 个API", len(results), len(targetAPIPaths))
		}
		return results, nil
	}
}
