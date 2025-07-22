package data

import (
	"context"
	"time"
)

// VideoForTrend 定义了用于趋势采集的视频基础信息
type VideoForTrend struct {
	AwemeId      string
	AwemePubTime time.Time `gorm:"type:timestamp"`
}

// FindVideosNeedingTrendUpdate 查找需要更新趋势数据的视频
func (r *videoRepo) FindVideosNeedingTrendUpdate(ctx context.Context, limit int) ([]*VideoForTrend, error) {
	var results []*VideoForTrend
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	oneHourAgo := time.Now().Add(-5 * time.Hour) // 防止任务刚创建就被重复拉取

	// 核心逻辑：筛选需要更新的视频，并排除近期已创建任务的视频
	// details:批量查找数据库中距离上次 trend 数据更新已超过24小时，且最近1小时内未被分配 trend 拉取任务的视频ID，优先返回最久未更新的，供后续采集或更新趋势数据。
	err := r.db.WithContext(ctx).Model(&Video{}).
		Select("aweme_id", "aweme_pub_time").
		Where("trend_updated_at IS NULL OR trend_updated_at < ?", twentyFourHoursAgo).
		Where("NOT EXISTS (SELECT 1 FROM source_data WHERE entity_id = videos.aweme_id AND data_type = 'video_trend' AND status = 0 AND fetched_at > ?)", oneHourAgo).
		Order("trend_updated_at asc NULLS FIRST").
		Limit(limit).
		Find(&results).Error

	return results, err
}
