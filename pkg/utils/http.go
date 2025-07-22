package utils

import (
	"fmt"
	"net/http"
	"strings"
)

// RequestToCurl 将 http.Request 对象转换为一个 curl 命令字符串，用于调试
func RequestToCurl(req *http.Request) string {
	var parts []string

	// 1. 添加 'curl' 命令和 URL
	// 使用 -g 开关来防止 curl 对 URL 中的括号、方括号等进行 globbing
	parts = append(parts, "curl -g")
	parts = append(parts, fmt.Sprintf(`'%s'`, req.URL.String()))

	// 2. 添加请求方法
	if req.Method != "GET" {
		parts = append(parts, fmt.Sprintf("-X %s", req.Method))
	}

	// 3. 添加所有请求头
	for key, values := range req.Header {
		for _, value := range values {
			parts = append(parts, fmt.Sprintf(`-H '%s: %s'`, key, value))
		}
	}

	// 4. 如果是 gzip 压缩，添加 --compressed 标志
	// curl 会自动处理解压
	if req.Header.Get("Accept-Encoding") == "gzip, deflate" {
		parts = append(parts, "--compressed")
	}

	return strings.Join(parts, " \\\n  ")
}
