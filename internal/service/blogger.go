package service

import (
	"aresdata/internal/biz"
	"context"

	pb "aresdata/api/v1"
)

type BloggerServiceService struct {
	pb.UnimplementedBloggerServiceServer
	uc *biz.BloggerUsecase
}

// GetBlogger retrieves a single blogger.
func (s *BloggerServiceService) GetBlogger(ctx context.Context, req *pb.BloggerQueryRequest) (*pb.BloggerQueryResponse, error) {
	blogger, err := s.uc.GetBlogger(ctx, req.BloggerId)
	if err != nil {
		return nil, err
	}
	return &pb.BloggerQueryResponse{Blogger: blogger}, nil
}

func NewBloggerServiceService(uc *biz.BloggerUsecase) *BloggerServiceService {
	return &BloggerServiceService{uc: uc}
}

func (s *BloggerServiceService) ListBloggers(ctx context.Context, req *pb.ListBloggersRequest) (*pb.ListBloggersResponse, error) {
	if req.Page == nil {
		req.Page = &pb.PageRequest{Page: 1, Size: 10}
	}
	if req.Page.Size == 0 {
		req.Page.Size = 10
	}

	bloggers, total, err := s.uc.ListBloggers(ctx, int(req.Page.Page), int(req.Page.Size), req.Query, req.SortBy, req.SortOrder)
	if err != nil {
		return nil, err
	}

	return &pb.ListBloggersResponse{
		Page:     &pb.PageResponse{Total: total},
		Bloggers: bloggers,
	}, nil
}
