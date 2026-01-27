package auth

import (
	"time"
)

// RefreshToken 刷新Token实体
// 参考: docs/design/auth/AUTH_DESIGN.md - 2.2 RefreshToken实体
// ID和UserID使用UUID格式（string），避免ObjectID转换的麻烦
type RefreshToken struct {
	ID        string    `bson:"_id,omitempty" json:"id"`      // UUID格式的ID
	UserID    string    `bson:"user_id" json:"user_id"`       // UUID格式的用户ID
	Token     string    `bson:"token" json:"token"`           // Refresh Token值
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"` // 过期时间
	CreatedAt time.Time `bson:"created_at" json:"created_at"` // 创建时间
}

// IsExpired 检查Token是否已过期
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}
