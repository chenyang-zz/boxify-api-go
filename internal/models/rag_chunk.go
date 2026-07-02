package models

import "github.com/google/uuid"

// RAGChunkSource 表示从 ES chunk _source 解码出的业务元数据。
type RAGChunkSource struct {
	ChunkID    uuid.UUID
	DocumentID uuid.UUID
	KBID       *uuid.UUID
	DocName    string
	SourceType string
}

// RAGChunkDocument 表示写入 ES chunk 索引的文档结构。
type RAGChunkDocument struct {
	ChunkID    string    `json:"chunk_id"`
	ParentID   string    `json:"parent_id,omitempty"`
	DocumentID string    `json:"document_id"`
	UserID     string    `json:"user_id"`
	KBID       string    `json:"kb_id,omitempty"`
	DocName    string    `json:"doc_name"`
	SourceType string    `json:"source_type"`
	Content    string    `json:"content"`
	Level      string    `json:"level"`
	Tags       []string  `json:"tags"`
	Vector     []float64 `json:"vector"`
}
