package model

import (
	"context"
	"reflect"

	"github.com/bytedance/sonic"
	"gorm.io/gorm/schema"
)

func init() {
	schema.RegisterSerializer("sonic", SonicSerializer{})
}

// SonicSerializer 基于 bytedance/sonic 的 GORM 字段序列化器
type SonicSerializer struct{}

// Scan 实现 schema.SerializerInterface — 从数据库读取时反序列化
func (SonicSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	fieldValue := reflect.New(field.FieldType)

	if dbValue != nil {
		var bytes []byte
		switch v := dbValue.(type) {
		case []byte:
			bytes = v
		case string:
			bytes = []byte(v)
		default:
			bytes, err = sonic.Marshal(v)
			if err != nil {
				return err
			}
		}

		if len(bytes) > 0 {
			err = sonic.Unmarshal(bytes, fieldValue.Interface())
		}
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return
}

// Value 实现 schema.SerializerValuerInterface — 写入数据库时序列化
func (SonicSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	result, err := sonic.Marshal(fieldValue)
	if string(result) == "null" {
		if field.TagSettings["NOT NULL"] != "" {
			return "", nil
		}
		return nil, err
	}
	return string(result), err
}
