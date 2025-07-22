package fetcher

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// throttledTransport 是一个实现了 http.RoundTripper 的结构体
// throttledTransport 现在包含一个 rateLimiter channel
// next: 被包裹的下一个 RoundTripper，通常是 http.DefaultTransport
// rateLimiter: 用于速率限制的 channel
// log: 日志辅助
type throttledTransport struct {
	next        http.RoundTripper
	rateLimiter <-chan time.Time // 从这个 channel 接收“令牌”
	log         *log.Helper
}

// RoundTrip 是实现接口的核心方法
func (t *throttledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.log.Infof("等待请求许可: %s", req.URL.Path)
	// 使用 select 来同时等待令牌和上下文取消信号
	select {
	case <-t.rateLimiter:
		// 成功获取到令牌，继续执行请求
		t.log.Infof("请求许可已获取，正在发送请求...")
		return t.next.RoundTrip(req)
	case <-req.Context().Done():
		// 在等待令牌期间，请求的上下文被取消了（例如，客户端超时）
		t.log.Warnf("请求在等待限流许可时被取消: %v", req.Context().Err())
		return nil, req.Context().Err()
	}
}

// NewThrottledTransport 创建一个新的节流 Transport
func NewThrottledTransport(minWaitMs, maxWaitMs int32, logger log.Logger) http.RoundTripper {
	// 创建一个 buffer 为 1 的 ticker channel，确保第一个请求可以立即发出
	ticker := make(chan time.Time, 1)
	ticker <- time.Now() // 立即放入一个令牌

	logHelper := log.NewHelper(log.With(logger, "module", "fetcher/transport"))

	// 启动一个 goroutine 来持续生成令牌
	go func() {
		for {
			// 计算下一次生成令牌的随机等待时间
			minWait := time.Duration(minWaitMs) * time.Millisecond
			maxWait := time.Duration(maxWaitMs) * time.Millisecond
			waitDuration := minWait
			if maxWait > minWait {
				waitDuration += time.Duration(rand.Int63n(int64(maxWait - minWait)))
			}

			logHelper.Infof("下一次请求许可将在 %v 后生成", waitDuration)
			time.Sleep(waitDuration)
			ticker <- time.Now()
		}
	}()

	//tr := &http.Transport{
	//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//}

	return &throttledTransport{
		next:        http.DefaultTransport,
		rateLimiter: ticker,
		log:         logHelper,
	}
}
