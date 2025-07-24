package etl

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"time"
)

// ProviderSet 保持不变，wire 会自动注入 VideoRepo
var ProviderSet = wire.NewSet(
	NewETLUsecase,
	NewVideoRankProcessor,
	NewVideoDetailProcessor,
)

// Processor defines a generic ETL processor.
type Processor interface {
	Process(ctx context.Context, rawData *v1.SourceData) error
}

// ETLUsecase orchestrates ETL processors.
type ETLUsecase struct {
	log            *log.Helper
	sourceDataRepo data.SourceDataRepo
	processors     map[string]Processor
}

// NewETLUsecase 是 ETLUsecase 的构造函数，负责依赖注入和初始化
// wire 会自动找到并注入 NewVideoRankProcessor 和 NewVideoDetailProcessor 的实例
func NewETLUsecase(
	logger log.Logger,
	sdRepo data.SourceDataRepo,
	vrp *VideoRankProcessor, // video rank processor
	vdp *VideoDetailProcessor, // video detail processor
) *ETLUsecase {
	// key 是 source_data 表的 data_type
	processors := map[string]Processor{
		// 旧的 rank 处理器保持不变
		"video_rank_day": vrp,

		// 新的：将 summary 和 trend 两种数据类型都指向同一个 Detail 处理器
		"video_summary_headless": vdp,
		"video_trend_headless":   vdp,
	}

	return &ETLUsecase{
		log:            log.NewHelper(log.With(logger, "module", "usecase/etl")),
		sourceDataRepo: sdRepo,
		processors:     processors,
	}
}

// RunWithType 根据指定的数据类型，查找并处理所有未处理的数据
func (u *ETLUsecase) RunWithType(ctx context.Context, dataType string) error {
	u.log.WithContext(ctx).Infof("开始处理类型为 [%s] 的源数据...", dataType)

	list, err := u.sourceDataRepo.FindUnprocessed(ctx, dataType)
	if err != nil {
		u.log.WithContext(ctx).Errorf("查找类型为 [%s] 的未处理数据失败: %v", dataType, err)
		return err
	}

	if len(list) == 0 {
		u.log.WithContext(ctx).Infof("类型为 [%s] 的数据无需处理。", dataType)
		return nil
	}

	u.log.WithContext(ctx).Infof("发现 %d 条类型为 [%s] 的数据待处理。", len(list), dataType)

	for _, raw := range list {
		// 根据数据类型从 map 中查找对应的处理器
		processor, ok := u.processors[raw.DataType]
		if !ok {
			u.log.WithContext(ctx).Warnf("未找到数据类型 [%s] (SourceID: %d) 对应的处理器，已跳过。", raw.DataType, raw.Id)
			continue
		}

		// 执行处理逻辑
		if err := processor.Process(ctx, raw); err != nil {
			// 即使单个任务失败，也只记录错误并继续处理下一个，不中断整个批次
			u.log.WithContext(ctx).Errorf("处理数据 (SourceID: %d, DataType: %s) 失败: %v", raw.Id, raw.DataType, err)
			continue
		}
		time.Sleep(2 * time.Microsecond)
	}

	u.log.WithContext(ctx).Infof("类型为 [%s] 的数据已全部处理完毕。", dataType)
	return nil
}

// Run processes all unprocessed source data.
func (u *ETLUsecase) Run(ctx context.Context, dataType string) error {
	return u.RunWithType(ctx, dataType)
}
