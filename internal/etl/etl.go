package etl

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewETL,
	NewVideoRankProcessor,
)

// Processor defines a generic ETL processor.
type Processor interface {
	Process(ctx context.Context, rawData *v1.SourceData) error
}

// ETL orchestrates ETL processors.
type ETL struct {
	sourceDataRepo data.SourceDataRepo
	processors     map[string]Processor
}

func NewETL(sdRepo data.SourceDataRepo, vr *VideoRankProcessor) *ETL {
	processors := map[string]Processor{
		"video_rank_day": vr,
		// 扩展
	}

	return &ETL{
		sourceDataRepo: sdRepo,
		processors:     processors,
	}
}

// Run processes all unprocessed source data.
func (u *ETL) Run(ctx context.Context, dataType string) error {
	return u.RunWithType(ctx, dataType)
}

// RunWithType processes unprocessed source data, filtered by dataType if provided.
func (u *ETL) RunWithType(ctx context.Context, dataType string) error {
	list, err := u.sourceDataRepo.FindUnprocessed(ctx)
	if err != nil {
		return err
	}
	for _, raw := range list {
		if dataType != "" && raw.DataType != dataType {
			continue // 跳过非指定类型
		}
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
