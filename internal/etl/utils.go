package etl

import (
	"encoding/json"
	"time"
)

func toInt64(n json.Number) int64 {
	i, _ := n.Int64()
	return i
}

func toFloat64(n json.Number) float64 {
	f, _ := n.Float64()
	return f
}

// ProcessError 自定义错误类型，方便记录日志
type ProcessError struct {
	Msg      string
	SourceID int64
	Err      error
}

func (e *ProcessError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

// getPeriodDates 生成榜单周期的起止日期和主日期
func getPeriodDates(period, datecode string) (startDate, endDate, rankDate string) {
	// 默认都为 datecode
	startDate, endDate, rankDate = datecode, datecode, datecode
	switch period {
	case "day":
		// 都为当天
		startDate, endDate, rankDate = datecode, datecode, datecode
	case "week":
		d, err := time.Parse("20060102", datecode)
		if err != nil {
			return
		}
		weekday := int(d.Weekday())
		if weekday == 0 {
			weekday = 7 // 周日
		}
		monday := d.AddDate(0, 0, -weekday+1)
		sunday := d.AddDate(0, 0, 7-weekday)
		startDate = monday.Format("20060102")
		endDate = sunday.Format("20060102")
		rankDate = startDate
	case "month":
		d, err := time.Parse("20060102", datecode)
		if err != nil {
			return
		}
		first := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
		var nextMonth time.Time
		if d.Month() == 12 {
			nextMonth = time.Date(d.Year()+1, 1, 1, 0, 0, 0, 0, d.Location())
		} else {
			nextMonth = time.Date(d.Year(), d.Month()+1, 1, 0, 0, 0, 0, d.Location())
		}
		last := nextMonth.AddDate(0, 0, -1)
		startDate = first.Format("20060102")
		endDate = last.Format("20060102")
		rankDate = startDate
	}
	return
}
