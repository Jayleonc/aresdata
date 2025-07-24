// internal/biz/headless_uc.go
package biz

import (
	"aresdata/internal/conf"
	"aresdata/internal/data"
	"aresdata/internal/fetcher"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

// API路径常量化
const (
	apiPathTrend   = "/api/v3/aweme/detail/detail/trends"
	apiPathSummary = "/api/v3/aweme/detail/detail/sumData"
)

type HeadlessUsecase struct {
	repo            data.SourceDataRepo
	headlessFetcher *fetcher.HeadlessFetcher
	log             *log.Helper
	cfg             *conf.Feigua
}

func NewHeadlessUsecase(repo data.SourceDataRepo, hf *fetcher.HeadlessFetcher, cfg *conf.Data, logger log.Logger) *HeadlessUsecase {
	return &HeadlessUsecase{
		repo:            repo,
		headlessFetcher: hf,
		log:             log.NewHelper(log.With(logger, "module", "usecase/headless")),
		cfg:             cfg.GetFeigua(),
	}
}

// FetchAndStoreVideoDetails 在浏览器主上下文中，为单个视频创建一个标签页，并采集所有详情API。
func (uc *HeadlessUsecase) FetchAndStoreVideoDetails(browserCtx context.Context, awemeDetailURL, awemeID, dateCode string) error {
	if uc.cfg == nil || uc.cfg.BaseUrl == "" {
		return fmt.Errorf("feigua base_url not configured")
	}

	entryURL := uc.cfg.BaseUrl + "/app/" + awemeDetailURL

	// 定义我们这次业务需要采集的所有API
	targetAPIs := []string{apiPathTrend, apiPathSummary}

	// 【核心】为本次视频处理，创建唯一一个新标签页
	tabCtx, cancelTab := chromedp.NewContext(browserCtx)
	defer cancelTab()

	// 为这个标签页的操作设置一个独立的超时
	opCtx, cancelOp := context.WithTimeout(tabCtx, time.Duration(uc.cfg.Timeout)*time.Second)
	defer cancelOp()

	// 调用Fetcher，在新建的标签页中执行“导航一次，监听多个”
	apiResults, err := uc.headlessFetcher.CaptureAPIs(opCtx, entryURL, targetAPIs)
	if err != nil {
		return fmt.Errorf("为视频 %s 采集详情失败: %w", awemeID, err)
	}

	if len(apiResults) == 0 {
		return fmt.Errorf("为视频 %s 未捕获到任何API响应", awemeID)
	}

	// 串行处理和存储捕获到的数据
	for apiPath, rawContent := range apiResults {
		var dataType string
		switch apiPath {
		case apiPathTrend:
			dataType = "video_trend_headless"
		case apiPathSummary:
			dataType = "video_summary_headless"
		default:
			uc.log.Warnf("捕获到未配置的API响应: %s", apiPath)
			continue
		}

		sourceData := &data.SourceData{
			ProviderName: "headless_chrome",
			DataType:     dataType,
			RawContent:   rawContent,
			Date:         dateCode,
			EntityId:     awemeID,
			Status:       0,
			FetchedAt:    time.Now(),
			RequestUrl:   apiPath, // 只记录相对路径
		}
		if _, err := uc.repo.Save(opCtx, data.CopySourceDataToDTO(sourceData)); err != nil {
			// 只记录错误，不中断其他数据的保存
			uc.log.Errorf("保存 %s 数据失败: %v", dataType, err)
		}
	}

	uc.log.Infof("成功处理并存储了视频 %s 的 %d 个详情API", awemeID, len(apiResults))
	return nil
}
