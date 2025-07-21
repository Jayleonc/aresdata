package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
)

// ProductUsecase 封装商品维度相关的业务逻辑
type ProductUsecase struct {
	repo data.ProductRepo
}

// GetProduct retrieves a single product by its ID.
func (uc *ProductUsecase) GetProduct(ctx context.Context, goodsId string) (*v1.ProductDTO, error) {
	product, err := uc.repo.Get(ctx, goodsId)
	if err != nil {
		return nil, err
	}
	return data.CopyProductToDTO(product), nil
}

// NewProductUsecase 构造 ProductUsecase
func NewProductUsecase(repo data.ProductRepo) *ProductUsecase {
	return &ProductUsecase{repo: repo}
}

// ListProducts 分页查询商品
func (uc *ProductUsecase) ListProducts(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*v1.ProductDTO, int64, error) {
	products, total, err := uc.repo.ListPage(ctx, page, size, query, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*v1.ProductDTO, len(products))
	for i, p := range products {
		dtos[i] = data.CopyProductToDTO(p)
	}
	return dtos, total, nil
}
