package fetcher

import (
	"context"
	"fmt"
	v1 "github.com/Jayleonc/aresdata/api/v1"
	"time"

	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/go-kratos/kratos/v2/log"
)

// HeadlessUsecase 是业务流程层（指挥官），负责编排采集和存储
type HeadlessUsecase struct {
	log            *log.Helper
	fetcherManager *FetcherManager     // <--- 新增依赖
	videoRepo      data.VideoRepo      // 新增依赖
	sourceDataRepo data.SourceDataRepo // 新增依赖
}

func NewHeadlessUsecase(
	fm *FetcherManager, // <--- 修改参数
	videoRepo data.VideoRepo,
	sourceDataRepo data.SourceDataRepo,
	logger log.Logger,
) *HeadlessUsecase {
	return &HeadlessUsecase{
		fetcherManager: fm, // <--- 修改字段赋值
		videoRepo:      videoRepo,
		sourceDataRepo: sourceDataRepo,
		log:            log.NewHelper(log.With(logger, "module", "usecase.headless")),
	}
} // 兼容老用法，repo 仍然保留

// FetchAndStoreVideoDetails 负责一个视频详情的完整采集和存储流程
// 新版本：一次性调用士兵的 CaptureVideoDetails，高效获取两份数据
func (uc *HeadlessUsecase) FetchAndStoreVideoDetails(ctx context.Context, video *data.VideoForCollection) error {
	uc.log.Infof("开始为视频 %s 执行详情采集...", video.AwemeId)
	dateCode := video.AwemePubTime.Format("20060102")

	// 【关键修改】在运行时从管理器中按名称获取一个 headless 类型的士兵
	// 注意: "feigua_headless_primary" 必须与你的 config.yaml 中的 name 完全对应
	rawFetcher, ok := uc.fetcherManager.Get("feigua_headless_primary")
	if !ok {
		return fmt.Errorf("名为 'feigua_headless_primary' 的 headless fetcher 未找到")
	}

	summaryRaw, trendRaw, err := rawFetcher.CaptureVideoDetails(ctx, video.AwemeDetailUrl)
	if err != nil {
		uc.log.Errorf("采集视频 %s 的详情数据失败: %v", video.AwemeId, err)
		return err
	}

	uc.log.Infof("视频 %s 的原始数据已成功采集，准备存入数据库...", video.AwemeId)

	// 2. 调用 "档案室" 存储 Summary 数据
	if summaryRaw != "" {
		_, saveErr := uc.sourceDataRepo.Save(ctx, &v1.SourceData{
			ProviderName: rawFetcher.GetConfig().Name,
			DataType:     data.DataTypeVideoSummaryHeadless,
			RawContent:   summaryRaw,
			EntityId:     video.AwemeId,
			FetchedAt:    time.Now().Format(time.RFC3339),
			Date:         dateCode,
		})
		if saveErr != nil {
			uc.log.Errorf("存储视频 %s 的 Summary 数据失败: %v", video.AwemeId, saveErr)
		}
	} else {
		uc.log.Warnf("视频 %s 的 Summary 数据为空，跳过存储。", video.AwemeId)
	}

	// 3. 调用 "档案室" 存储 Trend 数据
	if trendRaw != "" {
		_, saveErr := uc.sourceDataRepo.Save(ctx, &v1.SourceData{
			ProviderName: rawFetcher.GetConfig().Name,
			DataType:     data.DataTypeVideoTrendHeadless,
			RawContent:   trendRaw,
			EntityId:     video.AwemeId,
			FetchedAt:    time.Now().Format(time.RFC3339),
			Date:         dateCode,
		})
		if saveErr != nil {
			uc.log.Errorf("存储视频 %s 的 Trend 数据失败: %v", video.AwemeId, saveErr)
		}
	} else {
		uc.log.Warnf("视频 %s 的 Trend 数据为空，跳过存储。", video.AwemeId)
	}

	return nil // 整个流程成功完成（即使部分存储失败，也认为采集任务本身已成功）
}

// GetPartiallyCollectedVideos 查找部分采集失败的视频，这是 Fetcher 层的业务逻辑
func (uc *HeadlessUsecase) GetPartiallyCollectedVideos(ctx context.Context, hoursAgo int, limit int) ([]*data.VideoForCollection, error) {
	uc.log.Info("开始在 Usecase 层查找待修复视频...")
	since := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)
	dataTypes := []string{"video_trend_headless", "video_summary_headless"}

	// 步骤1：调用 sourceDataRepo，获取“成功了一半”的视频ID列表
	partiallyCollectedIDs, err := uc.sourceDataRepo.FindPartiallyCollectedEntityIDs(ctx, since, dataTypes)
	if err != nil {
		return nil, fmt.Errorf("Usecase 调用 sourceDataRepo 失败: %w", err)
	}

	if len(partiallyCollectedIDs) == 0 {
		uc.log.Info("未发现需要修复的视频。")
		return nil, nil // 没有需要修复的，直接返回
	}

	uc.log.Infof("发现 %d 个待修复的视频ID，正在查询详细信息...", len(partiallyCollectedIDs))

	// 步骤2：调用 videoRepo，用上一步获取的ID列表查询完整的视频信息
	return uc.videoRepo.FindVideosByIDs(ctx, partiallyCollectedIDs, limit)
}

// GetVideosForFirstCollection 获取用于首次详情采集的视频列表。
// 这是一个业务层方法，它编排了多个Repo来实现复杂的查询逻辑。
func (uc *HeadlessUsecase) GetVideosForFirstCollection(ctx context.Context, limit int) ([]*data.VideoForCollection, error) {
	uc.log.Info("Usecase 开始查找需要首次采集的视频...")

	// 定义需要排除的数据类型
	dataTypes := []string{"video_trend_headless", "video_summary_headless"}

	// 步骤1：调用 sourceDataRepo，拿到所有“已采集过”的视频ID黑名单
	excludeIDs, err := uc.sourceDataRepo.ListAllCollectedEntityIDs(ctx, dataTypes)
	if err != nil {
		return nil, fmt.Errorf("Usecase 获取已采集ID黑名单失败: %w", err)
	}

	// 步骤2：调用 videoRepo，获取所有不在黑名单里的视频
	videos, err := uc.videoRepo.FindVideosExcludingIDs(ctx, excludeIDs, limit)
	if err != nil {
		return nil, fmt.Errorf("Usecase 调用 videoRepo 查找待采集视频失败: %w", err)
	}

	uc.log.Infof("成功查找到 %d 个待采集视频。", len(videos))
	return videos, nil
}
