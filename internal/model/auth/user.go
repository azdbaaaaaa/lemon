package auth

import (
	"time"
)

// User 用户实体
// 参考: docs/design/auth/AUTH_DESIGN.md - 2.1 User实体
// ID使用UUID格式（string），避免ObjectID转换的麻烦
type User struct {
	ID          string       `bson:"_id,omitempty" json:"id"`                    // UUID格式的ID
	Username    string       `bson:"username" json:"username"`                   // 用户名（唯一）
	Email       string       `bson:"email" json:"email"`                         // 邮箱（唯一）
	Password    string       `bson:"password" json:"-"`                          // 密码（加密存储，不返回）
	Role        UserRole     `bson:"role" json:"role"`                           // 角色
	Status      UserStatus   `bson:"status" json:"status"`                       // 状态
	Profile     *UserProfile `bson:"profile,omitempty" json:"profile,omitempty"` // 用户资料
	LastLoginAt *time.Time   `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
	CreatedAt   time.Time    `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time    `bson:"updated_at" json:"updated_at"`
}

// UserProfile 用户资料
// 参考: docs/design/auth/AUTH_DESIGN.md - 2.1 User实体
type UserProfile struct {
	Nickname string `bson:"nickname,omitempty" json:"nickname,omitempty"`
	Avatar   string `bson:"avatar,omitempty" json:"avatar,omitempty"`
	Phone    string `bson:"phone,omitempty" json:"phone,omitempty"`
}

// UserRole 用户角色
// 参考: docs/design/auth/AUTH_DESIGN.md - 1.2 角色定义
type UserRole string

const (
	RoleAdmin    UserRole = "admin"    // 超级管理员
	RoleEditor   UserRole = "editor"   // 编辑人员
	RoleReviewer UserRole = "reviewer" // 审核人员
)

// IsValid 检查角色是否有效
func (r UserRole) IsValid() bool {
	return r == RoleAdmin || r == RoleEditor || r == RoleReviewer
}

// String 返回角色字符串
func (r UserRole) String() string {
	return string(r)
}

// UserStatus 用户状态
// 参考: docs/design/auth/AUTH_DESIGN.md - 2.1 User实体
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"   // 激活
	UserStatusInactive UserStatus = "inactive" // 未激活（注册待审核）
	UserStatusBanned   UserStatus = "banned"   // 禁用
)

// IsValid 检查状态是否有效
func (s UserStatus) IsValid() bool {
	return s == UserStatusActive || s == UserStatusInactive || s == UserStatusBanned
}

// String 返回状态字符串
func (s UserStatus) String() string {
	return string(s)
}
