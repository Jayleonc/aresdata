package etl

// FeiguaBaseResponse 通过嵌套(embedding)的方式，为所有飞瓜API响应提供通用字段
type FeiguaBaseResponse struct {
	Status bool   `json:"Status"`
	Msg    string `json:"Msg"`
	Code   int    `json:"Code"`
}
