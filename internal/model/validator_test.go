package model

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestValidator_TrimmedRequired(t *testing.T) {
	v := GetValidator()

	t.Run("trimmedRequired拒绝空字符串", func(t *testing.T) {
		type stub struct {
			Field string `validate:"trimmedRequired"`
		}
		err := v.Struct(stub{Field: ""})
		if err == nil {
			t.Fatal("expected error for empty string, got nil")
		}
	})

	t.Run("trimmedRequired拒绝纯空格", func(t *testing.T) {
		type stub struct {
			Field string `validate:"trimmedRequired"`
		}
		err := v.Struct(stub{Field: "   \t\n"})
		if err == nil {
			t.Fatal("expected error for whitespace-only string, got nil")
		}
	})

	t.Run("trimmedRequired接受正常内容", func(t *testing.T) {
		type stub struct {
			Field string `validate:"trimmedRequired"`
		}
		err := v.Struct(stub{Field: "hello"})
		if err != nil {
			t.Fatalf("expected pass, got error: %v", err)
		}
	})
}

func TestValidator_TrimmedMax(t *testing.T) {
	v := GetValidator()

	t.Run("trimmedMax拒绝超长内容", func(t *testing.T) {
		type stub struct {
			Field string `validate:"trimmedMax=10"`
		}
		err := v.Struct(stub{Field: strings.Repeat("a", 11)})
		if err == nil {
			t.Fatal("expected error for exceeding max length, got nil")
		}
	})

	t.Run("trimmedMax接受边界值", func(t *testing.T) {
		type stub struct {
			Field string `validate:"trimmedMax=10"`
		}
		err := v.Struct(stub{Field: strings.Repeat("a", 10)})
		if err != nil {
			t.Fatalf("expected pass for exactly max length, got error: %v", err)
		}
	})

	t.Run("trimmedMax先trim再判断长度", func(t *testing.T) {
		type stub struct {
			Field string `validate:"trimmedMax=10"`
		}
		// 7 个字符 + 3 个尾部空格，trim 后长度 7，在限制内
		err := v.Struct(stub{Field: "abcdefg   "})
		if err != nil {
			t.Fatalf("expected pass after trim (len 7 <= 10), got error: %v", err)
		}
	})
}

// 确保返回的是同一个实例（单例）
func TestGetValidator_Singleton(t *testing.T) {
	v1 := GetValidator()
	v2 := GetValidator()
	if v1 != v2 {
		t.Error("GetValidator should return the same instance")
	}
}

// 确保 validator 实现了 *validator.Validate 接口
var _ *validator.Validate = GetValidator()
