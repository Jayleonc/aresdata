package etl

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"aresdata/pkg/crypto"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"strings"
	"time"
)

type FeiguaAwemeDto struct {
	AwemeId       string `json:"awemeId"`
	AwemeCoverUrl string `json:"awemeCoverUrl"`
	AwemeDesc     string `json:"awemeDesc"`
	AwemePubTime  string `json:"awemePubTime"`
	AwemeShareUrl string `json:"awemeShareUrl"`
	DurationStr   string `json:"durationStr"`
	AwemeScoreStr string `json:"awemeScoreStr"`
}

type FeiguaGoodsDto struct {
	Gid             string      `json:"gid"`
	Title           string      `json:"title"`
	CoverUrl        string      `json:"coverUrl"`
	PriceRange      string      `json:"priceRange"`
	Price           json.Number `json:"price"`
	CosRatio        string      `json:"cosRatio"`
	CommissionPrice string      `json:"commissionPrice"`
	ShopName        string      `json:"shopName"`
	DouyinBrandName string      `json:"douyinBrandName"`
	CateNames       string      `json:"cateNames"`
}

type FeiguaBloggerDto struct {
	BloggerId     json.Number `json:"bloggerId"`
	BloggerUid    string      `json:"bloggerUid"`
	BloggerName   string      `json:"bloggerName"`
	BloggerAvatar string      `json:"bloggerAvatar"`
	FansNum       json.Number `json:"fansNum"`
	Tag           string      `json:"tag"`
}

type FeiguaVideoRankItem struct {
	RankNum      json.Number      `json:"rankNum"`
	AwemeDto     FeiguaAwemeDto   `json:"baseAwemeDto"`
	GoodsDto     FeiguaGoodsDto   `json:"baseGoodsDto"`
	BloggerDto   FeiguaBloggerDto `json:"baseBloggerDto"`
	SalesCount   string           `json:"salesCount"`
	TotalSales   string           `json:"totalSales"`
	LikeCountInc string           `json:"likeCountInc"`
	PlayCountInc string           `json:"playCountInc"`
}

// FeiguaVideoRankResponse 统一了API响应结构，嵌套通用响应字段
// Status, Msg, Code 字段通过 FeiguaBaseResponse 提供
// Encrypt, Rnd, Data 为加密数据相关字段
type FeiguaVideoRankResponse struct {
	FeiguaBaseResponse
	Encrypt bool   `json:"Encrypt"`
	Rnd     string `json:"Rnd"`
	Data    string `json:"Data"`
}

// VideoRankProcessor implements Processor for video rank data.
type VideoRankProcessor struct {
	videoRankRepo  data.VideoRankRepo
	sourceDataRepo data.SourceDataRepo
	videoRepo      data.VideoRepo
	productRepo    data.ProductRepo // 新增
	bloggerRepo    data.BloggerRepo // 新增
	log            *log.Helper
}

func NewVideoRankProcessor(vrRepo data.VideoRankRepo, sdRepo data.SourceDataRepo, vRepo data.VideoRepo, pRepo data.ProductRepo, bRepo data.BloggerRepo, logger log.Logger) *VideoRankProcessor {
	return &VideoRankProcessor{
		videoRankRepo:  vrRepo,
		sourceDataRepo: sdRepo,
		videoRepo:      vRepo,
		productRepo:    pRepo,
		bloggerRepo:    bRepo,
		log:            log.NewHelper(log.With(logger, "module", "etl/video-rank")),
	}
}

func (p *VideoRankProcessor) Process(ctx context.Context, rawData *v1.SourceData) error {
	// Step 1: 一次性解析包含状态和数据的完整响应
	var resp FeiguaVideoRankResponse
	if err := json.Unmarshal([]byte(rawData.RawContent), &resp); err != nil {
		return &ProcessError{Msg: "failed to parse raw_content payload", SourceID: rawData.Id, Err: err}
	}

	// Step 2: 预检API业务状态
	if !resp.Status {
		logMsg := fmt.Sprintf("API returned error status: Code=%d, Msg=%s", resp.Code, resp.Msg)
		return p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, logMsg)
	}

	// Step 3: 解密数据
	var decrypted string
	if resp.Encrypt {
		if resp.Data == "" || len(resp.Rnd) < 8 {
			return &ProcessError{Msg: "missing encrypted data or invalid rnd field", SourceID: rawData.Id}
		}
		var err error
		decrypted, err = crypto.FeiguaDecrypt(resp.Data, resp.Rnd)
		if err != nil {
			return &ProcessError{Msg: "failed to decrypt data", SourceID: rawData.Id, Err: err}
		}
	} else {
		decrypted = resp.Data
	}

	// Step 4: 解析解密后的具体业务数据
	var listPayload struct {
		List []FeiguaVideoRankItem `json:"List"`
	}
	if err := json.Unmarshal([]byte(decrypted), &listPayload); err != nil {
		// 注意：这里的错误可能是因为解密后的内容不是预期的JSON，例如内容为空
		// 我们需要更健壮地处理这种情况
		p.sourceDataRepo.UpdateStatusAndLog(ctx, rawData.Id, -1, "failed to unmarshal decrypted list payload: "+err.Error())
		return &ProcessError{Msg: "failed to unmarshal decrypted list", SourceID: rawData.Id, Err: err}
	}

	if len(listPayload.List) == 0 {
		return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
	}

	// Step 4: Map each item to data.VideoRank
	ranksToCreate := make([]*data.VideoRank, 0, len(listPayload.List))
	for _, item := range listPayload.List {
		// 解析时间
		pubTime, _ := time.Parse("2006/01/02 15:04:05", item.AwemeDto.AwemePubTime)
		// 从 source_data 获取榜单周期和日期信息
		period := strings.Replace(rawData.DataType, "video_rank_", "", 1)
		datecode := rawData.Date
		startDate, endDate, rankDate := getPeriodDates(period, datecode)

		vr := &data.VideoRank{
			// Rank Info
			RankNum:    int(toInt64(item.RankNum)),
			PeriodType: period,
			RankDate:   rankDate,
			StartDate:  startDate,
			EndDate:    endDate,

			// Aweme Info
			AwemeId:       item.AwemeDto.AwemeId,
			AwemeCoverUrl: item.AwemeDto.AwemeCoverUrl,
			AwemeDesc:     item.AwemeDto.AwemeDesc,
			AwemePubTime:  pubTime,
			AwemeShareUrl: item.AwemeDto.AwemeShareUrl,
			DurationStr:   item.AwemeDto.DurationStr,
			AwemeScoreStr: item.AwemeDto.AwemeScoreStr,

			// Goods Info
			GoodsId:         item.GoodsDto.Gid,
			GoodsTitle:      item.GoodsDto.Title,
			GoodsCoverUrl:   item.GoodsDto.CoverUrl,
			GoodsPriceRange: item.GoodsDto.PriceRange,
			GoodsPrice:      toFloat64(item.GoodsDto.Price),
			CosRatio:        item.GoodsDto.CosRatio,
			CommissionPrice: item.GoodsDto.CommissionPrice,
			ShopName:        item.GoodsDto.ShopName,
			BrandName:       item.GoodsDto.DouyinBrandName,
			CategoryNames:   item.GoodsDto.CateNames,

			// Blogger Info
			BloggerId:      int(toInt64(item.BloggerDto.BloggerId)),
			BloggerUid:     item.BloggerDto.BloggerUid,
			BloggerName:    item.BloggerDto.BloggerName,
			BloggerAvatar:  item.BloggerDto.BloggerAvatar,
			BloggerFansNum: int(toInt64(item.BloggerDto.FansNum)),
			BloggerTag:     item.BloggerDto.Tag,

			// Stat Info
			SalesCountStr:   item.SalesCount,
			TotalSalesStr:   item.TotalSales,
			LikeCountIncStr: item.LikeCountInc,
			PlayCountIncStr: item.PlayCountInc,
		}
		// 存储原始JSON，便于追溯
		//rawJSONBytes, _ := json.Marshal(item)
		//vr.RawJson = string(rawJSONBytes)
		ranksToCreate = append(ranksToCreate, vr)

		// --- 维度表更新 ---
		// 1. 更新/插入 Video 维度表 (修复：使用为Rank定制的Upsert)
		videoDim := &data.Video{
			AwemeId:       item.AwemeDto.AwemeId,
			AwemeDesc:     item.AwemeDto.AwemeDesc,
			AwemeCoverUrl: item.AwemeDto.AwemeCoverUrl,
			AwemePubTime:  pubTime,
			BloggerId:     toInt64(item.BloggerDto.BloggerId),
		}
		if err := p.videoRepo.UpsertFromRank(ctx, videoDim); err != nil {
			p.log.Errorf("failed to upsert video dimension for awemeId %s: %v", videoDim.AwemeId, err)
		}

		// 2. 更新/插入 Product 维度表
		productDim := &data.Product{
			GoodsId:         item.GoodsDto.Gid,
			GoodsTitle:      item.GoodsDto.Title,
			GoodsCoverUrl:   item.GoodsDto.CoverUrl,
			GoodsPriceRange: item.GoodsDto.PriceRange,
			GoodsPrice:      toFloat64(item.GoodsDto.Price),
			CosRatio:        item.GoodsDto.CosRatio,
			CommissionPrice: item.GoodsDto.CommissionPrice,
			ShopName:        item.GoodsDto.ShopName,
			BrandName:       item.GoodsDto.DouyinBrandName,
			CategoryNames:   item.GoodsDto.CateNames,
		}
		if err := p.productRepo.Upsert(ctx, productDim); err != nil {
			p.log.Errorf("failed to upsert product dimension for goodsId %s: %v", productDim.GoodsId, err)
		}

		// 3. 更新/插入 Blogger 维度表
		bloggerDim := &data.Blogger{
			BloggerId:      toInt64(item.BloggerDto.BloggerId),
			BloggerUid:     item.BloggerDto.BloggerUid,
			BloggerName:    item.BloggerDto.BloggerName,
			BloggerAvatar:  item.BloggerDto.BloggerAvatar,
			BloggerFansNum: toInt64(item.BloggerDto.FansNum),
			BloggerTag:     item.BloggerDto.Tag,
		}
		if err := p.bloggerRepo.Upsert(ctx, bloggerDim); err != nil {
			p.log.Errorf("failed to upsert blogger dimension for bloggerId %d: %v", bloggerDim.BloggerId, err)
		}

	}

	// Step 5: Batch insert
	var ranksToCreateDTO = make([]*v1.VideoRankDTO, 0, len(ranksToCreate))
	for _, r := range ranksToCreate {
		ranksToCreateDTO = append(ranksToCreateDTO, data.CopyVideoRankToDTO(r))
	}

	if err := p.videoRankRepo.BatchCreate(ctx, ranksToCreateDTO); err != nil {
		return &ProcessError{Msg: "failed to batch create video ranks", SourceID: rawData.Id, Err: err}
	}

	// Step 6: Update source data status
	return p.sourceDataRepo.UpdateStatus(ctx, rawData.Id, 1)
}
