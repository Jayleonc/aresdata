package service

import (
	"context"

	pb "aresdata/api/v1"
	"aresdata/internal/biz"
)

type VideoTrendServiceService struct {
	pb.UnimplementedVideoTrendServiceServer
	uc *biz.VideoTrendUsecase
}

func NewVideoTrendServiceService(uc *biz.VideoTrendUsecase) *VideoTrendServiceService {
	return &VideoTrendServiceService{uc: uc}
}

func (s *VideoTrendServiceService) ListVideoTrends(ctx context.Context, req *pb.ListVideoTrendsRequest) (*pb.ListVideoTrendsResponse, error) {
	if req.Page == nil {
		req.Page = &pb.PageRequest{Page: 1, Size: 10}
	}
	if req.Page.Size == 0 {
		req.Page.Size = 10
	}
	if req.Page.Size > 100 { // 限制每页最大数量
		req.Page.Size = 100
	}

	trends, total, err := s.uc.ListVideoTrends(ctx, int(req.Page.Page), int(req.Page.Size), req.AwemeId, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	return &pb.ListVideoTrendsResponse{
		Page:   &pb.PageResponse{Total: total},
		Trends: trends,
	}, nil
}
