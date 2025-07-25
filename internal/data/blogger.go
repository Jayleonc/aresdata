package data

import (
	"context"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"gorm.io/gorm/clause"
	"time"
)

// Blogger 博主维度表
// Blogger 博主维度表
type Blogger struct {
	BloggerId      int64     `gorm:"primaryKey"`
	CreatedAt      time.Time `gorm:"autoCreateTime;type:timestamp"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime;type:timestamp"`
	BloggerUid     string    `gorm:"size:255;index"`
	BloggerName    string    `gorm:"size:255"`
	BloggerAvatar  string    `gorm:"size:1024"`
	BloggerFansNum int64
	BloggerTag     string `gorm:"size:255"`
}

func (Blogger) TableName() string {
	return "bloggers"
}

type BloggerRepo interface {
	Upsert(ctx context.Context, blogger *Blogger) error
	ListPage(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*Blogger, int64, error)
	Get(ctx context.Context, bloggerId int64) (*Blogger, error) // 新增此行
}

type bloggerRepo struct {
	*Data
}

func NewBloggerRepo(data *Data) BloggerRepo {
	return &bloggerRepo{Data: data}
}

// Upsert a blogger record. Safe for rank data.
func (r *bloggerRepo) Upsert(ctx context.Context, blogger *Blogger) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blogger_id"}},
		DoUpdates: clause.AssignmentColumns(allBloggerColumnsFromRank()),
	}).Create(blogger).Error
}

// allBloggerColumnsFromRank lists columns that can be updated from rank data.
func allBloggerColumnsFromRank() []string {
	return []string{
		"updated_at", "blogger_uid", "blogger_name", "blogger_avatar",
		"blogger_fans_num", "blogger_tag",
	}
}

// ListPage 实现分页、模糊查询和排序
func (r *bloggerRepo) ListPage(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*Blogger, int64, error) {
	var bloggers []*Blogger
	var total int64

	db := r.db.WithContext(ctx).Model(&Blogger{})

	// 模糊查询
	if query != "" {
		db = db.Where("blogger_name LIKE ?", "%"+query+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
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
		db = db.Order("updated_at DESC")
	}

	// 分页
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Find(&bloggers).Error; err != nil {
		return nil, 0, err
	}

	return bloggers, total, nil
}

// CopyBloggerToDTO 将 data.Blogger 模型转换为 v1.BloggerDTO
func CopyBloggerToDTO(b *Blogger) *v1.BloggerDTO {
	if b == nil {
		return nil
	}
	return &v1.BloggerDTO{
		BloggerId:      b.BloggerId,
		CreatedAt:      b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      b.UpdatedAt.Format(time.RFC3339),
		BloggerUid:     b.BloggerUid,
		BloggerName:    b.BloggerName,
		BloggerAvatar:  b.BloggerAvatar,
		BloggerFansNum: b.BloggerFansNum,
		BloggerTag:     b.BloggerTag,
	}
}

// Get finds a single blogger by their blogger_id.
func (r *bloggerRepo) Get(ctx context.Context, bloggerId int64) (*Blogger, error) {
	var blogger Blogger
	if err := r.db.WithContext(ctx).Where("blogger_id = ?", bloggerId).First(&blogger).Error; err != nil {
		return nil, err
	}
	return &blogger, nil
}
