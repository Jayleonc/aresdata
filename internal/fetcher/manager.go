package fetcher

import (
	"fmt"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	ProviderTypeHeadless = "headless"
	ProviderTypeHttp     = "http"
)

// FetcherManager 负责持有和管理所有数据源的 Fetcher 实例。
type FetcherManager struct {
	fetchers map[string]Fetcher
	log      *log.Helper
}

// NewFetcherManager 根据配置创建并初始化所有 Fetcher。
func NewFetcherManager(dataSources []*conf.DataSource, logger log.Logger) (*FetcherManager, error) {
	helper := log.NewHelper(log.With(logger, "module", "fetcher/manager"))
	mgr := &FetcherManager{
		fetchers: make(map[string]Fetcher),
		log:      helper,
	}

	helper.Infof("正在初始化 FetcherManager，共 %d 个数据源...", len(dataSources))

	for _, dsConfig := range dataSources {
		helper.Infof("正在为数据源 [%s] (类型: %s) 创建 Fetcher...", dsConfig.Name, dsConfig.Type)

		if len(dsConfig.AccountPool) == 0 {
			helper.Warnf("数据源 '%s' 的账号池为空，已跳过。", dsConfig.Name)
			continue
		}

		// 1. 为当前数据源创建独立的账号池
		accountPool, err := NewAccountPool(dsConfig.AccountPool, logger)
		if err != nil {
			helper.Errorf("为数据源 '%s' 创建账号池失败: %v。已跳过。", dsConfig.Name, err)
			continue
		}

		// 2. 根据类型创建对应的 Fetcher 实例
		var fetcherInstance Fetcher
		switch dsConfig.Type {
		case "headless":
			fetcherInstance = NewHeadlessFetcher(dsConfig, accountPool, logger)
			helper.Infof("成功初始化 'headless' 类型 Fetcher: %s", dsConfig.Name)
		case "http":
			fetcherInstance = NewHttpFetcher(dsConfig, accountPool, logger)
			helper.Infof("成功初始化 'http' 类型 Fetcher: %s", dsConfig.Name)
		default:
			helper.Warnf("不支持的 Fetcher 类型 '%s' (数据源: '%s')。已跳过。", dsConfig.Type, dsConfig.Name)
			continue // 跳过不支持的类型
		}

		// 3. 将完全符合接口规范的实例存入管理器
		mgr.fetchers[dsConfig.Name] = fetcherInstance
	}

	if len(mgr.fetchers) == 0 {
		return nil, fmt.Errorf("配置中没有任何有效的数据源被成功初始化")
	}

	helper.Info("FetcherManager 初始化成功。")
	return mgr, nil
}

// Get 按名称安全地获取一个 Fetcher 实例。
func (m *FetcherManager) Get(name string) (Fetcher, bool) {
	f, ok := m.fetchers[name]
	if !ok {
		m.log.Warnf("试图获取一个不存在的 Fetcher: %s", name)
	}
	return f, ok
}

// GetDataSourceNames 返回所有可用数据源的名称列表。
func (m *FetcherManager) GetDataSourceNames() []string {
	names := make([]string, 0, len(m.fetchers))
	for name := range m.fetchers {
		names = append(names, name)
	}
	return names
}

// GetDataSourcesByType returns a list of data source names for a specific type.
func (m *FetcherManager) GetDataSourcesByType(fetcherType string) []string {
	names := make([]string, 0)
	for name, fetcher := range m.fetchers {
		// 通过检查 fetcher 的配置来判断其类型
		if fetcher.GetConfig().Type == fetcherType {
			names = append(names, name)
		}
	}
	return names
}
