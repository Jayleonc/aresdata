package etl

import (
	v1 "aresdata/api/v1"
	"aresdata/internal/data"
	"aresdata/pkg/crypto"
	"context"
	"encoding/json"
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

// VideoRankProcessor implements Processor for video rank data.
type VideoRankProcessor struct {
	videoRankRepo  data.VideoRankRepo
	sourceDataRepo data.SourceDataRepo
	videoRepo      data.VideoRepo // 新增
}

func NewVideoRankProcessor(vrRepo data.VideoRankRepo, sdRepo data.SourceDataRepo, vRepo data.VideoRepo) *VideoRankProcessor {
	return &VideoRankProcessor{
		videoRankRepo:  vrRepo,
		sourceDataRepo: sdRepo,
		videoRepo:      vRepo, // 新增
	}
}

func (p *VideoRankProcessor) Process(ctx context.Context, rawData *v1.SourceData) error {
	// Step 1: Parse raw JSON to get Data and Rnd
	var payload struct {
		Data    string `json:"Data"`
		Encrypt bool   `json:"Encrypt"`
		Rnd     string `json:"Rnd"`
	}
	if err := json.Unmarshal([]byte(rawData.RawContent), &payload); err != nil {
		return &ProcessError{Msg: "failed to parse raw_content payload", SourceID: rawData.Id, Err: err}
	}

	var decrypted string
	if payload.Encrypt {
		if payload.Data == "" || len(payload.Rnd) < 8 {
			return &ProcessError{Msg: "missing data or invalid rnd field", SourceID: rawData.Id}
		}
		var err error
		decrypted, err = crypto.FeiguaDecrypt(payload.Data, payload.Rnd)
		if err != nil {
			return &ProcessError{Msg: "failed to decrypt data", SourceID: rawData.Id, Err: err}
		}
	} else {
		// 如果未加密，Data字段本身就是JSON字符串
		decrypted = payload.Data
	}

	// Step 3: Unmarshal decrypted JSON to a strongly-typed struct
	var listPayload struct {
		List []FeiguaVideoRankItem `json:"List"`
	}
	if err := json.Unmarshal([]byte(decrypted), &listPayload); err != nil {
		return &ProcessError{Msg: "failed to unmarshal decrypted list", SourceID: rawData.Id, Err: err}
	}

	if len(listPayload.List) == 0 {
		// 即使列表为空，也标记为已处理，防止重复执行
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
		// 2a. 更新/插入 Video 维度表
		videoDim := &data.Video{
			AwemeId:       item.AwemeDto.AwemeId,
			AwemeDesc:     item.AwemeDto.AwemeDesc,
			AwemeCoverUrl: item.AwemeDto.AwemeCoverUrl,
			AwemePubTime:  pubTime,
			BloggerId:     toInt64(item.BloggerDto.BloggerId),
		}
		if err := p.videoRepo.Upsert(ctx, videoDim); err != nil {
			// 记录错误，但通常不应该中断整个ETL流程
			if pLogger, ok := any(p).(interface {
				logf(format string, args ...any)
			}); ok {
				pLogger.logf("failed to upsert video dimension for awemeId %s: %v", videoDim.AwemeId, err)
			}
		}
		// TODO: 2b. 将来在这里添加对 Product 和 Blogger 维度表的 Upsert
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
