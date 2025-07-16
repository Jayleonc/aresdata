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

// BatchGetVideoRank 批量查询视频榜
// BatchGetVideoRank 批量查询视频榜单
func (s *VideoRankService) BatchGetVideoRank(ctx context.Context, req *pb.BatchVideoRankQueryRequest) (*pb.BatchVideoRankQueryResponse, error) {
	ranks, err := s.uc.BatchGetVideoRank(ctx, req.AwemeIds, req.RankType, req.RankDate)
	if err != nil {
		return nil, err
	}
	return &pb.BatchVideoRankQueryResponse{Ranks: ranks}, nil
}
