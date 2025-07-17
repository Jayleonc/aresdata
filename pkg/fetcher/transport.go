package fetcher

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// throttledTransport 是一个实现了 http.RoundTripper 的结构体
type throttledTransport struct {
	next    http.RoundTripper // 被包裹的下一个 RoundTripper，通常是 http.DefaultTransport
	minWait time.Duration
	maxWait time.Duration
	log     *log.Helper
}

// RoundTrip 是实现接口的核心方法
func (t *throttledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 计算随机等待时间
	waitDuration := t.minWait
	if t.maxWait > t.minWait {
		waitDuration += time.Duration(rand.Int63n(int64(t.maxWait - t.minWait)))
	}

	t.log.Infof("请求被限流，等待 %v 后发送到 %s", waitDuration, req.URL.Path)

	// 等待
	time.Sleep(waitDuration)

	// 调用被包裹的 RoundTripper，执行真正的请求
	return t.next.RoundTrip(req)
}

// NewThrottledTransport 创建一个新的节流 Transport
func NewThrottledTransport(minWaitMs, maxWaitMs int32, logger log.Logger) http.RoundTripper {
	return &throttledTransport{
		next:    http.DefaultTransport, // 使用系统默认的 Transport
		minWait: time.Duration(minWaitMs) * time.Millisecond,
		maxWait: time.Duration(maxWaitMs) * time.Millisecond,
		log:     log.NewHelper(log.With(logger, "module", "fetcher/transport")),
	}
}
