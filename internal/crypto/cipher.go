package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Cipher 使用 AES-256-GCM 对持久化敏感字段进行加解密。
type Cipher struct {
	key []byte
}

// NewCipher 使用 32 字节主密钥构建 Cipher。
func NewCipher(masterKey []byte) (*Cipher, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("expected 32-byte master key, got %d", len(masterKey))
	}
	cp := make([]byte, 32)
	copy(cp, masterKey)
	return &Cipher{key: cp}, nil
}

// DecodeMasterKey 解析 64 位十六进制或 Base64 编码的 32 字节主密钥。
func DecodeMasterKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) == 64 {
		if b, err := hex.DecodeString(trimmed); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	if b, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, errors.New("master key must be 32 bytes (64 hex chars or base64)")
}

// GenerateMasterKeyHex 生成随机 32 字节主密钥并返回 64 位十六进制字符串。
func GenerateMasterKeyHex() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generate master key: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// EncryptString 加密明文字符串并返回 Base64 密文；空字符串直接返回空。
func (c *Cipher) EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	out := make([]byte, 0, len(nonce)+len(sealed))
	out = append(out, nonce...)
	out = append(out, sealed...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// DecryptString 解密 EncryptString 产生的 Base64 密文；空字符串直接返回空。
func (c *Cipher) DecryptString(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode base64 ciphertext: %w", err)
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
