// Package models 定义 API 请求/响应的数据结构。
// JSON 字段名使用 camelCase，与前端及原 MongoDB 导出字段保持一致。
package models

import "time"

// PaginatedResult 分页列表的统一响应格式。
type PaginatedResult[T any] struct {
	Data  []T `json:"data"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
} 

// Diary 旅行日记 / 日常日记（共用结构，存不同表）。
type Diary struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content,omitempty"`
	Images    []string  `json:"images"` // 对外只暴露 images，不暴露旧版 image 字段
	Date      string    `json:"date"`   // YYYY-MM-DD
	Likes     int       `json:"likes"`
	Comments  int       `json:"comments"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// Whisper 悄悄话。
type Whisper struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	Date      string    `json:"date"`
	Likes     int       `json:"likes"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// SpecialDate 纪念日 / 生日等特殊日期。
type SpecialDate struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Date          string    `json:"date"`
	Description   string    `json:"description,omitempty"`
	IsAnniversary bool      `json:"isAnniversary"`
	Type          string    `json:"type,omitempty"` // anniversary | birthday 等
	CreatedAt     time.Time `json:"createdAt,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt,omitempty"`
}

// TreeLog 浇水记录条目。
type TreeLog struct {
	Date string `json:"date"`
}

// Tree 爱情树状态（months 由 handler 按 startDate 动态计算，不入库）。
type Tree struct {
	ID          int       `json:"id"`
	StartDate   string    `json:"startDate"`
	LastWatered string    `json:"lastWatered"`
	WaterCount  int       `json:"waterCount"`
	Logs        []TreeLog `json:"logs,omitempty"`
	Months      int       `json:"months,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// User 登录用户（password 不序列化到 JSON）。
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
	Avatar   string `json:"avatar,omitempty"`
}

// Comment 日记评论（含评论人展示信息）。
type Comment struct {
	ID        int       `json:"id"`
	DiaryID   int       `json:"diaryId"`
	Content   string    `json:"content"`
	Username  string    `json:"username"`
	Name      string    `json:"name"` // 展示名称，与 username 一致
	Avatar    string    `json:"avatar,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// CommentInput 发表评论请求体。
type CommentInput struct {
	Content string `json:"content"`
}

// DiaryInput 创建/更新日记时的请求体。
type DiaryInput struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Date     string   `json:"date"`
	Image    string   `json:"image"`  // 兼容旧版单图 base64
	Images   []string `json:"images"` // 新版多图 base64 数组
	Likes    int      `json:"likes"`
	Comments int      `json:"comments"`
}

// WhisperInput 创建/更新悄悄话时的请求体。
type WhisperInput struct {
	Content string `json:"content"`
	Date    string `json:"date"`
	Likes   int    `json:"likes"`
}

// SpecialDateInput 创建/更新特殊日期时的请求体。
type SpecialDateInput struct {
	Title         string `json:"title"`
	Date          string `json:"date"`
	Description   string `json:"description"`
	IsAnniversary bool   `json:"isAnniversary"`
	Type          string `json:"type"`
}
