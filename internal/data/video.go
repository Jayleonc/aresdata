package data

import (
	"context"
	"gorm.io/gorm/clause"
	"time"
)

// Video 视频维度表
type Video struct {
	AwemeId   string    `gorm:"primaryKey;size:1024"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// --- 基础信息 ---
	AwemeDesc     string `gorm:"type:text"`
	AwemeCoverUrl string `gorm:"size:1024"`
	AwemePubTime  time.Time
	BloggerId     int64 `gorm:"index"`

	// --- 新增：总览数据 (来自总量接口的原始字符串) ---
	PlayCountStr       string `gorm:"size:255"`
	LikeCountStr       string `gorm:"size:255"`
	CommentCountStr    string `gorm:"size:255"`
	ShareCountStr      string `gorm:"size:255"`
	CollectCountStr    string `gorm:"size:255"`
	InteractionRateStr string `gorm:"size:255"`
	ScoreStr           string `gorm:"size:255;comment:视频分数"`
	LikeCommentRateStr string `gorm:"size:255"`
	SalesGmvStr        string `gorm:"size:255"`
	SalesCountStr      string `gorm:"size:255"`
	GoodsCountStr      string `gorm:"size:255"`
	GpmStr             string `gorm:"size:255;column:gpm_str"` // 明确指定列名
	AwemeType          int32  `gorm:"type:integer"`

	// --- 详情信息 (来自下钻采集) ---
	DyTagsJSON          string `gorm:"type:text"`
	HotSearchWordsJSON  string `gorm:"type:text"`
	TopicsJSON          string `gorm:"type:text"`
	CommentSegmentsJSON string `gorm:"type:text"`
	InteractionJSON     string `gorm:"type:text"`
	AudienceProfileJSON string `gorm:"type:text"`

	SummaryUpdatedAt *time.Time `gorm:"index;comment:总览数据更新时间"`
}

func (Video) TableName() string {
	return "videos"
}

type VideoRepo interface {
	UpsertFromRank(ctx context.Context, video *Video) error    // 新方法
	UpdateFromSummary(ctx context.Context, video *Video) error // 新方法
	FindVideosNeedingSummaryUpdate(ctx context.Context, limit int) ([]*VideoForSummary, error)
}

type videoRepo struct {
	*Data
}

func NewVideoRepo(data *Data) VideoRepo {
	return &videoRepo{Data: data}
}

// UpsertFromRank 安全地创建或更新来自榜单数据的视频基础信息
func (r *videoRepo) UpsertFromRank(ctx context.Context, video *Video) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "aweme_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"updated_at", "aweme_desc", "aweme_cover_url", "aweme_pub_time", "blogger_id",
		}),
	}).Create(video).Error
}

// UpdateFromSummary 安全地只更新 Video 模型中与总览数据相关的字段
func (r *videoRepo) UpdateFromSummary(ctx context.Context, video *Video) error {
	// 使用 Updates 方法，GORM 将只更新 video 对象中的非零值字段
	return r.db.WithContext(ctx).Model(&Video{AwemeId: video.AwemeId}).Updates(video).Error
}
