package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
)

// VideoRankUsecase 封装视频榜单相关的业务逻辑
// 依赖 data.VideoRankRepo
type VideoRankUsecase struct {
	repo data.VideoRankRepo
}

// NewVideoRankUsecase 构造 VideoRankUsecase
func NewVideoRankUsecase(repo data.VideoRankRepo) *VideoRankUsecase {
	return &VideoRankUsecase{repo: repo}
}

// GetVideoRank 查询单个视频榜单
func (uc *VideoRankUsecase) GetVideoRank(ctx context.Context, awemeID, rankType, rankDate string) (*v1.VideoRankDTO, error) {
	return uc.repo.GetByAwemeID(ctx, awemeID, rankType, rankDate)
}

// ListVideoRank 分页查询视频榜单
func (uc *VideoRankUsecase) ListVideoRank(ctx context.Context, page, size int, rankType, rankDate string) ([]*v1.VideoRankDTO, int64, error) {
	return uc.repo.ListPage(ctx, page, size, rankType, rankDate)
}
