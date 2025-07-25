package data

import (
	"context"
	"fmt"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm/clause"
)

// Video 视频维度表
type Video struct {
	AwemeId   string    `gorm:"primaryKey;size:1024"`
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp"`

	// --- 基础信息 ---
	AwemeDesc      string    `gorm:"type:text"`
	AwemeCoverUrl  string    `gorm:"size:1024"`
	AwemePubTime   time.Time `gorm:"type:timestamp"`
	AwemeShareUrl  string    `gorm:"size:1024"`
	AwemeDetailUrl string    `gorm:"size:1024"`
	BloggerId      int64     `gorm:"index"`

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
	GoodsId            string `gorm:"size:255;comment:商品ID"`

	// --- 详情信息 (来自下钻采集) 暂时用不上 ---
	DyTagsJSON          string `gorm:"type:text"`
	HotSearchWordsJSON  string `gorm:"type:text"`
	TopicsJSON          string `gorm:"type:text"`
	CommentSegmentsJSON string `gorm:"type:text"`
	InteractionJSON     string `gorm:"type:text"`
	AudienceProfileJSON string `gorm:"type:text"`

	//
	SummaryUpdatedAt *time.Time `gorm:"index;comment:总览数据更新时间;type:timestamp"`
	TrendUpdatedAt   *time.Time `gorm:"index;comment:趋势数据更新时间;type:timestamp"`
	CollectionStatus int32      `gorm:"comment:采集状态"`
}

func (Video) TableName() string {
	return "videos"
}

type VideoRepo interface {
	SaveSourceData(context.Context, *v1.SourceData) (*v1.SourceData, error)
	FindVideosByIDs(ctx context.Context, awemeIDs []string, limit int) ([]*VideoForCollection, error)
	UpsertFromRank(ctx context.Context, video *Video) error
	UpdateFromSummary(ctx context.Context, video *Video) error
	FindVideosNeedingSummaryUpdate(ctx context.Context, limit int) ([]*VideoForSummary, error)
	ListPage(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*Video, int64, error)
	Get(ctx context.Context, awemeId string) (*Video, error)
	FindRecentActiveAwemeIds(ctx context.Context, days int) ([]string, error)
	//FindVideosNeedingTrendUpdate(ctx context.Context, limit int) ([]*VideoForTrend, error)
	FindVideosForDetailsCollection(ctx context.Context, limit int) ([]*VideoForCollection, error)
	UpdateTrendTimestamp(ctx context.Context, awemeId string) error
	FindVideosExcludingIDs(ctx context.Context, ids []string, limit int) ([]*VideoForCollection, error)
	FindPartiallyCollectedVideos(ctx context.Context, hoursAgo int, limit int) ([]*VideoForCollection, error)
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
			"updated_at", "aweme_desc", "aweme_cover_url", "aweme_pub_time", "blogger_id", "goods_id",
		}),
	}).Create(video).Error
}

// UpdateFromSummary 安全地只更新 Video 模型中与总览数据相关的字段
func (r *videoRepo) UpdateFromSummary(ctx context.Context, video *Video) error {
	// 使用 Updates 方法，GORM 将只更新 video 对象中的非零值字段
	return r.db.WithContext(ctx).Model(&Video{AwemeId: video.AwemeId}).Updates(video).Error
}

// ListPage 实现分页、模糊查询和排序
func (r *videoRepo) ListPage(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*Video, int64, error) {
	var videos []*Video
	var total int64

	db := r.db.WithContext(ctx).Model(&Video{})

	// 模糊查询
	if query != "" {
		db = db.Where("aweme_desc LIKE ?", "%"+query+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序逻辑重构
	if sortBy != "" {
		var order string
		switch sortOrder {
		case v1.SortOrder_ASC:
			order = sortBy + " ASC"
		case v1.SortOrder_DESC:
			order = sortBy + " DESC"
		default:
			order = sortBy + " DESC"
		}
		db = db.Order(order)
	} else {
		// 默认排序
		db = db.Order("aweme_pub_time DESC")
	}

	// 分页
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// CopyVideoToDTO 将 data.Video 模型转换为 v1.VideoDTO
func CopyVideoToDTO(v *Video) *v1.VideoDTO {
	if v == nil {
		return nil
	}
	dto := &v1.VideoDTO{
		AwemeId:            v.AwemeId,
		CreatedAt:          v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          v.UpdatedAt.Format(time.RFC3339),
		AwemeDesc:          v.AwemeDesc,
		AwemeCoverUrl:      v.AwemeCoverUrl,
		AwemePubTime:       v.AwemePubTime.Format(time.RFC3339),
		AwemeShareUrl:      v.AwemeShareUrl,
		AwemeDetailUrl:     v.AwemeDetailUrl,
		BloggerId:          v.BloggerId,
		PlayCountStr:       v.PlayCountStr,
		LikeCountStr:       v.LikeCountStr,
		CommentCountStr:    v.CommentCountStr,
		ShareCountStr:      v.ShareCountStr,
		CollectCountStr:    v.CollectCountStr,
		InteractionRateStr: v.InteractionRateStr,
		ScoreStr:           v.ScoreStr,
		LikeCommentRateStr: v.LikeCommentRateStr,
		SalesGmvStr:        v.SalesGmvStr,
		SalesCountStr:      v.SalesCountStr,
		GoodsCountStr:      v.GoodsCountStr,
		GpmStr:             v.GpmStr,
		AwemeType:          v.AwemeType,
	}
	if v.SummaryUpdatedAt != nil {
		dto.SummaryUpdatedAt = v.SummaryUpdatedAt.Format(time.RFC3339)
	}

	dto.GoodsId = v.GoodsId

	return dto
}

func (r *videoRepo) UpdateTrendTimestamp(ctx context.Context, awemeId string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&Video{AwemeId: awemeId}).Update("trend_updated_at", &now).Error
}

func (r *videoRepo) Get(ctx context.Context, awemeId string) (*Video, error) {
	var video Video
	if err := r.db.WithContext(ctx).Where("aweme_id = ?", awemeId).First(&video).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

// FindRecentActiveAwemeIds 查找最近几天内有更新的视频ID
// FindVideosByIDs 根据提供的 aweme_id 列表，批量查询视频信息。
func (r *videoRepo) FindVideosByIDs(ctx context.Context, awemeIDs []string, limit int) ([]*VideoForCollection, error) {
	if len(awemeIDs) == 0 {
		return nil, nil // 如果传入空的ID列表，直接返回空结果，避免数据库查询
	}

	var results []*VideoForCollection

	// 使用 IN 子句进行批量查询
	err := r.db.WithContext(ctx).Model(&Video{}).
		Select("aweme_id", "aweme_pub_time", "aweme_detail_url").
		Where("aweme_id IN ?", awemeIDs).
		Order("updated_at ASC").
		Limit(limit).
		Find(&results).Error

	if err != nil {
		return nil, fmt.Errorf("根据ID列表查询视频失败: %w", err)
	}
	return results, nil
}

func (r *videoRepo) FindRecentActiveAwemeIds(ctx context.Context, days int) ([]string, error) {
	const defaultLimit = 100
	var awemeIds []string
	var videos []*Video
	err := r.db.WithContext(ctx).Model(&Video{}).
		Where("updated_at >= ?", time.Now().AddDate(0, 0, -days)).
		Order("aweme_pub_time DESC").
		Limit(defaultLimit).
		Find(&videos).Error

	if err != nil {
		return nil, err
	}

	for _, v := range videos {
		awemeIds = append(awemeIds, v.AwemeId)
	}

	return awemeIds, nil
}

func (r *videoRepo) SaveSourceData(ctx context.Context, d *v1.SourceData) (*v1.SourceData, error) {
	sd := CopySourceDataToDO(d)
	err := r.Data.db.WithContext(ctx).Create(sd).Error
	if err != nil {
		return nil, err
	}
	return CopySourceDataToDTO(sd), nil
}

func (r *videoRepo) FindVideosExcludingIDs(ctx context.Context, ids []string, limit int) ([]*VideoForCollection, error) {
	panic("implement me")
}

func (r *videoRepo) FindPartiallyCollectedVideos(ctx context.Context, hoursAgo int, limit int) ([]*VideoForCollection, error) {
	since := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)

	var videos []*VideoForCollection
	db := r.Data.db.WithContext(ctx).Model(&Video{}).
		Where("updated_at < ? AND collection_status = ?", since, v1.CollectionStatus_PARTIALLY_COLLECTED)

	err := db.Order("aweme_pub_time DESC").
		Limit(limit).
		Find(&videos).Error

	if err != nil {
		return nil, err
	}

	return videos, nil
}
