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

	// --- 新增：总览数据 (来自趋势接口或总量接口的最新一条) ---
	TotalLikes         int64  `gorm:"type:integer"`
	TotalComments      int64  `gorm:"type:integer"`
	TotalShares        int64  `gorm:"type:integer"`
	TotalCollects      int64  `gorm:"type:integer"`
	TotalSalesGmv      int64  `gorm:"type:integer;comment:累计销售额，单位分"`
	TotalSalesVolume   int64  `gorm:"type:integer;comment:累计销量"`
	InteractionRateStr string `gorm:"size:255"`
	GpmStr             string `gorm:"size:255"`

	// --- 详情信息 (来自下钻采集) ---
	DyTagsJSON          string `gorm:"type:text"`
	HotSearchWordsJSON  string `gorm:"type:text"`
	TopicsJSON          string `gorm:"type:text"`
	CommentSegmentsJSON string `gorm:"type:text"`
	InteractionJSON     string `gorm:"type:text"`
	AudienceProfileJSON string `gorm:"type:text"`
}

func (Video) TableName() string {
	return "videos"
}

type VideoRepo interface {
	Upsert(ctx context.Context, video *Video) error
	// FindVideosNeedingSummaryUpdate 查找需要更新总览数据的视频
	FindVideosNeedingSummaryUpdate(ctx context.Context, limit int) ([]*VideoForSummary, error)
}

type videoRepo struct {
	*Data
}

func NewVideoRepo(data *Data) VideoRepo {
	return &videoRepo{Data: data}
}

// Upsert a video record. If the record with the same AwemeId exists, it will be updated.
// Otherwise, a new record will be created.
func (r *videoRepo) Upsert(ctx context.Context, video *Video) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "aweme_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"updated_at",
			"aweme_desc",
			"aweme_cover_url",
			"aweme_pub_time",
			"blogger_id",
			"total_likes",
			"total_comments",
			"total_shares",
			"total_collects",
			"total_sales_gmv",
			"total_sales_volume",
			"interaction_rate_str",
			"gpm_str",
			"dy_tags_json",
			"hot_search_words_json",
			"topics_json",
			"comment_segments_json",
			"interaction_json",
			"audience_profile_json",
		}),
	}).Create(video).Error
}
