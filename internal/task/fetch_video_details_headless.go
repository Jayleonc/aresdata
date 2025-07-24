// internal/task/fetch_video_details_headless.go

package task

import (
	"aresdata/internal/biz"
	"aresdata/internal/conf"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/go-kratos/kratos/v2/log"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"
)

// FetchVideoDetailsHeadlessTask 负责调度采集视频的完整详情数据
type FetchVideoDetailsHeadlessTask struct {
	log        *log.Helper
	videoUC    *biz.VideoUsecase
	headlessUC *biz.HeadlessUsecase
	cfg        *conf.Feigua
}

// NewFetchVideoDetailsHeadlessTask 创建一个新的任务实例
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

// Name 返回任务的名称
func (t *FetchVideoDetailsHeadlessTask) Name() string {
	return FetchVideoDetailsHeadless
}

// Run 执行任务的核心逻辑，现在负责管理浏览器生命周期
func (t *FetchVideoDetailsHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [无头浏览器-视频详情] 拉取任务...")

	// --- 1. 创建唯一的浏览器实例 (逻辑不变) ---
	// --- 1. 在任务开始时，创建唯一的浏览器实例 ---
	// 【关键修改】根据是否为 headless 模式，应用不同的启动参数
	var opts []chromedp.ExecAllocatorOption

	// 基础参数，对两种模式都适用
	baseOpts := []chromedp.ExecAllocatorOption{
		chromedp.UserAgent(t.cfg.UserAgent),
		chromedp.WindowSize(1920, 1080), // 设置一个常规的窗口大小
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	}

	if t.cfg.Headless {
		// 【Headless 模式下的伪装参数】
		t.log.Info("当前为 Headless 模式，正在应用反检测参数...")
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			// 使用新的 headless 模式，更难被检测
			chromedp.Flag("headless", "new"),
			// 禁用自动化控制的特征，这是最关键的参数之一
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
			// 禁用一些可能暴露 headless 状态的组件
			chromedp.Flag("disable-extensions", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("disable-infobars", true),
			chromedp.Flag("disable-popup-blocking", true),
			chromedp.Flag("disable-dev-shm-usage", true), // 在Docker中运行时尤其重要
			chromedp.Flag("no-sandbox", true),            // 在Linux服务器上运行时通常需要
			chromedp.Flag("disable-setuid-sandbox", true),
		)
		opts = append(opts, baseOpts...)
	} else {
		// 【非 Headless 模式（调试模式）下的参数】
		t.log.Info("当前为非 Headless 模式，将启动浏览器界面...")
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
		)
		opts = append(opts, baseOpts...)
	}

	// 代理配置保持不变
	if t.cfg.Proxy != "" {
		opts = append(opts, chromedp.ProxyServer(t.cfg.Proxy))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// 创建浏览器主上下文
	browserCtx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(t.newChromedpLogger()),
	)
	browserCtx, cancel = context.WithTimeout(browserCtx, 60*time.Minute) // 整个任务的总超时
	defer cancel()

	// --- 2. 加载一次Cookie (逻辑不变) ---
	if err := chromedp.Run(browserCtx, t.loadCookiesAction()); err != nil {
		t.log.Errorf("浏览器启动时加载Cookie失败: %v", err)
		return err
	}
	t.log.Info("浏览器实例已启动，并成功加载初始Cookie。")

	// --- 【关键新增】浏览器预热环节 ---
	t.log.Info("正在执行浏览器预热...")
	warmupCtx, cancelWarmup := context.WithTimeout(browserCtx, 30*time.Second) // 给预热操作一个30秒的超时
	defer cancelWarmup()
	if err := chromedp.Run(warmupCtx,
		chromedp.Navigate(t.cfg.BaseUrl), // 访问一下根主页
		chromedp.Sleep(5*time.Second),    // 等待5秒，确保会话稳定
	); err != nil {
		t.log.Warnf("浏览器预热失败（不影响主流程）: %v", err)
	} else {
		t.log.Info("浏览器预热成功，准备开始执行主任务。")
	}
	// --- 预热环节结束 ---

	// --- 3. 获取待处理的视频列表 ---
	limit := 80
	// 【关键修改】调用全新的、逻辑正确的方法
	videos, err := t.videoUC.GetVideosForFirstCollection(ctx, limit)
	if err != nil {
		t.log.Errorf("获取首次采集视频列表失败: %v", err)
		return err
	}

	if len(videos) == 0 {
		t.log.Info("没有需要进行首次采集的视频。")
		return nil
	}
	t.log.Infof("获取到 %d 个需要进行首次采集的视频", len(videos))

	// --- 4. 遍历视频列表，调用业务层处理 ---
	for _, v := range videos {
		if v.AwemeDetailUrl == "" {
			t.log.Warnf("视频 %s 的 AwemeDetailUrl 为空，跳过采集", v.AwemeId)
			continue
		}
		dateCode := v.AwemePubTime.Format("20060102")

		// 【核心修改】直接调用业务方法，并传入【浏览器主上下文】
		// Biz层将负责在此浏览器实例中创建和销毁自己需要的标签页
		if err := t.headlessUC.FetchAndStoreVideoDetails(browserCtx, v.AwemeDetailUrl, v.AwemeId, dateCode); err != nil {
			t.log.Errorf("为视频 %s 处理详情时失败: %v", v.AwemeId, err)
		} else {
			t.log.Infof("成功下发视频 %s 的详情采集任务", v.AwemeId)
		}

		waitTime := rand.Intn(5)

		time.Sleep(time.Duration(waitTime) * time.Second)

		// 随机等待逻辑 (保持不变)
		//if i < len(videos)-1 {
		//	minWait := t.cfg.ThrottleMinWaitMs
		//	maxWait := t.cfg.ThrottleMaxWaitMs
		//	if minWait > 0 && maxWait > 0 && maxWait >= minWait {
		//		// 计算随机等待时间
		//		rand.Seed(time.Now().UnixNano())
		//		waitTime := rand.Intn(int(maxWait-minWait+1)) + int(minWait)
		//		t.log.Infof("任务节流：等待 %dms 后执行下一个任务...", waitTime)
		//		time.Sleep(time.Duration(waitTime) * time.Millisecond)
		//	}
		//}
	}

	t.log.Info("[无头浏览器-视频详情] 所有采集任务已处理完毕，浏览器将关闭。")
	return nil
}

// loadCookiesAction 是一个辅助方法，用于在浏览器启动时加载cookie
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

// newChromedpLogger 创建一个日志函数，该函数会过滤掉特定于chromedp的、已知的无害错误日志。
func (t *FetchVideoDetailsHeadlessTask) newChromedpLogger() func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		// 组合成完整的日志消息字符串
		msg := fmt.Sprintf(format, args...)

		// 【关键过滤逻辑】如果日志包含这个特定的错误，则直接忽略，不打印
		if strings.Contains(msg, "initiatorIPAddressSpace") {
			return
		}

		// 对于所有其他日志，正常通过任务本身的 logger 打印
		t.log.Info(msg)
	}
}
