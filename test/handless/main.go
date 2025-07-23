// final_test/main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {
	// 【关键】请将从榜单JSON的 "AwemeDetailUrl" 字段中复制出的完整URL，粘贴到这里
	// 注意：需要拼接上 BaseURL
	baseURL := "http://118.31.20.20:8080/app/"
	awemeDetailURLPart := "#/video-detail/index?awemeId=7528437837746769179&dateCode=20250718&ts=1753251065&sign=0b0d8824618eb915d6d0871269941ad3"
	signedURL := baseURL + awemeDetailURLPart

	// 我们自己拼接的目标API URL
	targetAPIURL := "http://118.31.20.20:8080/api/v3/aweme/detail/detail/trends?awemeId=7528437837746769179&dateCode=20250718&period=30&type=1"

	// 1. 设置 chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	var responseBody string

	// 2. 执行全新的“两步走”任务
	err := chromedp.Run(ctx,
		loadCookiesAction("configs/assets/cookies.json"), // 步骤1: 加载Cookie

		chromedp.Navigate(signedURL),
		chromedp.Sleep(9*time.Second), // 等待页面上的JS完成所有“授权”操作

		chromedp.Navigate(targetAPIURL),
		// 当直接访问一个返回JSON的URL时，其内容会作为纯文本显示在body中
		chromedp.Text("body", &responseBody, chromedp.ByQuery),
	)

	if err != nil {
		log.Fatalf("chromedp.Run 执行失败: %v", err)
	}

	// 3. 检查结果
	if responseBody != "" {
		log.Println("!!! [验证成功] “两步走”方案成功获取到数据！!!!")
		log.Println("trends API 响应内容:")

		// 格式化打印JSON
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, []byte(responseBody), "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
		} else {
			fmt.Println(responseBody)
		}
	} else {
		log.Println("!!! [验证失败] 未能获取到响应内容。!!!")
	}

	log.Println("测试结束，浏览器将在1分钟后自动关闭...")
	time.Sleep(1 * time.Minute)
}

func loadCookiesAction(cookiePath string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		cookiesBytes, err := ioutil.ReadFile(cookiePath)
		if err != nil {
			return fmt.Errorf("读取 cookie 文件失败 %s: %w", cookiePath, err)
		}
		log.Printf("从文件 %s 中加载 Cookies...", cookiePath)

		processedCookiesStr := strings.ReplaceAll(string(cookiesBytes), `"sameSite": "unspecified"`, `"sameSite": "Lax"`)
		cookiesBytes = []byte(processedCookiesStr)

		cookies := []*network.CookieParam{}
		if err := json.Unmarshal(cookiesBytes, &cookies); err != nil {
			return fmt.Errorf("解析 cookie JSON 失败: %w", err)
		}

		log.Printf("成功解析 %d 个 Cookies，准备设置...", len(cookies))
		return network.SetCookies(cookies).Do(ctx)
	})
}
