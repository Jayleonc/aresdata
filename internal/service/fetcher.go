package service

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

type FetcherService struct {
	v1.UnimplementedFetcherServer

	uc  *biz.FetcherUsecase
	log *log.Helper
}

func NewFetcherService(uc *biz.FetcherUsecase, logger log.Logger) *FetcherService {
	return &FetcherService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/fetcher")),
	}
}

func (s *FetcherService) Hello(ctx context.Context, req *v1.HelloRequest) (*v1.HelloReply, error) {
	return &v1.HelloReply{Message: "Hello AresData"}, nil
}
