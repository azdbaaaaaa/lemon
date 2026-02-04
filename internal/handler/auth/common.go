package auth

import (
	"time"

	"lemon/internal/model/auth"
	httputil "lemon/internal/pkg/http"
)

// ErrorResponse 错误响应类型别名（使用共用的 http.ErrorResponse）
type ErrorResponse = httputil.ErrorResponse

// toUserInfo 将User实体转换为UserInfo（所有API共用）
func toUserInfo(user *auth.User) UserInfo {
	info := UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		Status:   string(user.Status),
	}

	if user.Profile != nil {
		info.Profile = &UserProfile{
			Nickname: user.Profile.Nickname,
			Avatar:   user.Profile.Avatar,
			Phone:    user.Profile.Phone,
		}
	}

	if user.LastLoginAt != nil {
		info.LastLoginAt = user.LastLoginAt.Format(time.RFC3339)
	}
	info.CreatedAt = user.CreatedAt.Format(time.RFC3339)

	return info
}
