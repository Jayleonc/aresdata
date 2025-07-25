package biz

import (
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewVideoRankUsecase,
	NewVideoUsecase,
	NewProductUsecase,
	NewBloggerUsecase,
	NewVideoTrendUsecase, // 新增此行
)
