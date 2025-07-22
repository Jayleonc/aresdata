package data

import (
	"context"
	"time"
)

// VideoForSummary 定义了用于下钻采集的视频基础信息
type VideoForSummary struct {
	AwemeId      string
	AwemePubTime time.Time `gorm:"type:timestamp"`
}

// FindVideosNeedingSummaryUpdate 查找需要更新总览数据的视频
func (r *videoRepo) FindVideosNeedingSummaryUpdate(ctx context.Context, limit int) ([]*VideoForSummary, error) {
	var results []*VideoForSummary
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	oneHourAgo := time.Now().Add(-1 * time.Hour) // 防止任务刚创建就被重复拉取

	// 核心优化：增加 NOT EXISTS 子句，排除已在 source_data 中等待处理的任务
	err := r.db.WithContext(ctx).Model(&Video{}).
		Select("aweme_id", "aweme_pub_time").
		Where("summary_updated_at IS NULL OR summary_updated_at < ?", twentyFourHoursAgo).
		// 数据库在一次查询中，利用其强大的查询优化器，高效地完成 videos 表的筛选和与 source_data 表的关联子查询，直接返回最终的、精确的100个视频ID。
		Where("NOT EXISTS (SELECT 1 FROM source_data WHERE entity_id = videos.aweme_id AND data_type = 'video_summary' AND status = 0 AND fetched_at > ?)", oneHourAgo).
		Order("summary_updated_at asc NULLS FIRST").
		Limit(limit).
		Find(&results).Error
	return results, err
}
