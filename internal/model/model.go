// Package model 定义应用的核心数据结构。
package model

import "time"

// Deck 表示一个单词本。
type Deck struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// Word 表示一个单词，通过 DeckID 关联到所属单词本。
type Word struct {
	ID         int64
	DeckID     int64
	Term       string // 单词
	Phonetic   string // 音标
	Definition string // 完整释义
	Audio      string // 本地语音文件路径（可为空）
	// 学习状态
	Mastered    bool      // 是否已掌握
	ReviewCount int       // 复习次数
	LastReview  time.Time // 上次复习时间
}

// DeckStats 表示一个单词本的学习统计。
type DeckStats struct {
	Total    int // 单词总数
	Mastered int // 已掌握数
}
