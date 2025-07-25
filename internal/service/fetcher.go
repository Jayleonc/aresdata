package service

import (
	"context"

	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/fetcher"
	"github.com/go-kratos/kratos/v2/log"
)

type FetcherService struct {
	v1.UnimplementedFetcherServer

	uc  *fetcher.HttpUsecase
	log *log.Helper
}

func NewFetcherService(uc *fetcher.HttpUsecase, logger log.Logger) *FetcherService {
	return &FetcherService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/fetcher")),
	}
}

func (s *FetcherService) Hello(ctx context.Context, req *v1.HelloRequest) (*v1.HelloReply, error) {
	return &v1.HelloReply{Message: "Hello AresData"}, nil
}
