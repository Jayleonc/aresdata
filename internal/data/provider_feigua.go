package data

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/biz"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
)

type feiguaProvider struct {
	log *log.Helper
}

// NewFeiguaProvider 创建一个飞瓜数据提供商实例
func NewFeiguaProvider(logger log.Logger) biz.Provider {
	return &feiguaProvider{
		log: log.NewHelper(log.With(logger, "module", "provider/feigua")),
	}
}

func (p *feiguaProvider) GetName() string {
	return "FEIGUA" // 返回与proto枚举匹配的字符串
}

// Fetch 是一个分发器，根据任务类型调用不同的内部方法
func (p *feiguaProvider) Fetch(ctx context.Context, task *v1.Task) (string, error) {
	p.log.WithContext(ctx).Infof("Dispatching fetch task for provider [feigua], DataType: [%s]", task.DataType)

	switch task.DataType {
	case "video_rank_daily":
		return p.fetchVideoRank(ctx, task.Payload)
	case "product_detail":
		return p.fetchProductDetail(ctx, task.Payload)
	default:
		return "", fmt.Errorf("unsupported data type for feigua provider: %s", task.DataType)
	}
}

// fetchVideoRank 模拟采集“带货视频日榜”
func (p *feiguaProvider) fetchVideoRank(ctx context.Context, payload string) (string, error) {
	p.log.WithContext(ctx).Info("Executing fetchVideoRank...")
	mockData := map[string]interface{}{
		"id":   "video_rank_20250715",
		"list": []map[string]interface{}{{"rank": 1, "title": "模拟视频1"}},
	}
	rawContent, _ := json.Marshal(mockData)
	return string(rawContent), nil
}

// fetchProductDetail 模拟采集“商品详情”
func (p *feiguaProvider) fetchProductDetail(ctx context.Context, payload string) (string, error) {
	p.log.WithContext(ctx).Info("Executing fetchProductDetail...")
	mockData := map[string]interface{}{"id": "product789", "price": 99.9}
	rawContent, _ := json.Marshal(mockData)
	return string(rawContent), nil
}
