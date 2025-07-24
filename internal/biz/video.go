package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
	"fmt"
	"time"
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

// GetVideosForFirstCollection 获取用于首次详情采集的视频列表。
// 【最终正确版】：在Biz层进行逻辑编排，实现跨Repo查询。
func (uc *VideoUsecase) GetVideosForFirstCollection(ctx context.Context, limit int) ([]*data.VideoForCollection, error) {
	dataTypes := []string{"video_trend_headless", "video_summary_headless"}

	// 步骤1：调用 sourceDataRepo，拿到所有“已碰过”的视频ID黑名单
	excludeIDs, err := uc.sourceDataRepo.ListAllCollectedEntityIDs(ctx, dataTypes)
	if err != nil {
		return nil, fmt.Errorf("获取已采集ID黑名单失败: %w", err)
	}

	// 步骤2：调用 videoRepo，获取所有不在黑名单里的视频
	return uc.repo.FindVideosExcludingIDs(ctx, excludeIDs, limit)
}

// 【命名修正】
// GetVideosByTimeWindow 获取在24小时时间窗口内需要更新的视频 (旧逻辑)
func (uc *VideoUsecase) GetVideosByTimeWindow(ctx context.Context, limit int) ([]*data.VideoForCollection, error) {
	// 这个方法现在清晰地指向了那个基于时间的查询逻辑，如果其他地方仍有依赖，也不会出错
	return uc.repo.FindVideosForDetailsCollection(ctx, limit)
}

// GetPartiallyCollectedVideos 获取部分采集失败的视频列表，用于修复任务
func (uc *VideoUsecase) GetPartiallyCollectedVideos(ctx context.Context, hoursAgo int, limit int) ([]*data.VideoForCollection, error) {
	since := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)
	dataTypes := []string{"video_trend_headless", "video_summary_headless"}

	// 步骤1：调用 sourceDataRepo，获取“成功了一半”的视频ID列表
	partiallyCollectedIDs, err := uc.sourceDataRepo.FindPartiallyCollectedEntityIDs(ctx, since, dataTypes)
	if err != nil {
		return nil, fmt.Errorf("获取部分采集的ID列表失败: %w", err)
	}

	if len(partiallyCollectedIDs) == 0 {
		return nil, nil // 没有需要修复的，直接返回
	}

	// 步骤2：调用 videoRepo，用上一步获取的ID列表查询完整的视频信息
	return uc.repo.FindVideosByIDs(ctx, partiallyCollectedIDs, limit)
}
