package middlewares

import (
	"testing"

	"github.com/CuratorC/gocanned/logger"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

func init() {
	logger.Logger = zap.NewNop()
}

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
