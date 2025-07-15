package data

import (
	v1 "aresdata/api/v1"
	"context"
	"time"

	"aresdata/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SourceData 是数据库表的GORM模型
type SourceData struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	ProviderName string    `gorm:"type:varchar(50);not null"`
	DataType     string    `gorm:"type:varchar(100);not null"`
	RawContent   string    `gorm:"type:text"` // 使用 text 以存储大量JSON
	FetchedAt    time.Time `gorm:"default:current_timestamp"`
	Status       int32     `gorm:"default:0"`
	EntityID     string    `gorm:"type:varchar(255)"`
}

func (SourceData) TableName() string {
	return "source_data"
}

type sourceDataRepo struct {
	data *Data
	log  *log.Helper
}

// NewSourceDataRepo .
func NewSourceDataRepo(data *Data, logger log.Logger) biz.SourceDataRepo {
	return &sourceDataRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "repo/source-data")),
	}
}

// Save 实现了biz层的接口，负责将数据写入数据库
func (r *sourceDataRepo) Save(ctx context.Context, s *v1.SourceData) (*v1.SourceData, error) {
	model := &SourceData{
		ProviderName: s.ProviderName,
		DataType:     s.DataType,
		RawContent:   s.RawContent,
		EntityID:     s.EntityId,
		Status:       s.Status,
	}

	if err := r.data.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}

	return &v1.SourceData{
		Id:           model.ID,
		ProviderName: model.ProviderName,
		DataType:     model.DataType,
		RawContent:   model.RawContent,
		FetchedAt:    timestamppb.New(model.FetchedAt),
		Status:       model.Status,
		EntityId:     model.EntityID,
	}, nil
}
