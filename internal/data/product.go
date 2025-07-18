package data

import (
	"context"
	"gorm.io/gorm/clause"
	"time"
)

// Product 商品维度表
type Product struct {
	GoodsId   string    `gorm:"primaryKey;size:1024"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	GoodsTitle      string  `gorm:"type:text"`
	GoodsCoverUrl   string  `gorm:"size:1024"`
	GoodsPriceRange string  `gorm:"size:255"`
	GoodsPrice      float64 `gorm:"type:numeric"`
	CosRatio        string  `gorm:"size:255"`
	CommissionPrice string  `gorm:"size:255"`
	ShopName        string  `gorm:"size:255"`
	BrandName       string  `gorm:"size:255"`
	CategoryNames   string  `gorm:"size:1024"`
}

func (Product) TableName() string {
	return "products"
}

type ProductRepo interface {
	Upsert(ctx context.Context, product *Product) error
}

type productRepo struct {
	*Data
}

func NewProductRepo(data *Data) ProductRepo {
	return &productRepo{Data: data}
}

// Upsert a product record. If the record with the same GoodsId exists, it will be updated.
// Otherwise, a new record will be created. This is a safe upsert for rank data.
func (r *productRepo) Upsert(ctx context.Context, product *Product) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "goods_id"}},
		DoUpdates: clause.AssignmentColumns(allProductColumnsFromRank()),
	}).Create(product).Error
}

// allProductColumnsFromRank lists columns that can be updated from rank data.
func allProductColumnsFromRank() []string {
	return []string{
		"updated_at", "goods_title", "goods_cover_url", "goods_price_range",
		"goods_price", "cos_ratio", "commission_price", "shop_name",
		"brand_name", "category_names",
	}
}
