package id

import (
	"github.com/google/uuid"
)

// New 生成新的UUID（string格式）
func New() string {
	return uuid.New().String()
}

// IsValid 验证UUID格式是否有效
func IsValid(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
