package biz

import (
	"context"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/data"
)

// VideoUsecase 封装视频维度相关的业务逻辑
type VideoUsecase struct {
	repo           data.VideoRepo
	sourceDataRepo data.SourceDataRepo // <-- 注入 SourceDataRepo
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
func NewVideoUsecase(repo data.VideoRepo, sourceDataRepo data.SourceDataRepo) *VideoUsecase {
	return &VideoUsecase{
		repo:           repo,
		sourceDataRepo: sourceDataRepo,
	}
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

// GetVideosByTimeWindow 获取在24小时时间窗口内需要更新的视频 (旧逻辑)
func (uc *VideoUsecase) GetVideosByTimeWindow(ctx context.Context, limit int) ([]*data.VideoForCollection, error) {
	// 这个方法现在清晰地指向了那个基于时间的查询逻辑，如果其他地方仍有依赖，也不会出错
	return uc.repo.FindVideosForDetailsCollection(ctx, limit)
}
