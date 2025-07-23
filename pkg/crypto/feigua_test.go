package crypto

import (
	"os"
	"testing"
)

func TestFeiguaDecrypt(t *testing.T) {
	rnd := "638888766653375257"

	// 1. 读取 data.txt
	ciphertext, err := os.ReadFile("data.txt")
	if err != nil {
		t.Fatalf("读取 data.txt 失败: %v", err)
	}

	// 2. 解密
	decrypted, err := FeiguaDecrypt(string(ciphertext), rnd)
	if err != nil {
		t.Fatalf("FeiguaDecrypt failed: %v", err)
	}

	// 3. 写入 result.json
	if err := os.WriteFile("resu13131t12.json", []byte(decrypted), 0644); err != nil {
		t.Fatalf("写入 result.json 失败: %v", err)
	}

	t.Logf("解密成功，已写入 result.json")
}
