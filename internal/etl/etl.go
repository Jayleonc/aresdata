package etl

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
	"github.com/google/wire"
)

// ProviderSet 保持不变，wire 会自动注入 VideoRepo
var ProviderSet = wire.NewSet(
	NewETLUsecase,
	NewVideoRankProcessor,
	NewVideoTrendProcessor,
	NewVideoSummaryProcessor,
)

// Processor defines a generic ETL processor.
type Processor interface {
	Process(ctx context.Context, rawData *v1.SourceData) error
}

// ETLUsecase orchestrates ETL processors.
type ETLUsecase struct {
	sourceDataRepo data.SourceDataRepo
	processors     map[string]Processor
}

// 修改构造函数签名，注入新的 processor
func NewETLUsecase(sdRepo data.SourceDataRepo, vr *VideoRankProcessor, vt *VideoTrendProcessor, vs *VideoSummaryProcessor) *ETLUsecase {
	processors := map[string]Processor{
		"video_rank_day":    vr,
		"video_trend_daily": vt,
		"video_summary":     vs,
		// 扩展...
	}

	return &ETLUsecase{
		sourceDataRepo: sdRepo,
		processors:     processors,
	}
}

// Run processes all unprocessed source data.
func (u *ETLUsecase) Run(ctx context.Context, dataType string) error {
	return u.RunWithType(ctx, dataType)
}

// RunWithType processes unprocessed source data, filtered by dataType if provided.
func (u *ETLUsecase) RunWithType(ctx context.Context, dataType string) error {
	list, err := u.sourceDataRepo.FindUnprocessed(ctx, dataType)
	if err != nil {
		return err
	}
	for _, raw := range list {
		//if dataType != "" && raw.DataType != dataType {
		//	continue // 跳过非指定类型
		//}
		processor, ok := u.processors[raw.DataType]
		if !ok {
			continue // 未知类型跳过
		}
		if err := processor.Process(ctx, raw); err != nil {
			// 可记录日志或继续处理下一个
			continue
		}
	}
	return nil
}
