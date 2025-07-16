package crypto

import (
	"bytes"
	"compress/gzip"
	"crypto/cipher"
	"crypto/des"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
)

// pkcs7Unpad 移除 PKCS7 填充
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("pkcs7: data is empty")
	}
	if len(data)%blockSize != 0 {
		return nil, errors.New("pkcs7: data is not block-aligned")
	}
	paddingLen := int(data[len(data)-1])
	if paddingLen > blockSize || paddingLen == 0 {
		return nil, errors.New("pkcs7: invalid padding")
	}
	pad := data[len(data)-paddingLen:]
	for i := 0; i < paddingLen; i++ {
		if pad[i] != byte(paddingLen) {
			return nil, errors.New("pkcs7: invalid padding")
		}
	}
	return data[:len(data)-paddingLen], nil
}

// FeiguaDecrypt 严格按照飞瓜前端JS逻辑解密API响应
// Rnd: 任意长度，前8字节为key，最后8字节为iv
func FeiguaDecrypt(encryptedDataB64, rnd string) (string, error) {
	if len(rnd) < 8 {
		return "", errors.New("rnd must be at least 8 bytes for key")
	}
	key := []byte(rnd[:8])
	iv := []byte(rnd[len(rnd)-8:])

	// 1. Base64解码加密数据
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedDataB64)
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode encrypted data: %w", err)
	}

	// 2. DES CBC 解密
	block, err := des.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create new des cipher: %w", err)
	}

	if len(encryptedData)%block.BlockSize() != 0 {
		return "", errors.New("encrypted data is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decryptedPadded := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedPadded, encryptedData)

	// 3. 移除 PKCS7 填充
	decrypted, err := pkcs7Unpad(decryptedPadded, block.BlockSize())
	if err != nil {
		return "", fmt.Errorf("failed to unpad pkcs7: %w", err)
	}

	// 4. 解密结果是base64字符串，再次解码
	binaryData, err := base64.StdEncoding.DecodeString(string(decrypted))
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode the decrypted content: %w", err)
	}

	// 5. Gzip解压
	gzipReader, err := gzip.NewReader(bytes.NewReader(binaryData))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	decompressed, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		return "", fmt.Errorf("failed to decompress gzip data: %w", err)
	}

	// 6. 返回解析后的JSON字符串
	return string(decompressed), nil
}
