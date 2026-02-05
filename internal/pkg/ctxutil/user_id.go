package ctxutil

import "context"

// userIDKeyType 使用私有类型避免与其他 context key 冲突
type userIDKeyType struct{}

var userIDKey = userIDKeyType{}

// WithUserID 将 userID 注入到 context 中
// 说明：建议在认证中间件中调用，例如在解析 JWT 成功后：
//   ctx := ctxutil.WithUserID(c.Request.Context(), claims.UserID)
//   c.Request = c.Request.WithContext(ctx)
func WithUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID 从 context 中解析 userID
// 返回值：
//   - string: 解析到的 userID
//   - bool  : 是否存在有效的 userID
func GetUserID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v := ctx.Value(userIDKey)
	id, ok := v.(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}


