package mcp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"GopherAI/config"
)

var ErrFeatureDisabled = errors.New("mcp feature disabled")

func FeatureEnabled() bool {
	return strings.TrimSpace(config.GetConfig().MCPConfig.SecretKey) != ""
}

func secretKey() ([]byte, error) {
	rawKey := strings.TrimSpace(config.GetConfig().MCPConfig.SecretKey)
	if rawKey == "" {
		return nil, ErrFeatureDisabled
	}

	sum := sha256.Sum256([]byte(rawKey))
	return sum[:], nil
}

func EncryptHeaders(headers map[string]string) (string, error) {
	if len(headers) == 0 {
		return "", nil
	}

	key, err := secretKey()
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(headers)
	if err != nil {
		return "", fmt.Errorf("marshal mcp headers: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create mcp cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create mcp gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate mcp nonce: %w", err)
	}

	encrypted := gcm.Seal(nonce, nonce, payload, nil)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func DecryptHeaders(ciphertext string) (map[string]string, error) {
	if strings.TrimSpace(ciphertext) == "" {
		return map[string]string{}, nil
	}

	key, err := secretKey()
	if err != nil {
		return nil, err
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode mcp headers: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create mcp cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create mcp gcm: %w", err)
	}

	if len(raw) < gcm.NonceSize() {
		return nil, fmt.Errorf("invalid mcp ciphertext")
	}

	nonce := raw[:gcm.NonceSize()]
	data := raw[gcm.NonceSize():]
	payload, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt mcp headers: %w", err)
	}

	headers := make(map[string]string)
	if err := json.Unmarshal(payload, &headers); err != nil {
		return nil, fmt.Errorf("unmarshal mcp headers: %w", err)
	}

	return headers, nil
}

func SortedHeaderKeys(headers map[string]string) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func MaskSecret(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= 4 {
		return "****"
	}

	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}
