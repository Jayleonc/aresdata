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

func (s *FetcherService) FetchAndStore(ctx context.Context, req *v1.FetchAndStoreRequest) (*v1.FetchAndStoreReply, error) {
	// Service层的工作就是调用Biz层，然后封装返回
	savedData, err := s.uc.FetchAndStore(ctx, req.Task)
	if err != nil {
		return nil, err // 直接将业务错误返回
	}
	return &v1.FetchAndStoreReply{SavedData: savedData}, nil
}

func (s *FetcherService) Hello(ctx context.Context, req *v1.HelloRequest) (*v1.HelloReply, error) {
	return &v1.HelloReply{Message: "Hello Jayleonc"}, nil
}
