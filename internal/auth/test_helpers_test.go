package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// generateTestSecret 生成一个可用于测试的 TOTP 密钥
func generateTestSecret(t *testing.T) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "canned-exp-test",
		AccountName: "test@test.com",
	})
	if err != nil {
		t.Fatalf("generate totp key: %v", err)
	}
	return key.Secret()
}

// generateTestCode 根据 secret 生成当前有效的 TOTP 码
func generateTestCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	return code
}
