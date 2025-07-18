package data

import (
	v1 "aresdata/api/v1"
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// SourceDataRepo 是Biz层依赖的Data层接口，由 data/source_data.go 实现
type SourceDataRepo interface {
	Save(context.Context, *v1.SourceData) (*v1.SourceData, error)
	UpdateStatus(ctx context.Context, id int64, status int32) error

	FindUnprocessed(ctx context.Context, dataType string) ([]*v1.SourceData, error)
	// UpdateStatusAndLog 原子性地更新状态和处理日志
	UpdateStatusAndLog(ctx context.Context, id int64, status int32, log string) error
}

// SourceData is the GORM model for storing raw data from various providers.
// It is the Data Object (DO).
// SourceData is the GORM model for storing raw data from various providers.
type SourceData struct {
	ID            int64     `gorm:"primaryKey"`
	ProviderName  string    `gorm:"type:varchar(255);not null;index"`
	DataType      string    `gorm:"type:varchar(255);not null;index"`
	EntityId      string    `gorm:"type:varchar(255);index"`  // 可选，关联的主要实体ID，不同的数据类型可能有不同的ID
	Status        int32     `gorm:"not null;default:0;index"` // 0: unprocessed, 1: processed, -1: error
	FetchedAt     time.Time `gorm:"autoCreateTime"`
	Date          string    `gorm:"type:varchar(10);not null;index"`
	RawContent    string    `gorm:"type:text"`
	ProcessingLog string    `gorm:"type:text"`          // 存储ETL处理过程中的错误信息
	Retries       int       `gorm:"not null;default:0"` // 重试次数

	// --- 新增的请求上下文元数据 ---
	RequestMethod  string `gorm:"type:varchar(10)"` // "GET", "POST", etc.
	RequestUrl     string `gorm:"type:text"`
	RequestParams  string `gorm:"type:text"` // 存储 Query 或 Body 的 JSON 字符串
	RequestHeaders string `gorm:"type:text"` // 存储请求头的 JSON 字符串
}

func (SourceData) TableName() string {
	return "source_data"
}

type sourceDataRepo struct {
	data *Data
	log  *log.Helper
}

// NewSourceDataRepo .
func NewSourceDataRepo(data *Data, logger log.Logger) SourceDataRepo {
	return &sourceDataRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "repo/source-data")),
	}
}

// Save 实现了biz层的接口，负责将数据写入数据库
func copySourceDataToDO(s *v1.SourceData) *SourceData {
	var fetchedAt time.Time
	if s.FetchedAt != nil {
		fetchedAt = s.FetchedAt.AsTime()
	}
	return &SourceData{
		ID:             s.Id,
		ProviderName:   s.ProviderName,
		DataType:       s.DataType,
		EntityId:       s.EntityId,
		Status:         s.Status,
		FetchedAt:      fetchedAt,
		Date:           s.Date,
		RawContent:     s.RawContent,
		ProcessingLog:  s.ProcessingLog,
		Retries:        int(s.Retries),
		RequestMethod:  s.RequestMethod,
		RequestUrl:     s.RequestUrl,
		RequestParams:  s.RequestParams,
		RequestHeaders: s.RequestHeaders,
	}
}

func copySourceDataToDTO(s *SourceData) *v1.SourceData {
	return &v1.SourceData{
		Id:             s.ID,
		ProviderName:   s.ProviderName,
		DataType:       s.DataType,
		EntityId:       s.EntityId,
		Status:         s.Status,
		FetchedAt:      timestamppb.New(s.FetchedAt),
		Date:           s.Date,
		RawContent:     s.RawContent,
		ProcessingLog:  s.ProcessingLog,
		Retries:        int32(s.Retries),
		RequestMethod:  s.RequestMethod,
		RequestUrl:     s.RequestUrl,
		RequestParams:  s.RequestParams,
		RequestHeaders: s.RequestHeaders,
	}
}

func (r *sourceDataRepo) Save(ctx context.Context, s *v1.SourceData) (*v1.SourceData, error) {
	model := copySourceDataToDO(s)
	if err := r.data.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return copySourceDataToDTO(model), nil
}

// UpdateStatus 更新原始数据的处理状态
func (r *sourceDataRepo) UpdateStatus(ctx context.Context, id int64, status int32) error {
	return r.data.db.WithContext(ctx).Model(&SourceData{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateStatusAndLog 更新原始数据的处理状态和日志
func (r *sourceDataRepo) UpdateStatusAndLog(ctx context.Context, id int64, status int32, log string) error {
	return r.data.db.WithContext(ctx).Model(&SourceData{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":         status,
		"processing_log": log,
	}).Error
}

// FindUnprocessed 查找所有未处理的数据
func (r *sourceDataRepo) FindUnprocessed(ctx context.Context, dataType string) ([]*v1.SourceData, error) {
	var models []*SourceData
	if err := r.data.db.WithContext(ctx).Where("status = ? AND data_type = ?", 0, dataType).Find(&models).Error; err != nil {
		return nil, err
	}
	var result []*v1.SourceData
	for _, m := range models {
		result = append(result, copySourceDataToDTO(m))
	}
	return result, nil
}
