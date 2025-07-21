package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
)

// VideoUsecase 封装视频维度相关的业务逻辑
type VideoUsecase struct {
	repo data.VideoRepo
}

// GetVideo retrieves a single video by its ID.
func (uc *VideoUsecase) GetVideo(ctx context.Context, awemeId string) (*v1.VideoDTO, error) {
	video, err := uc.repo.Get(ctx, awemeId)
	if err != nil {
		return nil, err
	}
	return data.CopyVideoToDTO(video), nil
}

// NewVideoUsecase 构造 VideoUsecase
func NewVideoUsecase(repo data.VideoRepo) *VideoUsecase {
	return &VideoUsecase{repo: repo}
}

// ListVideos 分页查询视频
func (uc *VideoUsecase) ListVideos(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*v1.VideoDTO, int64, error) {
	videos, total, err := uc.repo.ListPage(ctx, page, size, query, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*v1.VideoDTO, len(videos))
	for i, v := range videos {
		dtos[i] = data.CopyVideoToDTO(v)
	}
	return dtos, total, nil
}
