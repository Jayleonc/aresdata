package data

import (
	"context"
	"fmt"
	"time"
)

// VideoForCollection 是为采集任务准备的结构体
type VideoForCollection struct {
	AwemeId        string
	AwemePubTime   time.Time
	AwemeDetailUrl string
}

// FindVideosForDetailsCollection 查找需要被采集详情的视频。
// 此版本逻辑已最终修正，完全基于“近期不存在采集记录”来判断，与status和ETL完全解耦。
func (r *videoRepo) FindVideosForDetailsCollection(ctx context.Context, limit int) ([]*VideoForCollection, error) {
	var results []*VideoForCollection
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	// 子查询: 找到最近24小时内，已经【尝试采集过】任何详情数据（trend或summary）的视频ID。
	// 我们不关心它的 status，只要存在记录，就意味着近期处理过。
	subQueryRecent := r.db.Model(&SourceData{}).
		Select("DISTINCT entity_id").
		Where("data_type IN ? AND created_at > ?", []string{"video_trend_headless", "video_summary_headless"}, twentyFourHoursAgo)

	// 【最终的、正确的查询逻辑】
	err := r.db.WithContext(ctx).Model(&Video{}).
		Select("aweme_id", "aweme_pub_time", "aweme_detail_url").

		// 条件: 视频ID 不在 “近期已采集” 的列表里
		Where("aweme_id NOT IN (?)", subQueryRecent).

		// 排序: 优先采集发布时间最新的视频，确保我们总是在跟进新内容
		Order("aweme_pub_time DESC").
		Limit(limit).
		Find(&results).Error

	if err != nil {
		return nil, fmt.Errorf("查询待采集详情视频失败: %w", err)
	}
	return results, nil
}

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
