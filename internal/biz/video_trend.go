package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
)

// VideoTrendUsecase 封装视频趋势相关的业务逻辑
type VideoTrendUsecase struct {
	repo data.VideoTrendRepo
}

// NewVideoTrendUsecase 构造 VideoTrendUsecase
func NewVideoTrendUsecase(repo data.VideoTrendRepo) *VideoTrendUsecase {
	return &VideoTrendUsecase{repo: repo}
}

// ListVideoTrends 分页查询视频趋势
func (uc *VideoTrendUsecase) ListVideoTrends(ctx context.Context, page, size int, awemeId, startDate, endDate string) ([]*v1.VideoTrendDTO, int64, error) {
	trends, total, err := uc.repo.ListPage(ctx, page, size, awemeId, startDate, endDate)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*v1.VideoTrendDTO, len(trends))
	for i, t := range trends {
		dtos[i] = data.CopyVideoTrendToDTO(t)
	}
	return dtos, total, nil
}
