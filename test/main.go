// temp_test/main.go
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {
	// 使用您正在测试的 awemeID
	awemeID := "7528437837746769179"
	cookie := "DYHHB_Session_2=4F390ED5B7E47D2FA5E7073C3BA4994F; il=2CD2D1033AE2554CF43E3DD40B5D08AC; mu=; body_collapsed=0"
	baseURL := "http://121.40.63.195:8085"

	client := &http.Client{Timeout: 30 * time.Second}

	// --- 步骤1: 获取应用构建和基础配置信息 ---
	log.Println("--- 步骤1: 获取应用构建和基础配置信息 ---")
	buildInfoURL := fmt.Sprintf("%s/app/build.json?_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取构建信息", buildInfoURL, cookie)

	cacheVersionURL := fmt.Sprintf("%s/api/v1/other/getCacheVersion?_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取缓存版本", cacheVersionURL, cookie)

	navActStateURL := fmt.Sprintf("%s/api/v1/other/GetNavActState?_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取导航活动状态", navActStateURL, cookie)

	// --- 步骤2: 模拟应用初始化API调用 ---
	log.Println("\n--- 步骤2: 模拟应用初始化API调用 ---")
	userInfoURL := fmt.Sprintf("%s/api/v1/user/info?_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取用户信息", userInfoURL, cookie)

	userNoticeURL := fmt.Sprintf("%s/api/v1/other/userNotice/switchState?_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取用户通知状态", userNoticeURL, cookie)

	announcementURL := fmt.Sprintf("%s/api/v1/other/GetAnnouncementState?_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取公告状态", announcementURL, cookie)

	trialConfigURL := fmt.Sprintf("%s/api/v3/commoncfg/getUserTrialApplyConfig?_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取试用配置", trialConfigURL, cookie)

	newestNoticeURL := fmt.Sprintf("%s/api/v3/other/getNewestNotice?mode=0&_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "获取最新通知", newestNoticeURL, cookie)

	// --- 步骤3: 注入操作日志 ---
	log.Println("\n--- 步骤3: 注入操作日志 ---")
	logURL1 := fmt.Sprintf("%s/api/v1/User/addOperateLog?operateEnum=1290000&_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "注入第一个日志", logURL1, cookie)
	logURL2 := fmt.Sprintf("%s/api/v1/User/addOperateLog?operateEnum=1290001&_=%d", baseURL, time.Now().UnixMilli())
	injectLog(client, "注入第二个日志", logURL2, cookie)

	// --- 步骤4: 调用 updateState 接口 ---
	log.Println("\n--- 步骤4: 调用 updateState 接口 ---")
	dateCode := "20250718"
	updateStateURL := fmt.Sprintf("%s/api/v3/aweme/detail/detail/updateState?awemeId=%s&dateCode=%s&_=%d", baseURL, awemeID, dateCode, time.Now().UnixMilli())
	injectLog(client, "调用updateState", updateStateURL, cookie)

	// --- 步骤5: 请求最终的 trends 接口 ---
	log.Println("\n--- 步骤5: 请求 trends 接口 ---")
	trendsURL := fmt.Sprintf("%s/api/v3/aweme/detail/detail/trends?awemeId=%s&dateCode=%s&period=30&type=1&_=%d", baseURL, awemeID, dateCode, time.Now().UnixMilli())
	log.Printf("请求 trends 接口: %s\n", trendsURL)

	reqTrend, _ := http.NewRequest("GET", trendsURL, nil)
	setHeaders(reqTrend, cookie)

	respTrend, err := client.Do(reqTrend)
	if err != nil {
		log.Fatalf("请求 trends 接口失败: %v", err)
	}
	defer respTrend.Body.Close()

	var reader io.ReadCloser
	switch respTrend.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(respTrend.Body)
		if err != nil {
			log.Fatalf("创建gzip reader失败: %v", err)
		}
		defer reader.Close()
	default:
		reader = respTrend.Body
	}

	bodyTrend, _ := ioutil.ReadAll(reader)
	log.Printf("trends 接口最终返回: 状态码 %d\n", respTrend.StatusCode)
	log.Println("trends 接口内容 (已解压):")

	if json.Valid(bodyTrend) {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyTrend, "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
		} else {
			fmt.Println(string(bodyTrend))
		}
	} else {
		fmt.Println(string(bodyTrend))
	}
}

// injectLog 和 setHeaders 辅助函数保持不变
func injectLog(client *http.Client, name, url, cookie string) {
	log.Printf("调用 [%s]: %s\n", name, url)
	req, _ := http.NewRequest("GET", url, nil)
	setHeaders(req, cookie)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求 [%s] 失败: %v\n", name, err)
	} else {
		defer resp.Body.Close()
		log.Printf("接口 [%s] 返回: 状态码 %d\n", name, resp.StatusCode)
	}
}

func setHeaders(req *http.Request, cookie string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Proxy-Connection", "keep-alive")
}
