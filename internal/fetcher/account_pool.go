package fetcher

import (
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"net/http"
	"os"
	"sync"
)

// Account represents a single account's cookies.
type Account struct {
	Cookies []*http.Cookie
	path    string
}

// AccountPool manages a collection of accounts for rotation.
type AccountPool struct {
	accounts []*Account
	current  int
	mu       sync.Mutex
	log      *log.Helper
}

// NewAccountPool creates a new AccountPool from a list of cookie file paths.
func NewAccountPool(cookiePaths []string, logger log.Logger) (*AccountPool, error) {
	helper := log.NewHelper(log.With(logger, "module", "fetcher/account-pool"))
	pool := &AccountPool{
		current: 0,
		log:     helper,
	}

	for _, path := range cookiePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			helper.Errorf("failed to read cookie file %s: %v", path, err)
			continue // Skip faulty files
		}

		var cookies []*http.Cookie
		if err := json.Unmarshal(data, &cookies); err != nil {
			helper.Errorf("failed to unmarshal cookie file %s: %v", path, err)
			continue
		}

		pool.accounts = append(pool.accounts, &Account{Cookies: cookies, path: path})
		helper.Infof("Successfully loaded account from %s", path)
	}

	if len(pool.accounts) == 0 {
		return nil, fmt.Errorf("no valid accounts loaded from paths: %v", cookiePaths)
	}

	return pool, nil
}

// GetNextAccount retrieves the next account from the pool in a round-robin fashion.
func (p *AccountPool) GetNextAccount() *Account {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.accounts) == 0 {
		p.log.Warn("GetNextAccount called on an empty account pool")
		return nil
	}

	// 主动轮换逻辑
	account := p.accounts[p.current]
	p.log.Infof("派发账户: %s", account.path)
	p.current = (p.current + 1) % len(p.accounts)
	return account
}
