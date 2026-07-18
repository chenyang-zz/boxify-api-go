package models

import "github.com/google/uuid"

// RAGChunkSource 表示从 ES chunk _source 解码出的业务元数据。
type RAGChunkSource struct {
	ChunkID    uuid.UUID
	SourceID   uuid.UUID
	KBID       *uuid.UUID
	Name       string
	SourceType string
}

// RAGChunkRecord 表示写入 ES chunk 索引的记录结构。
// source_id + source_type 标识来源实体（文档或图片等），不绑定单一业务类型。
type RAGChunkRecord struct {
	ChunkID    string    `json:"chunk_id"`
	ParentID   string    `json:"parent_id,omitempty"`
	SourceID   string    `json:"source_id"`
	UserID     string    `json:"user_id"`
	KBID       string    `json:"kb_id,omitempty"`
	Name       string    `json:"name"`
	SourceType string    `json:"source_type"`
	Content    string    `json:"content"`
	Level      string    `json:"level"`
	Tags       []string  `json:"tags"`
	Vector     []float64 `json:"vector"`
}
