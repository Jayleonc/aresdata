// internal/task/remedy_video_details_headless.go
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
	"strings"
	"time"
)

// RemedyVideoDetailsHeadlessTask 负责修复部分采集失败的视频详情数据
type RemedyVideoDetailsHeadlessTask struct {
	log        *log.Helper
	videoUC    *biz.VideoUsecase
	headlessUC *biz.HeadlessUsecase
	cfg        *conf.Feigua
}

// NewRemedyVideoDetailsHeadlessTask 创建一个新的修复任务实例
func NewRemedyVideoDetailsHeadlessTask(
	logger log.Logger,
	videoUC *biz.VideoUsecase,
	headlessUC *biz.HeadlessUsecase,
	cfg *conf.Data,
) *RemedyVideoDetailsHeadlessTask {
	return &RemedyVideoDetailsHeadlessTask{
		log:        log.NewHelper(log.With(logger, "module", "task.remedy_video_details_headless")),
		videoUC:    videoUC,
		headlessUC: headlessUC,
		cfg:        cfg.GetFeigua(),
	}
}

// Name 返回任务的名称
func (t *RemedyVideoDetailsHeadlessTask) Name() string {
	return RemedyVideoDetailsHeadless
}

// Run 执行任务的核心逻辑
func (t *RemedyVideoDetailsHeadlessTask) Run(ctx context.Context, args ...string) error {
	t.log.Info("开始执行 [无头浏览器-视频详情修复] 任务...")

	// 1. 获取过去24小时内，部分采集失败的视频列表
	limit := 50 // 每次修复50个
	hoursAgo := 24
	videosToRemedy, err := t.videoUC.GetPartiallyCollectedVideos(ctx, hoursAgo, limit)
	if err != nil {
		t.log.Errorf("获取待修复视频列表失败: %v", err)
		return err
	}

	if len(videosToRemedy) == 0 {
		t.log.Info("没有需要修复的视频详情数据。")
		return nil
	}

	t.log.Infof("获取到 %d 个需要修复详情的视频", len(videosToRemedy))

	// 2. 启动唯一的浏览器实例 (与主采集任务相同)
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
	browserCtx, cancel = context.WithTimeout(browserCtx, 30*time.Minute)
	defer cancel()
	if err := chromedp.Run(browserCtx, t.loadCookiesAction()); err != nil {
		t.log.Errorf("浏览器启动时加载Cookie失败: %v", err)
		return err
	}
	t.log.Info("浏览器实例已启动，并成功加载初始Cookie。")

	// 3. 遍历待修复列表，下发采集任务
	for _, v := range videosToRemedy {
		dateCode := v.AwemePubTime.Format("20060102")
		if err := t.headlessUC.FetchAndStoreVideoDetails(browserCtx, v.AwemeDetailUrl, v.AwemeId, dateCode); err != nil {
			t.log.Errorf("为视频 %s 执行修复任务失败: %v", v.AwemeId, err)
		} else {
			t.log.Infof("成功为视频 %s 下发修复任务", v.AwemeId)
		}
		// 可选: 随机等待
	}

	t.log.Info("[无头浏览器-视频详情修复] 所有修复任务已下发完毕，浏览器将关闭。")
	return nil
}

// loadCookiesAction 是一个辅助方法，用于在浏览器启动时加载cookie
func (t *RemedyVideoDetailsHeadlessTask) loadCookiesAction() chromedp.Action {
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
			return nil
		}
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
func (t *RemedyVideoDetailsHeadlessTask) newChromedpLogger() func(string, ...interface{}) {
	return func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if strings.Contains(msg, "initiatorIPAddressSpace") {
			return
		}
		t.log.Info(msg)
	}
}
