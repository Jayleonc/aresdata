package fetcher

import (
	"fmt"
	"github.com/Jayleonc/aresdata/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

// FetcherManager holds and manages different fetcher instances.
type FetcherManager struct {
	fetchers map[string]Fetcher
	log      *log.Helper
}

// NewFetcherManager creates and initializes fetchers based on the provided configuration.
func NewFetcherManager(dataSources []*conf.DataSource, logger log.Logger) (*FetcherManager, error) {
	helper := log.NewHelper(log.With(logger, "module", "fetcher/manager"))
	mgr := &FetcherManager{
		fetchers: make(map[string]Fetcher),
		log:      helper,
	}

	helper.Infof("Initializing FetcherManager with %d datasources...", len(dataSources))

	for _, dsConfig := range dataSources {
		helper.Infof("Creating fetcher for datasource: %s (type: %s)", dsConfig.Name, dsConfig.Type)

		if len(dsConfig.AccountPool) == 0 {
			helper.Warnf("Datasource '%s' has an empty account pool. Skipping.", dsConfig.Name)
			continue
		}

		// 1. Create an AccountPool for the current datasource
		accountPool, err := NewAccountPool(dsConfig.AccountPool, logger)
		if err != nil {
			helper.Errorf("failed to create account pool for datasource '%s': %v. Skipping.", dsConfig.Name, err)
			continue
		}

		// 2. Create the appropriate fetcher based on the type
		var fetcher Fetcher
		switch dsConfig.Type {
		case "headless":
			fetcher = NewHeadlessFetcher(dsConfig, accountPool, logger)
			helper.Infof("Initialized 'headless' fetcher: %s", dsConfig.Name)
		case "http":
			fetcher = NewHttpFetcher(dsConfig, accountPool, logger)
			helper.Infof("Initialized 'http' fetcher: %s", dsConfig.Name)
		default:
			helper.Warnf("unsupported fetcher type '%s' for datasource '%s'. Skipping.", dsConfig.Type, dsConfig.Name)
			continue
		}

		mgr.fetchers[dsConfig.Name] = fetcher
	}

	if len(mgr.fetchers) == 0 {
		return nil, fmt.Errorf("no valid fetchers were created from the configuration")
	}

	helper.Info("FetcherManager initialized successfully.")
	return mgr, nil
}

// Get retrieves a fetcher instance by its name.
func (m *FetcherManager) Get(name string) (Fetcher, bool) {
	f, ok := m.fetchers[name]
	if !ok {
		m.log.Warnf("Attempted to get non-existent fetcher: %s", name)
	}
	return f, ok
}

// GetDataSourceNames returns all available data source names
func (m *FetcherManager) GetDataSourceNames() []string {
	names := make([]string, 0, len(m.fetchers))
	for name := range m.fetchers {
		names = append(names, name)
	}
	return names
}
