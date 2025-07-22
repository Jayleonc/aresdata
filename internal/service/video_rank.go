package service

import (
	"context"

	pb "aresdata/api/v1"
	"aresdata/internal/biz"
)

// VideoRankService 提供视频榜单查询的 gRPC/HTTP 服务
// 依赖 biz.VideoRankUsecase

type VideoRankService struct {
	pb.UnimplementedVideoRankServer
	uc *biz.VideoRankUsecase
}

// NewVideoRankService 构造 VideoRankService
func NewVideoRankService(uc *biz.VideoRankUsecase) *VideoRankService {
	return &VideoRankService{uc: uc}
}

// GetVideoRank 查询单个视频榜单
func (s *VideoRankService) GetVideoRank(ctx context.Context, req *pb.VideoRankQueryRequest) (*pb.VideoRankQueryResponse, error) {
	rank, err := s.uc.GetVideoRank(ctx, req.AwemeId, req.RankType, req.RankDate)
	if err != nil {
		return nil, err
	}
	return &pb.VideoRankQueryResponse{Rank: rank}, nil
}

// ListVideoRank 分页查询视频榜单
func (s *VideoRankService) ListVideoRank(ctx context.Context, req *pb.ListVideoRankRequest) (*pb.ListVideoRankResponse, error) {
	if req.Page == nil {
		req.Page = &pb.PageRequest{Page: 1, Size: 10} // 默认值
	}
	if req.Page.Size == 0 {
		req.Page.Size = 10
	}

	sortBy := req.GetSortBy()
	sortOrder := req.GetSortOrder()

	ranks, total, err := s.uc.ListVideoRank(ctx, int(req.Page.Page), int(req.Page.Size), req.RankType, req.RankDate, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}
	return &pb.ListVideoRankResponse{
		Page:  &pb.PageResponse{Total: total},
		Ranks: ranks,
	}, nil
}
