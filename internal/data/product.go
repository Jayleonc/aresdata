package data

import (
	"context"
	"gorm.io/gorm/clause"
	"time"

	"aresdata/api/v1"
)

// Product 商品维度表
type Product struct {
	GoodsId   string    `gorm:"primaryKey;size:1024"`
	CreatedAt time.Time `gorm:"autoCreateTime;type:timestamp"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;type:timestamp"`

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
	ListPage(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*Product, int64, error)
	Get(ctx context.Context, goodsId string) (*Product, error)
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

// ListPage 实现分页、模糊查询和排序
func (r *productRepo) ListPage(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*Product, int64, error) {
	var products []*Product
	var total int64

	db := r.db.WithContext(ctx).Model(&Product{})

	// 模糊查询
	if query != "" {
		db = db.Where("goods_title LIKE ?", "%"+query+"%")
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
		db = db.Order("updated_at DESC")
	}

	// 分页
	offset := (page - 1) * size
	if err := db.Offset(offset).Limit(size).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// CopyProductToDTO 将 data.Product 模型转换为 v1.ProductDTO
func CopyProductToDTO(p *Product) *v1.ProductDTO {
	if p == nil {
		return nil
	}
	return &v1.ProductDTO{
		GoodsId:         p.GoodsId,
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
		GoodsTitle:      p.GoodsTitle,
		GoodsCoverUrl:   p.GoodsCoverUrl,
		GoodsPriceRange: p.GoodsPriceRange,
		GoodsPrice:      p.GoodsPrice,
		CosRatio:        p.CosRatio,
		CommissionPrice: p.CommissionPrice,
		ShopName:        p.ShopName,
		BrandName:       p.BrandName,
		CategoryNames:   p.CategoryNames,
	}
}

// Get finds a single product by its goods_id.
func (r *productRepo) Get(ctx context.Context, goodsId string) (*Product, error) {
	var product Product
	if err := r.db.WithContext(ctx).Where("goods_id = ?", goodsId).First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}
