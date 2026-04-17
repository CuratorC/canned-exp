package model

import (
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	validate *validator.Validate
)

// GetValidator 返回全局单例的 validator 引擎（已注册自定义规则）
func GetValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New()
		_ = validate.RegisterValidation("trimmedRequired", trimmedRequired)
		_ = validate.RegisterValidation("trimmedMax", trimmedMax)
		_ = validate.RegisterValidation("personalityType", personalityType)
	})
	return validate
}

// trimmedRequired 校验 trim 后不能为空
func trimmedRequired(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}

// trimmedMax 校验 trim 后的长度不超过指定值
// 用法: trimmedMax=8000
func trimmedMax(fl validator.FieldLevel) bool {
	maxLen := fl.Param()
	if maxLen == "" {
		return true
	}
	var limit int
	for _, c := range maxLen {
		if c < '0' || c > '9' {
			return true
		}
		limit = limit*10 + int(c-'0')
	}
	return len(strings.TrimSpace(fl.Field().String())) <= limit
}

// personalityType 校验 type 字段：空值或 string/text/number/boolean
func personalityType(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true
	}
	return val == "string" || val == "text" || val == "number" || val == "boolean"
}
