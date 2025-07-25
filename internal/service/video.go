package service

import (
	"context"
	"github.com/Jayleonc/aresdata/internal/biz"

	pb "github.com/Jayleonc/aresdata/api/v1"
)

type VideoServiceService struct {
	uc *biz.VideoUsecase
	pb.UnimplementedVideoServiceServer
}

func NewVideoServiceService(uc *biz.VideoUsecase) *VideoServiceService {
	return &VideoServiceService{uc: uc}
}

func (s *VideoServiceService) ListVideos(ctx context.Context, req *pb.ListVideosRequest) (*pb.ListVideosResponse, error) {
	if req.Page == nil {
		req.Page = &pb.PageRequest{Page: 1, Size: 10}
	}
	if req.Page.Size == 0 {
		req.Page.Size = 10
	}

	videos, total, err := s.uc.ListVideos(ctx, int(req.Page.Page), int(req.Page.Size), req.Query, req.SortBy, req.SortOrder)
	if err != nil {
		return nil, err
	}

	return &pb.ListVideosResponse{
		Page:   &pb.PageResponse{Total: total},
		Videos: videos,
	}, nil
}

func (s *VideoServiceService) GetVideo(ctx context.Context, req *pb.VideoQueryRequest) (*pb.VideoQueryResponse, error) {
	video, err := s.uc.GetVideo(ctx, req.AwemeId)
	if err != nil {
		return nil, err
	}
	return &pb.VideoQueryResponse{Video: video}, nil
}
