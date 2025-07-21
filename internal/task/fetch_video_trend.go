package task

import (
	"aresdata/internal/biz"
	"aresdata/internal/data"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"sync"
)

// FetchVideoTrendTask 负责每日拉取视频趋势数据的任务
type FetchVideoTrendTask struct {
	videoRepo  data.VideoRepo
	fetcherBiz *biz.FetcherUsecase
	log        *log.Helper
}

// NewFetchVideoTrendTask 构造任务实例
func NewFetchVideoTrendTask(repo data.VideoRepo, biz *biz.FetcherUsecase, logger log.Logger) *FetchVideoTrendTask {
	return &FetchVideoTrendTask{
		videoRepo:  repo,
		fetcherBiz: biz,
		log:        log.NewHelper(logger),
	}
}

func (t *FetchVideoTrendTask) Name() string {
	return ProcessVideoTrend
}

// Run 任务的执行入口
func (t *FetchVideoTrendTask) Run() {
	t.log.Info("开始执行 [每日视频趋势] 拉取任务...")
	ctx := context.Background()

	// 1. 获取过去30天内活跃的视频ID
	awemeIds, err := t.videoRepo.FindRecentActiveAwemeIds(ctx, 30)
	if err != nil {
		t.log.Errorf("获取活跃视频ID失败: %v", err)
		return
	}
	t.log.Infof("发现 %d 个活跃视频需要同步趋势数据", len(awemeIds))

	// 2. 并发拉取和保存数据 (可以设置并发度)
	var wg sync.WaitGroup
	concurrency := 5 // 同时处理5个视频
	idChannel := make(chan string, len(awemeIds))

	for _, id := range awemeIds {
		idChannel <- id
	}
	close(idChannel)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for awemeId := range idChannel {
				t.log.Infof("正在处理视频: %s", awemeId)
				err := t.fetcherBiz.FetchAndSaveVideoTrend(ctx, awemeId)
				if err != nil {
					t.log.Errorf("处理视频 %s 失败: %v", awemeId, err)
				}
			}
		}()
	}

	wg.Wait()
	t.log.Info("[每日视频趋势] 拉取任务执行完毕。")
}
