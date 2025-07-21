package service

import (
	"aresdata/internal/biz"
	"context"

	pb "aresdata/api/v1"
)

type ProductServiceService struct {
	pb.UnimplementedProductServiceServer
	uc *biz.ProductUsecase
}

// GetProduct retrieves a single product.
func (s *ProductServiceService) GetProduct(ctx context.Context, req *pb.ProductQueryRequest) (*pb.ProductQueryResponse, error) {
	product, err := s.uc.GetProduct(ctx, req.GoodsId)
	if err != nil {
		return nil, err
	}
	return &pb.ProductQueryResponse{Product: product}, nil
}

func NewProductServiceService(uc *biz.ProductUsecase) *ProductServiceService {
	return &ProductServiceService{uc: uc}
}

func (s *ProductServiceService) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	if req.Page == nil {
		req.Page = &pb.PageRequest{Page: 1, Size: 10}
	}
	if req.Page.Size == 0 {
		req.Page.Size = 10
	}

	products, total, err := s.uc.ListProducts(ctx, int(req.Page.Page), int(req.Page.Size), req.Query, req.SortBy, req.SortOrder)
	if err != nil {
		return nil, err
	}

	return &pb.ListProductsResponse{
		Page:     &pb.PageResponse{Total: total},
		Products: products,
	}, nil
}
