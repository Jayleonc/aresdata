package data

import (
	"context"
	"time"

	v1 "aresdata/api/v1"
	"gorm.io/gorm/clause"
)

// VideoTrend 视频每日趋势数据模型
type VideoTrend struct {
	Id                 int64     `gorm:"primaryKey"`
	CreatedAt          time.Time `gorm:"autoCreateTime;type:timestamp"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime;type:timestamp"`
	AwemeId            string    `gorm:"size:255;not null;uniqueIndex:uniq_video_trend_aweme_date"` // 视频ID
	DateCode           int       `gorm:"not null;uniqueIndex:uniq_video_trend_aweme_date"`          // 数据日期
	LikeCount          int64
	LikeCountStr       string `gorm:"size:20"`
	ShareCount         int64
	ShareCountStr      string `gorm:"size:20"`
	CommentCount       int64
	CommentCountStr    string `gorm:"size:20"`
	CollectCount       int64
	CollectCountStr    string `gorm:"size:20"`
	InteractionRate    float64
	InteractionRateStr string `gorm:"size:20"`
	IncLikeCount       int64
	IncLikeCountStr    string `gorm:"size:20"`
	IncShareCount      int64
	IncShareCountStr   string `gorm:"size:20"`
	IncCommentCount    int64
	IncCommentCountStr string `gorm:"size:20"`
	IncCollectCount    int64
	IncCollectCountStr string `gorm:"size:20"`
	SalesCount         int64
	SalesCountStr      string `gorm:"size:20"`
	SalesGmv           float64
	SalesGmvStr        string `gorm:"size:20"`
	Fans               int64
	FansStr            string `gorm:"size:20"`
	IncSalesCount      int64
	IncSalesCountStr   string `gorm:"size:20"`
	IncSalesGmv        float64
	IncSalesGmvStr     string `gorm:"size:20"`
	IncFans            int64
	IncFansStr         string `gorm:"size:20"`
	Gpm                float64
	GpmStr             string `gorm:"size:20"`
	ListTimeStr        string `gorm:"size:20"`
	TimeStamp          int64
}

// VideoTrendRepo 定义视频趋势数据仓库接口
type VideoTrendRepo interface {
	BatchUpsert(ctx context.Context, trends []*VideoTrend) error
	ListPage(ctx context.Context, page, size int, awemeId, startDate, endDate string) ([]*VideoTrend, int64, error)
}

type videoTrendRepo struct {
	*Data
}

// NewVideoTrendRepo .
func NewVideoTrendRepo(data *Data) VideoTrendRepo {
	return &videoTrendRepo{Data: data}
}

// BatchUpsert 批量插入或更新视频趋势数据
func (r *videoTrendRepo) BatchUpsert(ctx context.Context, trends []*VideoTrend) error {
	if len(trends) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "aweme_id"}, {Name: "date_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"updated_at", "like_count", "like_count_str", "share_count", "share_count_str",
			"comment_count", "comment_count_str", "collect_count", "collect_count_str",
			"interaction_rate", "interaction_rate_str", "inc_like_count", "inc_like_count_str",
			"inc_share_count", "inc_share_count_str", "inc_comment_count", "inc_comment_count_str",
			"inc_collect_count", "inc_collect_count_str", "sales_count", "sales_count_str",
			"sales_gmv", "sales_gmv_str", "fans", "fans_str", "inc_sales_count", "inc_sales_count_str",
			"inc_sales_gmv", "inc_sales_gmv_str", "inc_fans", "inc_fans_str", "gpm", "gpm_str",
			"list_time_str", "time_stamp",
		}),
	}).Create(&trends).Error
}

// ListPage 分页查询视频趋势数据
func (r *videoTrendRepo) ListPage(ctx context.Context, page, size int, awemeId, startDate, endDate string) ([]*VideoTrend, int64, error) {
	var trends []*VideoTrend
	var total int64

	db := r.db.WithContext(ctx).Model(&VideoTrend{})

	// 应用筛选条件
	if awemeId != "" {
		db = db.Where("aweme_id = ?", awemeId)
	}
	if startDate != "" {
		db = db.Where("date_code >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("date_code <= ?", endDate)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 默认按日期升序排序
	db = db.Order("date_code ASC")

	// 分页
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Find(&trends).Error; err != nil {
		return nil, 0, err
	}

	return trends, total, nil
}

// CopyVideoTrendToDTO 将 data.VideoTrend 模型转换为 v1.VideoTrendDTO
func CopyVideoTrendToDTO(vt *VideoTrend) *v1.VideoTrendDTO {
	if vt == nil {
		return nil
	}
	return &v1.VideoTrendDTO{
		Id:                 vt.Id,
		CreatedAt:          vt.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          vt.UpdatedAt.Format(time.RFC3339),
		AwemeId:            vt.AwemeId,
		DateCode:           int32(vt.DateCode),
		LikeCount:          vt.LikeCount,
		LikeCountStr:       vt.LikeCountStr,
		ShareCount:         vt.ShareCount,
		ShareCountStr:      vt.ShareCountStr,
		CommentCount:       vt.CommentCount,
		CommentCountStr:    vt.CommentCountStr,
		CollectCount:       vt.CollectCount,
		CollectCountStr:    vt.CollectCountStr,
		InteractionRate:    vt.InteractionRate,
		InteractionRateStr: vt.InteractionRateStr,
		IncLikeCount:       vt.IncLikeCount,
		IncLikeCountStr:    vt.IncLikeCountStr,
		IncShareCount:      vt.IncShareCount,
		IncShareCountStr:   vt.IncShareCountStr,
		IncCommentCount:    vt.IncCommentCount,
		IncCommentCountStr: vt.IncCommentCountStr,
		IncCollectCount:    vt.IncCollectCount,
		IncCollectCountStr: vt.IncCollectCountStr,
		SalesCount:         vt.SalesCount,
		SalesCountStr:      vt.SalesCountStr,
		SalesGmv:           vt.SalesGmv,
		SalesGmvStr:        vt.SalesGmvStr,
		Fans:               vt.Fans,
		FansStr:            vt.FansStr,
		IncSalesCount:      vt.IncSalesCount,
		IncSalesCountStr:   vt.IncSalesCountStr,
		IncSalesGmv:        vt.IncSalesGmv,
		IncSalesGmvStr:     vt.IncSalesGmvStr,
		IncFans:            vt.IncFans,
		IncFansStr:         vt.IncFansStr,
		Gpm:                vt.Gpm,
		GpmStr:             vt.GpmStr,
		ListTimeStr:        vt.ListTimeStr,
		TimeStamp:          vt.TimeStamp,
	}
}
