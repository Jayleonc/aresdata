package data

import (
	"context"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/pkg/utils"
	"strings"
	"time"
)

// VideoRankRepo defines the interface for batch creation of VideoRank records.
// VideoRankRepo 定义了视频榜单数据的持久化接口。
type VideoRankRepo interface {
	// 批量创建视频榜单记录
	BatchCreate(ctx context.Context, ranks []*v1.VideoRankDTO) error
	// 查询单个视频榜单
	GetByAwemeID(ctx context.Context, awemeID, rankType, rankDate string) (*v1.VideoRankDTO, error)
	// 分页查询视频榜单
	ListPage(ctx context.Context, page, size int, rankType, rankDate, sortBy, sortOrder string) ([]*v1.VideoRankDTO, int64, error)
	// GetDistinctAwemeIDsByDate 获取指定日期之后上过榜的、不重复的视频ID
	GetDistinctAwemeIDsByDate(ctx context.Context, sinceDate string) ([]string, error)
}

// VideoRank is the GORM model for storing video ranking data.
type VideoRank struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp"`

	// 榜单核心

	PeriodType string `gorm:"column:period_type;size:1024;not null"`
	RankDate   string `gorm:"column:rank_date;size:1024;not null"`
	StartDate  string `gorm:"column:start_date;size:1024;not null;default:''"`
	EndDate    string `gorm:"column:end_date;size:1024;not null;default:''"`

	// 视频信息
	AwemeId        string    `gorm:"column:aweme_id;size:1024;not null"`
	AwemeCoverUrl  string    `gorm:"column:aweme_cover_url;size:1024;not null;default:''"`
	AwemeDesc      string    `gorm:"column:aweme_desc;type:text;not null;default:''"`
	AwemePubTime   time.Time `gorm:"column:aweme_pub_time;type:timestamp"`
	AwemeShareUrl  string    `gorm:"column:aweme_share_url;size:1024;not null;default:''"`
	DurationStr    string    `gorm:"column:duration_str;size:1024;not null;default:''"`
	AwemeScoreStr  string    `gorm:"column:aweme_score_str;size:1024;not null;default:''"`
	AwemeDetailUrl string    `gorm:"column:aweme_detail_url;size:1024;not null;default:''"`

	// 商品信息
	GoodsId         string  `gorm:"column:goods_id;size:1024;not null"`
	GoodsTitle      string  `gorm:"column:goods_title;type:text;not null;default:''"`
	GoodsCoverUrl   string  `gorm:"column:goods_cover_url;size:1024;not null;default:''"`
	GoodsPriceRange string  `gorm:"column:goods_price_range;size:1024;not null;default:''"`
	GoodsPrice      float64 `gorm:"column:goods_price"`
	CosRatio        string  `gorm:"column:cos_ratio;size:1024;not null;default:''"`
	CommissionPrice string  `gorm:"column:commission_price;size:1024;not null;default:''"`
	ShopName        string  `gorm:"column:shop_name;size:1024;not null;default:''"`
	BrandName       string  `gorm:"column:brand_name;size:1024;not null;default:''"`
	CategoryNames   string  `gorm:"column:category_names;size:1024;not null;default:''"`

	// 博主信息
	BloggerId      int    `gorm:"column:blogger_id;not null"`
	BloggerUid     string `gorm:"column:blogger_uid;size:1024;not null;default:''"`
	BloggerName    string `gorm:"column:blogger_name;size:1024;not null;default:''"`
	BloggerAvatar  string `gorm:"column:blogger_avatar;size:1024;not null;default:''"`
	BloggerFansNum int    `gorm:"column:blogger_fans_num;not null;default:0"`
	BloggerTag     string `gorm:"column:blogger_tag;size:1024;not null;default:''"`

	// 榜单统计
	SalesCountStr   string `gorm:"column:sales_count_str;size:1024;not null;default:''"`
	TotalSalesStr   string `gorm:"column:total_sales_str;size:1024;not null;default:''"`
	LikeCountIncStr string `gorm:"column:like_count_inc_str;size:1024;not null;default:''"`
	PlayCountIncStr string `gorm:"column:play_count_inc_str;size:1024;not null;default:''"`

	SalesCountLow  int64 `gorm:"column:sales_count_low;comment:销量范围低值"`
	SalesCountHigh int64 `gorm:"column:sales_count_high;comment:销量范围高值"`
	TotalSalesLow  int64 `gorm:"column:total_sales_low;comment:销售额范围低值（分）"`
	TotalSalesHigh int64 `gorm:"column:total_sales_high;comment:销售额范围高值（分）"`
}

// videoRankRepo implements VideoRankRepo using GORM.

func (r *videoRankRepo) GetDistinctAwemeIDsByDate(ctx context.Context, sinceDate string) ([]string, error) {
	var awemeIDs []string
	err := r.db.WithContext(ctx).
		Model(&VideoRank{}).
		Distinct("aweme_id").
		Where("rank_date >= ?", sinceDate).
		Pluck("aweme_id", &awemeIDs).Error
	if err != nil {
		return nil, err
	}
	return awemeIDs, nil
}

type videoRankRepo struct {
	*Data
}

// GetByAwemeID 查询单个视频榜单
func (r *videoRankRepo) GetByAwemeID(ctx context.Context, awemeID, rankType, rankDate string) (*v1.VideoRankDTO, error) {
	var model VideoRank
	err := r.db.WithContext(ctx).Where("aweme_id = ? AND period_type = ? AND rank_date = ?", awemeID, rankType, rankDate).First(&model).Error
	if err != nil {
		return nil, err
	}
	return CopyVideoRankToDTO(&model), nil
}

// ListPage 分页查询视频榜单
func (r *videoRankRepo) ListPage(ctx context.Context, page, size int, rankType, rankDate, sortBy, sortOrder string) ([]*v1.VideoRankDTO, int64, error) {
	var models []*VideoRank
	var total int64

	db := r.db.WithContext(ctx).Model(&VideoRank{})

	// 添加查询条件
	if rankType != "" {
		db = db.Where("period_type = ?", rankType)
	}
	if rankDate != "" {
		db = db.Where("rank_date = ?", rankDate)
	}

	// 应用排序逻辑
	if sortBy != "" {
		var orderColumn string
		switch sortBy {
		case "salesCountStr":
			orderColumn = "sales_count_high" // 按销量范围最高值排序
		case "totalSalesStr":
			orderColumn = "total_sales_high" // 按销售额范围最高值排序
		// 可以根据需要添加其他排序字段
		default:
			orderColumn = "" // 对于未知的排序字段，忽略排序
		}

		if orderColumn != "" {
			// 根据 sort_order 参数决定升序或降序
			if strings.ToLower(sortOrder) == "desc" {
				db = db.Order(orderColumn + " DESC")
			} else {
				db = db.Order(orderColumn + " ASC")
			}
		}
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*v1.VideoRankDTO, 0, len(models))
	for _, m := range models {
		result = append(result, CopyVideoRankToDTO(m))
	}

	return result, total, nil
}

// BatchCreate inserts multiple VideoRank records into the database.
func (r *videoRankRepo) BatchCreate(ctx context.Context, ranks []*v1.VideoRankDTO) error {
	var models []*VideoRank
	for _, rank := range ranks {
		models = append(models, copyVideoRankToDO(rank))
	}
	return r.db.WithContext(ctx).CreateInBatches(models, len(models)).Error
}

// NewVideoRankRepo creates a new VideoRankRepo.
func NewVideoRankRepo(db *Data) VideoRankRepo {
	return &videoRankRepo{Data: db}
}

func copyVideoRankToDO(dto *v1.VideoRankDTO) *VideoRank {
	if dto == nil {
		return nil
	}
	return &VideoRank{
		PeriodType:      dto.PeriodType,
		RankDate:        dto.RankDate,
		StartDate:       dto.StartDate,
		EndDate:         dto.EndDate,
		AwemeId:         dto.AwemeId,
		AwemeCoverUrl:   dto.AwemeCoverUrl,
		AwemeDesc:       dto.AwemeDesc,
		AwemePubTime:    utils.ParseTimeRFC3339(dto.AwemePubTime),
		AwemeShareUrl:   dto.AwemeShareUrl,
		AwemeDetailUrl:  dto.AwemeDetailUrl,
		DurationStr:     dto.DurationStr,
		AwemeScoreStr:   dto.AwemeScoreStr,
		GoodsId:         dto.GoodsId,
		GoodsTitle:      dto.GoodsTitle,
		GoodsCoverUrl:   dto.GoodsCoverUrl,
		GoodsPriceRange: dto.GoodsPriceRange,
		GoodsPrice:      dto.GoodsPrice,
		CosRatio:        dto.CosRatio,
		CommissionPrice: dto.CommissionPrice,
		ShopName:        dto.ShopName,
		BrandName:       dto.BrandName,
		CategoryNames:   dto.CategoryNames,
		BloggerId:       int(dto.BloggerId),
		BloggerUid:      dto.BloggerUid,
		BloggerName:     dto.BloggerName,
		BloggerAvatar:   dto.BloggerAvatar,
		BloggerFansNum:  int(dto.BloggerFansNum),
		BloggerTag:      dto.BloggerTag,
		SalesCountStr:   dto.SalesCountStr,
		TotalSalesStr:   dto.TotalSalesStr,
		LikeCountIncStr: dto.LikeCountIncStr,
		PlayCountIncStr: dto.PlayCountIncStr,
		SalesCountLow:   dto.SalesCountLow,
		SalesCountHigh:  dto.SalesCountHigh,
		TotalSalesLow:   dto.TotalSalesLow,
		TotalSalesHigh:  dto.TotalSalesHigh,
	}
}

func CopyVideoRankToDTO(do *VideoRank) *v1.VideoRankDTO {
	if do == nil {
		return nil
	}
	return &v1.VideoRankDTO{
		Id: int64(do.ID),

		PeriodType:      do.PeriodType,
		RankDate:        do.RankDate,
		StartDate:       do.StartDate,
		EndDate:         do.EndDate,
		AwemeId:         do.AwemeId,
		AwemeCoverUrl:   do.AwemeCoverUrl,
		AwemeDesc:       do.AwemeDesc,
		AwemePubTime:    do.AwemePubTime.Format(time.RFC3339),
		AwemeShareUrl:   do.AwemeShareUrl,
		AwemeDetailUrl:  do.AwemeDetailUrl,
		DurationStr:     do.DurationStr,
		AwemeScoreStr:   do.AwemeScoreStr,
		GoodsId:         do.GoodsId,
		GoodsTitle:      do.GoodsTitle,
		GoodsCoverUrl:   do.GoodsCoverUrl,
		GoodsPriceRange: do.GoodsPriceRange,
		GoodsPrice:      do.GoodsPrice,
		CosRatio:        do.CosRatio,
		CommissionPrice: do.CommissionPrice,
		ShopName:        do.ShopName,
		BrandName:       do.BrandName,
		CategoryNames:   do.CategoryNames,
		BloggerId:       int32(do.BloggerId),
		BloggerUid:      do.BloggerUid,
		BloggerName:     do.BloggerName,
		BloggerAvatar:   do.BloggerAvatar,
		BloggerFansNum:  int32(do.BloggerFansNum),
		BloggerTag:      do.BloggerTag,
		SalesCountStr:   do.SalesCountStr,
		TotalSalesStr:   do.TotalSalesStr,
		LikeCountIncStr: do.LikeCountIncStr,
		PlayCountIncStr: do.PlayCountIncStr,
		SalesCountLow:   do.SalesCountLow,
		SalesCountHigh:  do.SalesCountHigh,
		TotalSalesLow:   do.TotalSalesLow,
		TotalSalesHigh:  do.TotalSalesHigh,
	}
}
