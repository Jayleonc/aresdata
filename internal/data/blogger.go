package data

import (
	"context"
	"gorm.io/gorm/clause"
	"time"
)

// Blogger 博主维度表
type Blogger struct {
	BloggerId int64     `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	BloggerUid     string `gorm:"size:255;index"`
	BloggerName    string `gorm:"size:255"`
	BloggerAvatar  string `gorm:"size:1024"`
	BloggerFansNum int64
	BloggerTag     string `gorm:"size:255"`
}

func (Blogger) TableName() string {
	return "bloggers"
}

type BloggerRepo interface {
	Upsert(ctx context.Context, blogger *Blogger) error
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
