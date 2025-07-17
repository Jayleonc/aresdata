package data

import (
	"context"
	"gorm.io/gorm/clause"
	"time"
)

// VideoTrendStat 记录视频每日的数据快照
// 表名: video_trend_stats
// 用于存储视频每日的累计数据快照，便于趋势分析
type VideoTrendStat struct {
	AwemeId   string    `gorm:"primaryKey;size:1024"` // 视频ID，关联到 videos 表
	Date      time.Time `gorm:"primaryKey;type:date"` // 数据日期
	CreatedAt time.Time `gorm:"autoCreateTime"`       // 创建时间
	UpdatedAt time.Time `gorm:"autoUpdateTime"`       // 更新时间

	TotalLikes        int64   // 截止到当天的总点赞数
	TotalComments     int64   // 截止到当天的总评论数
	TotalShares       int64   // 截止到当天的总分享数
	TotalCollects     int64   // 截止到当天的总收藏数
	TotalSalesGmv     float64 // 截止到当天的累计销售额
	TotalSalesVolume  int64   // 截止到当天的累计销量
	BloggerFansAtDate int64   // 当天博主的粉丝数
}

func (VideoTrendStat) TableName() string {
	return "video_trend_stats" // 表名也更新
}

type VideoTrendStatRepo interface {
	BatchUpsert(ctx context.Context, stats []*VideoTrendStat) error
}

type videoTrendStatRepo struct {
	*Data
}

func NewVideoTrendStatRepo(data *Data) VideoTrendStatRepo {
	return &videoTrendStatRepo{Data: data}
}

func (r *videoTrendStatRepo) BatchUpsert(ctx context.Context, stats []*VideoTrendStat) error {
	if len(stats) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "aweme_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"updated_at",
			"total_likes",
			"total_comments",
			"total_shares",
			"total_collects",
			"total_sales_gmv",
			"total_sales_volume",
			"blogger_fans_at_date",
		}),
	}).Create(&stats).Error
}
