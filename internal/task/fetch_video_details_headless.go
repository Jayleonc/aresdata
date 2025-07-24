package task

import (
	"aresdata/internal/biz"
	"aresdata/internal/conf"
	"aresdata/internal/data" // <-- Import the data package
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/go-kratos/kratos/v2/log"
)

// =======================================================================================
// 1. 定义可复用的“拟人化调度器” (Humanized Scheduler)
// =======================================================================================

// HumanizedSchedulerConfig 用于配置拟人化行为的所有参数
type HumanizedSchedulerConfig struct {
	// 工作周期：一个任务实例总共执行多少个批次
	MinBatchesPerCycle int
	MaxBatchesPerCycle int

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

// GetTotalBatchesForCycle 获取本次工作周期总共要执行多少个批次
func (s *HumanizedScheduler) GetTotalBatchesForCycle() int {
	return rand.Intn(s.config.MaxBatchesPerCycle-s.config.MinBatchesPerCycle+1) + s.config.MinBatchesPerCycle
}

// =======================================================================================
// 2. 将调度器应用到我们的主任务中
// =======================================================================================

// FetchVideoDetailsHeadlessTask 结构体保持不变
type FetchVideoDetailsHeadlessTask struct {
	log        *log.Helper
	videoUC    *biz.VideoUsecase
	headlessUC *biz.HeadlessUsecase
	cfg        *conf.Feigua
}

// NewFetchVideoDetailsHeadlessTask 构造函数保持不变
func NewFetchVideoDetailsHeadlessTask(
	logger log.Logger,
	videoUC *biz.VideoUsecase,
	headlessUC *biz.HeadlessUsecase,
	cfg *conf.Data,
) *FetchVideoDetailsHeadlessTask {
	return &FetchVideoDetailsHeadlessTask{
		log:        log.NewHelper(log.With(logger, "module", "task.fetch_video_details_headless")),
		videoUC:    videoUC,
		headlessUC: headlessUC,
		cfg:        cfg.GetFeigua(),
	}
}

// Name 方法保持不变
func (t *FetchVideoDetailsHeadlessTask) Name() string {
	return FetchVideoDetailsHeadless
}

// Run 方法被完全重构，以使用新的调度器
func (t *FetchVideoDetailsHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [无头浏览器-视频详情] 拟人化采集任务...")

	// --- 1. 浏览器启动和预热 (逻辑不变) ---
	var opts []chromedp.ExecAllocatorOption
	baseOpts := []chromedp.ExecAllocatorOption{
		chromedp.UserAgent(t.cfg.UserAgent),
		chromedp.WindowSize(1920, 1080),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	}
	if t.cfg.Headless {
		t.log.Info("当前为 Headless 模式，正在应用反检测参数...")
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", "new"),
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-infobars", true),
			chromedp.Flag("disable-popup-blocking", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-setuid-sandbox", true),
		)
		opts = append(opts, baseOpts...)
	} else {
		t.log.Info("当前为非 Headless 模式，将启动浏览器界面...")
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
		)
		opts = append(opts, baseOpts...)
	}
	if t.cfg.Proxy != "" {
		opts = append(opts, chromedp.ProxyServer(t.cfg.Proxy))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	browserCtx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(t.newChromedpLogger()),
	)
	browserCtx, cancel = context.WithTimeout(browserCtx, 60*time.Minute)
	defer cancel()
	if err := chromedp.Run(browserCtx, t.loadCookiesAction()); err != nil {
		t.log.Errorf("浏览器启动时加载Cookie失败: %v", err)
		return err
	}
	t.log.Info("浏览器实例已启动，并成功加载初始Cookie。")
	t.log.Info("正在执行浏览器预热...")
	warmupCtx, cancelWarmup := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancelWarmup()
	if err := chromedp.Run(warmupCtx,
		chromedp.Navigate(t.cfg.BaseUrl),
		chromedp.Sleep(5*time.Second),
	); err != nil {
		t.log.Warnf("浏览器预热失败（不影响主流程）: %v", err)
	} else {
		t.log.Info("浏览器预热成功，准备开始执行主任务。")
	}

	// --- 2. 初始化我们的“拟人化调度器” ---
	schedulerConfig := &HumanizedSchedulerConfig{
		MinBatchesPerCycle: 3,   // 每次任务最少跑3个批次
		MaxBatchesPerCycle: 5,   // 最多跑5个批次
		MinBatchSize:       8,   // 每批次最少拿8个视频
		MaxBatchSize:       22,  // 最多拿22个
		MinShortBreakSec:   35,  // 单个视频处理完，最少等35秒
		MaxShortBreakSec:   180, // 最多等3分钟
		MinLongBreakSec:    300, // 单批次处理完，最少等5分钟
		MaxLongBreakSec:    900, // 最多等15分钟
	}
	scheduler := NewHumanizedScheduler(t.log, schedulerConfig)
	totalBatches := scheduler.GetTotalBatchesForCycle()
	t.log.Infof("本次工作周期计划执行 %d 个批次。", totalBatches)

	// --- 3. 【核心改造】全新的、基于调度器的工作循环 ---
	for batchNum := 1; batchNum <= totalBatches; batchNum++ {
		t.log.Infof("开始处理第 %d / %d 批次...", batchNum, totalBatches)

		// 3.1 获取一个随机大小的批次
		batchSize := scheduler.GetNextBatchSize()
		videos, err := t.videoUC.GetVideosForFirstCollection(ctx, batchSize)
		if err != nil {
			t.log.Errorf("获取第 %d 批次视频列表失败: %v", batchNum, err)
			continue // 本批次失败，休息一下，然后尝试下一批
		}
		if len(videos) == 0 {
			t.log.Info("没有需要进行首次采集的视频了，任务提前结束。")
			break // 数据库里没新视频了，直接结束整个任务
		}
		t.log.Infof("获取到 %d 个需要进行首次采集的视频", len(videos))

		// 3.2 打乱处理顺序
		scheduler.ShuffleVideos(videos)

		// 3.3 遍历处理本批次的视频
		for _, v := range videos {
			if v.AwemeDetailUrl == "" {
				t.log.Warnf("视频 %s 的 AwemeDetailUrl 为空，跳过采集", v.AwemeId)
				continue
			}
			dateCode := v.AwemePubTime.Format("20060102")

			// 使用我们已有的重试逻辑来增加单次任务的成功率
			var taskErr error
			maxRetries := 2
			for attempt := 0; attempt <= maxRetries; attempt++ {
				taskErr = t.headlessUC.FetchAndStoreVideoDetails(browserCtx, v.AwemeDetailUrl, v.AwemeId, dateCode)
				if taskErr == nil {
					break
				}
				t.log.Errorf("视频 %s 第 %d 次尝试失败: %v", v.AwemeId, attempt, taskErr)
				// 可以在重试前加入一个短暂的固定等待
				if attempt < maxRetries {
					time.Sleep(5 * time.Second)
				}
			}
			if taskErr != nil {
				t.log.Errorf("视频 %s 在所有重试后最终失败: %v", v.AwemeId, taskErr)
			} else {
				t.log.Infof("成功处理视频 %s 的详情采集", v.AwemeId)
			}

			// 3.4 短暂休眠
			scheduler.ShortBreak()
		}

		// 3.5 如果不是最后一个批次，则进行长时间休眠
		if batchNum < totalBatches {
			scheduler.LongBreak()
		}
	}

	t.log.Info("[无头浏览器-视频详情] 本次工作周期的所有任务已处理完毕，浏览器将关闭。")
	return nil
}

// loadCookiesAction 和 newChromedpLogger 两个辅助方法保持不变
func (t *FetchVideoDetailsHeadlessTask) loadCookiesAction() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var cookieBytes []byte
		var err error
		if t.cfg.CookieContent != "" {
			cookieBytes = []byte(t.cfg.CookieContent)
		} else if t.cfg.CookiePath != "" {
			cookieBytes, err = ioutil.ReadFile(t.cfg.CookiePath)
			if err != nil {
				return fmt.Errorf("读取cookie文件失败: %w", err)
			}
		} else {
			t.log.Warn("未配置Cookie，将以无身份状态访问")
			return nil // 没有cookie也继续
		}

		// 替换 "sameSite" 字段以兼容chromedp
		processedCookiesStr := strings.ReplaceAll(string(cookieBytes), `"sameSite": "unspecified"`, `"sameSite": "Lax"`)
		cookieBytes = []byte(processedCookiesStr)

		cookies := []*network.CookieParam{}
		if err := json.Unmarshal(cookieBytes, &cookies); err != nil {
			return fmt.Errorf("解析cookie JSON失败: %w", err)
		}
		t.log.Infof("成功加载 %d 个Cookies", len(cookies))
		return network.SetCookies(cookies).Do(ctx)
	})
}

func (t *FetchVideoDetailsHeadlessTask) newChromedpLogger() func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if strings.Contains(msg, "initiatorIPAddressSpace") {
			return
		}
		t.log.Info(msg)
	}
}
