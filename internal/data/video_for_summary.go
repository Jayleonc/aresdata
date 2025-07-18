package data

import (
	"context"
	"time"
)

// VideoForSummary 定义了用于下钻采集的视频基础信息
type VideoForSummary struct {
	AwemeId      string
	AwemePubTime time.Time
}

// FindVideosNeedingSummaryUpdate 查找需要更新总览数据的视频
func (r *videoRepo) FindVideosNeedingSummaryUpdate(ctx context.Context, limit int) ([]*VideoForSummary, error) {
	var results []*VideoForSummary
	// 以 total_likes = 0 作为需要更新的标志，优先处理最旧的记录
	err := r.db.WithContext(ctx).Model(&Video{}).
		Select("aweme_id", "aweme_pub_time").
		Where("total_likes = ?", 0).
		Order("updated_at asc").
		Limit(limit).
		Find(&results).Error
	return results, err
}
