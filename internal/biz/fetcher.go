package biz

import (
	v1 "aresdata/api/v1"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
)

// SourceDataRepo 是Biz层依赖的Data层接口，由 data/source_data.go 实现
type SourceDataRepo interface {
	Save(context.Context, *v1.SourceData) (*v1.SourceData, error)
}

type FetcherUsecase struct {
	repo    SourceDataRepo
	factory *ProviderFactory
	log     *log.Helper
}

func NewFetcherUsecase(repo SourceDataRepo, factory *ProviderFactory, logger log.Logger) *FetcherUsecase {
	return &FetcherUsecase{
		repo:    repo,
		factory: factory,
		log:     log.NewHelper(log.With(logger, "module", "usecase/fetcher")),
	}
}

// FetchAndStore 是核心业务方法，负责编排整个流程
func (uc *FetcherUsecase) FetchAndStore(ctx context.Context, task *v1.Task) (*v1.SourceData, error) {
	providerName := task.Provider.String()

	// 1. 从工厂获取对应的Provider
	provider, ok := uc.factory.GetProvider(providerName)
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// 2. 调用Provider执行采集
	rawContent, err := provider.Fetch(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// 3. 构造SourceData对象准备入库
	var contentData map[string]interface{}
	_ = json.Unmarshal([]byte(rawContent), &contentData)
	entityID, _ := contentData["id"].(string)

	sourceData := &v1.SourceData{
		ProviderName: providerName,
		DataType:     task.DataType,
		RawContent:   rawContent,
		EntityId:     entityID,
		Status:       0, // 初始状态为 未处理
	}

	// 4. 调用Repo存储到数据库
	return uc.repo.Save(ctx, sourceData)
}
