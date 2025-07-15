package biz

import (
	v1 "aresdata/api/v1"
	"context"
)

// Provider 定义了所有数据源提供商必须实现的行为
type Provider interface {
	Fetch(ctx context.Context, task *v1.Task) (rawContent string, err error)
	GetName() string
}

// ProviderFactory 用于根据名称获取 Provider 实例
type ProviderFactory struct {
	providers map[string]Provider
}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		providers: make(map[string]Provider),
	}
}

func (f *ProviderFactory) Register(provider Provider) {
	f.providers[provider.GetName()] = provider
}

func (f *ProviderFactory) GetProvider(name string) (Provider, bool) {
	p, ok := f.providers[name]
	return p, ok
}
