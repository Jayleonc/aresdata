package data

import (
	"github.com/Jayleonc/aresdata/internal/conf"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewRedisClient,
	NewSourceDataRepo,
	NewVideoRankRepo,
	NewVideoRepo,
	NewVideoTrendRepo,
	NewProductRepo,
	NewBloggerRepo,
)

// Data .
type Data struct {
	db          *gorm.DB
	redis       redis.Cmdable
	Logger      *log.Helper
	DataSources []*conf.DataSource // assume this exists
}

// NewData .
func NewData(c *conf.Data, redisClient redis.Cmdable, logger log.Logger) (*Data, func(), error) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	gormConfig := &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().In(loc)
		},
		DefaultTransactionTimeout: 10 * time.Second,
	}
	db, err := gorm.Open(postgres.Open(c.Database.Source), gormConfig)
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	// Set the timezone for the current session.
	_, err = sqlDB.Exec("SET TIME ZONE 'Asia/Shanghai'")
	if err != nil {
		return nil, nil, err
	}

	helper := log.NewHelper(logger)
	cleanup := func() {
		helper.Info("closing the data resources")
		sqlDB.Close()
		_ = redisClient.(*redis.Client).Close()
	}

	db.AutoMigrate(&SourceData{}, &VideoRank{}, &Video{}, &VideoTrend{}, &Product{}, &Blogger{})

	return &Data{
		db:     db,
		redis:  redisClient,
		Logger: helper,
	}, cleanup, nil
}

// NewRedisClient 初始化 Redis 客户端
func NewRedisClient(conf *conf.Data) redis.Cmdable {
	return redis.NewClient(&redis.Options{
		Addr:         conf.Redis.Addr,
		Network:      conf.Redis.Network,
		Password:     conf.Redis.Password,
		ReadTimeout:  conf.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: conf.Redis.WriteTimeout.AsDuration(),
	})
}
