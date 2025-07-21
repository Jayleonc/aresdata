package data

import (
	"context"
	"time"

	v1 "aresdata/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VideoTrend 视频每日趋势数据模型
type VideoTrend struct {
	Id                 int64     `gorm:"primaryKey"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
	AwemeId            string    `gorm:"size:255;not null;uniqueIndex:uniq_video_trend_aweme_date"` // 视频ID
	DateCode           int       `gorm:"not null;uniqueIndex:uniq_video_trend_aweme_date"`          // 数据日期
	LikeCountStr       string    `gorm:"size:255"`                                                  // 当日总点赞数字符串
	ShareCountStr      string    `gorm:"size:255"`                                                  // 当日总分享数字符串
	CommentCountStr    string    `gorm:"size:255"`                                                  // 当日总评论数字符串
	CollectCountStr    string    `gorm:"size:255"`                                                  // 当日总收藏数字符串
	InteractionRateStr string    `gorm:"size:255"`                                                  // 当日总互动率字符串
	IncLikeCountStr    string    `gorm:"size:255"`                                                  // 当日新增点赞数字符串
	IncShareCountStr   string    `gorm:"size:255"`                                                  // 当日新增分享数字符串
	IncCommentCountStr string    `gorm:"size:255"`                                                  // 当日新增评论数字符串
	IncCollectCountStr string    `gorm:"size:255"`                                                  // 当日新增收藏数字符串
	SalesCountStr      string    `gorm:"size:255"`                                                  // 当日总销量字符串
	SalesGmvStr        string    `gorm:"size:255"`                                                  // 当日总GMV字符串
	FansStr            string    `gorm:"size:255"`                                                  // 当日总粉丝数字符串
	IncSalesCountStr   string    `gorm:"size:255"`                                                  // 当日新增销量字符串
	IncSalesGmvStr     string    `gorm:"size:255"`                                                  // 当日新增GMV字符串
	IncFansStr         string    `gorm:"size:255"`                                                  // 当日新增粉丝数字符串
	GpmStr             string    `gorm:"size:255"`                                                  // 当日总GPM字符串
	ListTimeStr        string    `gorm:"size:255"`                                                  // 原始时间戳字符串
	TimeStamp          int64     // 原始时间戳
}

// VideoTrendRepo 定义视频趋势数据仓库接口
type VideoTrendRepo interface {
	BatchUpsert(ctx context.Context, trends []*VideoTrend) error
}

type videoTrendRepo struct {
	db *gorm.DB
}

// NewVideoTrendRepo .
func NewVideoTrendRepo(db *gorm.DB) VideoTrendRepo {
	return &videoTrendRepo{db: db}
}

// BatchUpsert 批量插入或更新视频趋势数据
// 使用 "ON CONFLICT" 语句确保数据幂等性
func (r *videoTrendRepo) BatchUpsert(ctx context.Context, trends []*VideoTrend) error {
	if len(trends) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "aweme_id"}, {Name: "date_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"updated_at", "like_count_str", "share_count_str", "comment_count_str",
			"collect_count_str", "interaction_rate_str", "inc_like_count_str",
			"inc_share_count_str", "inc_comment_count_str", "inc_collect_count_str",
			"sales_count_str", "sales_gmv_str", "fans_str", "inc_sales_count_str",
			"inc_sales_gmv_str", "inc_fans_str", "gpm_str", "list_time_str", "time_stamp",
		}),
	}).Create(&trends).Error
}

// CopyVideoTrendToDTO 将 data.VideoTrend 模型转换为 v1.VideoTrendDTO
func CopyVideoTrendToDTO(vt *VideoTrend) *v1.VideoTrendDTO {
	if vt == nil {
		return nil
	}
	return &v1.VideoTrendDTO{
		Id:                 vt.Id,
		CreatedAt:          timestamppb.New(vt.CreatedAt),
		UpdatedAt:          timestamppb.New(vt.UpdatedAt),
		AwemeId:            vt.AwemeId,
		DateCode:           int32(vt.DateCode),
		LikeCountStr:       vt.LikeCountStr,
		ShareCountStr:      vt.ShareCountStr,
		CommentCountStr:    vt.CommentCountStr,
		CollectCountStr:    vt.CollectCountStr,
		InteractionRateStr: vt.InteractionRateStr,
		IncLikeCountStr:    vt.IncLikeCountStr,
		IncShareCountStr:   vt.IncShareCountStr,
		IncCommentCountStr: vt.IncCommentCountStr,
		IncCollectCountStr: vt.IncCollectCountStr,
		SalesCountStr:      vt.SalesCountStr,
		SalesGmvStr:        vt.SalesGmvStr,
		FansStr:            vt.FansStr,
		IncSalesCountStr:   vt.IncSalesCountStr,
		IncSalesGmvStr:     vt.IncSalesGmvStr,
		IncFansStr:         vt.IncFansStr,
		GpmStr:             vt.GpmStr,
		ListTimeStr:        vt.ListTimeStr,
		TimeStamp:          vt.TimeStamp,
	}
}
