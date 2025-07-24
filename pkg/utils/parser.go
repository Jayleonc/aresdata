package utils

import (
	"strconv"
	"strings"
	"time"
)

// TimeToPtr returns a pointer to a time.Time value.
func TimeToPtr(t time.Time) *time.Time {
	return &t
}

// ParseTimeRFC3339 parses an RFC3339 string to time.Time. If parsing fails, returns zero value.
func ParseTimeRFC3339(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ParseRangeStr 解析销量/销售额范围字符串，例如 "7500-1w" 或 "2.5w-5w"
// 返回范围的低值和高值
func ParseRangeStr(rangeStr string) (int64, int64) {
	if rangeStr == "" || rangeStr == "--" {
		return 0, 0
	}

	parts := strings.Split(rangeStr, "-")
	lowVal := parseValue(parts[0])
	if len(parts) > 1 {
		highVal := parseValue(parts[1])
		return lowVal, highVal
	}
	return lowVal, lowVal
}

// parseValue 解析单个值，处理 "w" (万) 单位
func parseValue(valStr string) int64 {
	valStr = strings.TrimSpace(valStr)
	if valStr == "" {
		return 0
	}

	var multiplier float64 = 1.0
	if strings.HasSuffix(valStr, "w") {
		multiplier = 10000.0
		valStr = strings.TrimSuffix(valStr, "w")
	}

	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0
	}

	return int64(val * multiplier)
}

func ParseUnitStrToInt64(str string) int64 {
	atoi, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return int64(atoi)
}
