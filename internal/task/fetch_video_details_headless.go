package task

import (
	"context"
	"math/rand"
	"time"

	v1 "github.com/Jayleonc/aresdata/api/v1"
	"github.com/Jayleonc/aresdata/internal/data"
	"github.com/Jayleonc/aresdata/internal/fetcher"
	"github.com/go-kratos/kratos/v2/log"
)

// =======================================================================================
// 1. 定义可复用的"拟人化调度器" (Humanized Scheduler)
// =======================================================================================

// HumanizedSchedulerConfig 用于配置拟人化行为的所有参数
type HumanizedSchedulerConfig struct {
	// 批次大小：每个批次处理多少个视频
	MinBatchSize int
	MaxBatchSize int

	// 短暂休息（视频之间）：处理完一个视频后等待多久
	MinShortBreakSec int
	MaxShortBreakSec int

	// 长时间休息（批次之间）：处理完一个批次后等待多久
	MinLongBreakSec int
	MaxLongBreakSec int
}

// HumanizedScheduler 封装了所有拟人化的调度逻辑
type HumanizedScheduler struct {
	log    *log.Helper
	config *HumanizedSchedulerConfig
}

// NewHumanizedScheduler 创建一个新的调度器实例
func NewHumanizedScheduler(log *log.Helper, config *HumanizedSchedulerConfig) *HumanizedScheduler {
	// 初始化随机数种子，这是保证随机性的关键
	rand.Seed(time.Now().UnixNano())
	return &HumanizedScheduler{log: log, config: config}
}

// ShuffleVideos 随机打乱视频列表的处理顺序
func (s *HumanizedScheduler) ShuffleVideos(videos []*data.VideoForCollection) {
	s.log.Info("正在打乱视频处理顺序...")
	rand.Shuffle(len(videos), func(i, j int) {
		videos[i], videos[j] = videos[j], videos[i]
	})
}

// ShortBreak 执行一次短暂的、随机的休眠
func (s *HumanizedScheduler) ShortBreak() {
	waitTime := rand.Intn(s.config.MaxShortBreakSec-s.config.MinShortBreakSec+1) + s.config.MinShortBreakSec
	s.log.Infof("任务节流：进入短暂休眠，持续 %d 秒...", waitTime)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

// LongBreak 执行一次长时间的、随机的休眠
func (s *HumanizedScheduler) LongBreak() {
	waitTime := rand.Intn(s.config.MaxLongBreakSec-s.config.MinLongBreakSec+1) + s.config.MinLongBreakSec
	s.log.Infof("批次任务完成：进入长时间休眠，持续 %d 秒...", waitTime)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

// GetNextBatchSize 获取下一个随机的批次大小
func (s *HumanizedScheduler) GetNextBatchSize() int {
	return rand.Intn(s.config.MaxBatchSize-s.config.MinBatchSize+1) + s.config.MinBatchSize
}

// =======================================================================================
// 2. 重新定义的任务结构体
// =======================================================================================

// FetchVideoDetailsHeadlessTask 重新定义的任务结构体，只包含必要字段
type FetchVideoDetailsHeadlessTask struct {
	log            *log.Helper
	videoRepo      data.VideoRepo
	fetcherManager *fetcher.FetcherManager
	scheduler      *HumanizedScheduler      // 引入我们之前定义的拟人化调度器
	headlessUC     *fetcher.HeadlessUsecase // 新增，负责采集和存储
}

// NewFetchVideoDetailsHeadlessTask 重写构造函数，初始化所有字段
func NewFetchVideoDetailsHeadlessTask(
	logger log.Logger,
	videoRepo data.VideoRepo,
	fetcherManager *fetcher.FetcherManager,
	headlessUC *fetcher.HeadlessUsecase, // 新增参数
) *FetchVideoDetailsHeadlessTask {
	// 创建拟人化调度器配置
	schedulerConfig := &HumanizedSchedulerConfig{
		MinBatchSize:     3,   // 每批次最少处理3个视频
		MaxBatchSize:     10,  // 每批次最多处理10个视频
		MinShortBreakSec: 60,  // 视频间最少休息2秒
		MaxShortBreakSec: 80,  // 视频间最多休息8秒
		MinLongBreakSec:  120, // 批次间最少休息30秒
		MaxLongBreakSec:  300, // 批次间最多休息120秒
	}

	logHelper := log.NewHelper(log.With(logger, "module", "task.fetch_video_details_headless"))
	scheduler := NewHumanizedScheduler(logHelper, schedulerConfig)

	return &FetchVideoDetailsHeadlessTask{
		log:            logHelper,
		videoRepo:      videoRepo,
		fetcherManager: fetcherManager,
		scheduler:      scheduler,
		headlessUC:     headlessUC, // 注入 usecase
	}
}

// Name 返回任务名称
func (t *FetchVideoDetailsHeadlessTask) Name() string {
	return FetchVideoDetailsHeadless
}

// Run 彻底重写的主要执行方法，实现双层调度循环
func (t *FetchVideoDetailsHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [无头浏览器-视频详情] 拟人化采集任务...")

	// 1. 获取所有数据源
	headlessSources := t.fetcherManager.GetDataSourceNames()
	if len(headlessSources) == 0 {
		t.log.Error("未找到任何数据源")
		return nil
	}
	t.log.Infof("发现 %d 个数据源: %v", len(headlessSources), headlessSources)

	// 2. 主循环 (无限循环)
	for {
		// 3. 外层循环 (数据源轮换)
		for _, datasourceName := range headlessSources {
			t.log.Infof("======> 开始处理数据源: [%s] <======", datasourceName)

			// 4. 内层循环 (批次处理)
			for {
				// 4.1 使用 scheduler.GetNextBatchSize() 获取本批次的视频数量
				batchSize := t.scheduler.GetNextBatchSize()
				t.log.Infof("数据源 [%s] 开始处理批次，大小: %d", datasourceName, batchSize)

				// 4.2 调用 videoRepo 从数据库中获取相应数量的待采集视频
				videos, err := t.videoRepo.FindVideosForDetailsCollection(ctx, batchSize)
				if err != nil {
					t.log.Errorf("从数据库获取待采集视频失败: %v", err)
					time.Sleep(30 * time.Second) // 数据库查询失败，等待一段时间再试
					continue
				}

				// 4.3 如果获取不到视频，则表明已无待采数据，可以 break 内层循环
				if len(videos) == 0 {
					t.log.Info("数据库中已没有待采集的视频，结束当前数据源的批次处理")
					break // 跳出内层循环，切换到下一个数据源
				}

				// 4.4 使用 scheduler.ShuffleVideos() 打乱视频处理顺序
				t.scheduler.ShuffleVideos(videos)

				// 4.5 遍历本批次的视频列表，对每一个视频，调用 headlessUC.FetchAndStoreVideoDetails()
				for _, video := range videos {
					t.log.Infof("开始处理视频 %s (来自数据源: %s)", video.AwemeId, datasourceName)

					// 将 VideoForCollection 转换为 VideoDTO
					videoDTO := &v1.VideoDTO{
						AwemeId:        video.AwemeId,
						AwemePubTime:   video.AwemePubTime.Format("2006-01-02 15:04:05"),
						AwemeDetailUrl: video.AwemeDetailUrl,
					}

					// 调用 HeadlessUsecase 进行采集
					if err := t.headlessUC.FetchAndStoreVideoDetails(ctx, videoDTO); err != nil {
						t.log.Errorf("视频 %s 采集失败: %v", video.AwemeId, err)
					} else {
						t.log.Infof("成功处理视频 %s 的详情采集", video.AwemeId)
					}

					// 4.6 每处理完一个视频，调用 scheduler.ShortBreak() 进行短暂休眠
					t.scheduler.ShortBreak()
				}

				// 4.7 批次结束: 内层循环结束后，意味着一个批次已完成
				t.log.Infof("数据源 [%s] 批次处理完成，准备进入长时间休眠", datasourceName)
				t.scheduler.LongBreak()
			}

			t.log.Infof("======> 数据源 [%s] 处理完成 <======", datasourceName)
		}

		t.log.Info("所有数据源轮换完成，开始新一轮循环...")
	}
}
