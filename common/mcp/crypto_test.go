package mcp

import (
	"reflect"
	"testing"

	"GopherAI/config"
)

func TestEncryptHeadersRoundTrip(t *testing.T) {
	conf := config.GetConfig()
	previousSecret := conf.MCPConfig.SecretKey
	conf.MCPConfig.SecretKey = "unit-test-secret"
	defer func() {
		conf.MCPConfig.SecretKey = previousSecret
	}()

	headers := map[string]string{
		"Authorization": "Bearer secret-token",
		"X-Trace-ID":    "trace-123",
	}

	ciphertext, err := EncryptHeaders(headers)
	if err != nil {
		t.Fatalf("EncryptHeaders returned error: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("expected ciphertext to be generated")
	}

	decoded, err := DecryptHeaders(ciphertext)
	if err != nil {
		t.Fatalf("DecryptHeaders returned error: %v", err)
	}
	if !reflect.DeepEqual(decoded, headers) {
		t.Fatalf("unexpected decrypted headers: %#v", decoded)
	}
}

func TestMaskSecretReturnsMaskedValue(t *testing.T) {
	masked := MaskSecret("Bearer secret-token")
	if masked == "" {
		t.Fatal("expected masked secret")
	}
	if masked == "Bearer secret-token" {
		t.Fatal("expected secret to be masked")
	}
}
