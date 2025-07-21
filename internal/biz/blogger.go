package biz

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
)

// BloggerUsecase 封装视频博主维度相关的业务逻辑
type BloggerUsecase struct {
	repo data.BloggerRepo
}

// GetBlogger retrieves a single blogger by their ID.
func (uc *BloggerUsecase) GetBlogger(ctx context.Context, bloggerId int64) (*v1.BloggerDTO, error) {
	blogger, err := uc.repo.Get(ctx, bloggerId)
	if err != nil {
		return nil, err
	}
	return data.CopyBloggerToDTO(blogger), nil
}

// NewBloggerUsecase 构造 BloggerUsecase
func NewBloggerUsecase(repo data.BloggerRepo) *BloggerUsecase {
	return &BloggerUsecase{repo: repo}
}

// ListBloggers 分页查询视频博主
func (uc *BloggerUsecase) ListBloggers(ctx context.Context, page, size int, query, sortBy string, sortOrder v1.SortOrder) ([]*v1.BloggerDTO, int64, error) {
	bloggers, total, err := uc.repo.ListPage(ctx, page, size, query, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*v1.BloggerDTO, len(bloggers))
	for i, b := range bloggers {
		dtos[i] = data.CopyBloggerToDTO(b)
	}
	return dtos, total, nil
}
