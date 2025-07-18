package utils

import (
	"strconv"
	"strings"
)

// ParseCountStr 将 "5.2w" 这样的字符串转换为 int64
func ParseCountStr(s string) (int64, error) {
	if s == "" || s == "--" {
		return 0, nil
	}
	s = strings.ToLower(s)
	var multiplier float64 = 1
	if strings.HasSuffix(s, "w") {
		multiplier = 10000
		s = strings.TrimSuffix(s, "w")
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(val * multiplier), nil
}

// ParseSalesCount 将 "7.5w-10w" 这样的范围字符串解析为平均销量（或最大值）
func ParseSalesCount(s string) (int64, error) {
	if s == "" || s == "--" {
		return 0, nil
	}
	parts := strings.Split(s, "-")
	// 我们取范围的第二个值作为代表
	lastPart := parts[len(parts)-1]
	return ParseCountStr(lastPart)
}

// ParseSalesGmv 将 "250w-500w" 这样的范围字符串解析为平均GMV（单位：分）
func ParseSalesGmv(s string) (int64, error) {
	if s == "" || s == "--" {
		return 0, nil
	}
	gmv, err := ParseSalesCount(s)
	if err != nil {
		return 0, err
	}
	// 假设原始单位是元，转换为分
	return gmv * 100, nil
}
