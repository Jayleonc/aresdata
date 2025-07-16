package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"aresdata/pkg/fetcher"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type FetcherUsecase struct {
	repo    data.SourceDataRepo
	fetcher *fetcher.FeiguaFetcher
	log     *log.Helper
}

func NewFetcherUsecase(repo data.SourceDataRepo, fetcher *fetcher.FeiguaFetcher, logger log.Logger) *FetcherUsecase {
	return &FetcherUsecase{
		repo:    repo,
		fetcher: fetcher,
		log:     log.NewHelper(log.With(logger, "module", "usecase/fetcher")),
	}
}

// FetchAndStoreVideoRank 是一个具体的业务方法，负责采集视频榜单并存储
func (uc *FetcherUsecase) FetchAndStoreVideoRank(ctx context.Context, period, datecode string) (*v1.SourceData, error) {
	// 1. 调用 Fetcher 获取原始数据
	rawContent, err := uc.fetcher.FetchVideoRank(ctx, period, datecode)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to fetch video rank from feigua: %v", err)
		return nil, err
	}

	uc.log.WithContext(ctx).Infof("Successfully fetched data for period=%s, datecode=%s", period, datecode)

	// 2. 构造 SourceData 对象准备入库
	// 注意：我们仍然将未解密的原始数据存入，将解析工作留给ETL
	var contentData map[string]interface{}
	_ = json.Unmarshal([]byte(rawContent), &contentData)

	sourceData := &v1.SourceData{
		ProviderName: "feigua",
		DataType:     "video_rank_" + period, // 例如: video_rank_day
		RawContent:   rawContent,
		EntityId:     fmt.Sprintf("%s_%s", period, datecode),
		Status:       0, // 初始状态为 未处理
		FetchedAt:    timestamppb.New(time.Now()),
		Date:         datecode,
	}

	// 3. 调用Repo存储到数据库
	return uc.repo.Save(ctx, sourceData)
}
