package task

import (
	"context"
	"github.com/Jayleonc/aresdata/internal/data"
	"math/rand"
	"time"

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
	log       *log.Helper
	scheduler *HumanizedScheduler
	provider  *HeadlessTaskProvider // 只依赖 Provider
}

// NewFetchVideoDetailsHeadlessTask 重写构造函数，初始化所有字段
func NewFetchVideoDetailsHeadlessTask(
	logger log.Logger,
	provider *HeadlessTaskProvider, // 只注入 Provider
) *FetchVideoDetailsHeadlessTask {
	// 创建拟人化调度器配置
	schedulerConfig := &HumanizedSchedulerConfig{
		MinBatchSize:     3,
		MaxBatchSize:     10,
		MinShortBreakSec: 60,
		MaxShortBreakSec: 80,
		MinLongBreakSec:  120,
		MaxLongBreakSec:  300,
	}
	logHelper := log.NewHelper(log.With(logger, "module", "task.fetch_video_details_headless"))
	scheduler := NewHumanizedScheduler(logHelper, schedulerConfig)
	return &FetchVideoDetailsHeadlessTask{
		log:       logHelper,
		scheduler: scheduler,
		provider:  provider,
	}
}

// Name 返回任务名称
func (t *FetchVideoDetailsHeadlessTask) Name() string {
	return FetchVideoDetailsHeadless
}

// Run 彻底重写的主要执行方法，实现双层调度循环
func (t *FetchVideoDetailsHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [无头浏览器-视频详情] 拟人化采集任务...")

	// 1. 获取所有配置的数据源部队
	headlessSources := t.provider.FetcherManager.GetDataSourceNames()
	if len(headlessSources) == 0 {
		t.log.Error("错误：配置文件中未找到任何 headless 类型的数据源")
		return nil
	}
	t.log.Infof("发现 %d 个 headless 数据源: %v", len(headlessSources), headlessSources)

	// 2. 主循环 (无限循环，确保任务持续运行)
	for {
		// 3. 外层循环 (数据源轮换)
		for _, datasourceName := range headlessSources {
			t.log.Infof("======> 开始处理数据源: [%s] <======", datasourceName)

			// 4. 内层循环 (批次处理)
			for {
				// 4.1 获取本批次的视频数量
				batchSize := t.scheduler.GetNextBatchSize()
				t.log.Infof("数据源 [%s] 新批次启动，计划处理 %d 个视频", datasourceName, batchSize)

				// 4.2 【核心修正】通过 Usecase 层获取待采集视频，不再直接调用 Repo
				videos, err := t.provider.HeadlessUC.GetVideosForFirstCollection(ctx, batchSize)
				if err != nil {
					t.log.Errorf("从 Usecase 获取待采集视频失败: %v", err)
					time.Sleep(30 * time.Second) // 发生错误，等待一段时间再试
					continue
				}

				// 4.3 如果没有待采数据，则结束当前数据源的批次，换下一个
				if len(videos) == 0 {
					t.log.Info("已无待采集的视频，结束当前数据源的批次处理。")
					break
				}

				// 4.4 随机打乱处理顺序，其实这步不是必须的吧？
				t.scheduler.ShuffleVideos(videos)

				// 4.5 遍历视频，下发采集任务
				for _, video := range videos {
					t.log.Infof("开始处理视频 %s (来自数据源: %s)", video.AwemeId, datasourceName)

					if err := t.provider.HeadlessUC.FetchAndStoreVideoDetails(ctx, video); err != nil {
						t.log.Errorf("视频 %s 采集失败: %v", video.AwemeId, err)
					} else {
						t.log.Infof("成功处理视频 %s 的详情采集", video.AwemeId)
					}

					// 4.6 视频间短暂休眠
					t.scheduler.ShortBreak()
				}

				// 4.7 批次结束，长时间休眠
				t.log.Infof("数据源 [%s] 批次处理完成，进入长时间休眠", datasourceName)
				t.scheduler.LongBreak()
			}
			t.log.Infof("======> 数据源 [%s] 处理完成 <======", datasourceName)
		}
		t.log.Info("所有数据源已轮换一遍，30分钟后开始新一轮大循环...")
		time.Sleep(30 * time.Minute)
	}
}
