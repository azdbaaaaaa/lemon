package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation 对话实体
type Conversation struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Title     string             `bson:"title" json:"title"`
	Model     string             `bson:"model" json:"model"`
	Messages  []Message          `bson:"messages" json:"messages"`
	Metadata  map[string]any     `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// Message 消息
type Message struct {
	Role       string      `bson:"role" json:"role"`
	Content    string      `bson:"content" json:"content"`
	Timestamp  time.Time   `bson:"timestamp" json:"timestamp"`
	TokenUsage *TokenUsage `bson:"token_usage,omitempty" json:"token_usage,omitempty"`
}
