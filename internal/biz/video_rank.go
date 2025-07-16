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

// BatchGetVideoRank 批量查询视频榜单
func (uc *VideoRankUsecase) BatchGetVideoRank(ctx context.Context, awemeIDs []string, rankType, rankDate string) ([]*v1.VideoRankDTO, error) {
	return uc.repo.BatchGetByAwemeIDs(ctx, awemeIDs, rankType, rankDate)
}
