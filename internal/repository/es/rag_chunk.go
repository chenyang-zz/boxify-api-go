package es

import (
	"context"
	"fmt"
	"strings"

	ragchunker "github.com/boxify/api-go/internal/core/rag/chunker"
	"github.com/boxify/api-go/internal/core/valuex"
	infraes "github.com/boxify/api-go/internal/infrastructure/db/es"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const DefaultChunkIndex = "boxify_chunks"

// SourceTypeImage 表示 ES chunk 来源为图片。
const SourceTypeImage = "image"

type RAGChunkRepository struct {
	client *infraes.Client
	index  string
}

func NewRAGChunkRepository(client *infraes.Client, index string) repository.RAGChunkRepository {
	if client == nil {
		panic("elasticsearch client is required")
	}
	index = strings.TrimSpace(index)
	if index == "" {
		index = DefaultChunkIndex
	}
	return &RAGChunkRepository{client: client, index: index}
}

// EnsureIndex 确保索引存在
func (r *RAGChunkRepository) EnsureIndex(ctx context.Context, embeddingDim int) error {
	if embeddingDim <= 0 {
		embeddingDim = 1024
	}
	exists, err := r.client.IndexExists(ctx, r.index)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = r.client.CreateIndex(ctx, r.index, chunkIndexMapping(embeddingDim))
	return err
}

// IndexDocumentChunks 索引文档 chunk
func (r *RAGChunkRepository) IndexDocumentChunks(ctx context.Context, doc *models.Document, chunks []*ragchunker.Chunk, vectors [][]float64) error {
	records, err := r.buildDocumentChunkRecords(doc, chunks, vectors)
	if err != nil {
		return err
	}
	for _, record := range records {
		if _, err := r.client.Index(ctx, r.index, record.ChunkID, record); err != nil {
			return err
		}
	}
	return nil
}

// IndexImageChunk 索引图片描述 chunk。
func (r *RAGChunkRepository) IndexImageChunk(ctx context.Context, image *models.Image, content string, vector []float64) error {
	if image == nil {
		return xerr.Internal("图片为空", nil)
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return xerr.Internal("图片描述内容为空", nil)
	}
	if len(vector) == 0 {
		return xerr.Internal("图片向量为空", nil)
	}
	kbID := ""
	if image.KBID != nil {
		kbID = image.KBID.String()
	}
	chunkID := deterministicImageChunkID(image.ID).String()
	record := models.RAGChunkRecord{
		ChunkID:    chunkID,
		SourceID:   image.ID.String(),
		UserID:     image.UserID.String(),
		KBID:       kbID,
		Name:       image.FileName,
		SourceType: SourceTypeImage,
		Content:    content,
		Level:      "parent",
		Tags:       tagNames(image.Tags),
		Vector:     vector,
	}
	if _, err := r.client.Index(ctx, r.index, chunkID, record); err != nil {
		return err
	}
	return nil
}

// DeleteBySource 按来源实体 ID 删除 chunk。
func (r *RAGChunkRepository) DeleteBySource(ctx context.Context, userID uuid.UUID, sourceID uuid.UUID) error {
	_, err := r.client.DeleteByQuery(ctx, r.index, map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"user_id": userID.String()}},
					map[string]any{"term": map[string]any{"source_id": sourceID.String()}},
				},
			},
		},
	})
	return err
}

// UpdateKnowledgeBase 更新来源 chunk 的知识库归属。
func (r *RAGChunkRepository) UpdateKnowledgeBase(ctx context.Context, userID uuid.UUID, sourceID uuid.UUID, kbID uuid.UUID) error {
	_, err := r.client.UpdateByQuery(ctx, r.index, map[string]any{
		"script": map[string]any{
			"source": "ctx._source.kb_id = params.kb_id",
			"params": map[string]any{
				"kb_id": kbID.String(),
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"user_id": userID.String()}},
					map[string]any{"term": map[string]any{"source_id": sourceID.String()}},
				},
			},
		},
	})
	return err
}

// UpdateTags 更新来源 chunk 的标签。
func (r *RAGChunkRepository) UpdateTags(ctx context.Context, userID uuid.UUID, sourceID uuid.UUID, tags []string) error {
	_, err := r.client.UpdateByQuery(ctx, r.index, map[string]any{
		"script": map[string]any{
			"source": "ctx._source.tags = params.tags",
			"params": map[string]any{
				"tags": tags,
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"user_id": userID.String()}},
					map[string]any{"term": map[string]any{"source_id": sourceID.String()}},
				},
			},
		},
	})
	return err
}

// DecodeSource 解码源数据
func (r *RAGChunkRepository) DecodeSource(src map[string]any) (models.RAGChunkSource, error) {
	chunkID, err := uuid.Parse(valuex.String(src["chunk_id"]))
	if err != nil {
		return models.RAGChunkSource{}, fmt.Errorf("invalid chunk_id: %w", err)
	}
	sourceID, err := uuid.Parse(valuex.String(src["source_id"]))
	if err != nil {
		return models.RAGChunkSource{}, fmt.Errorf("invalid source_id: %w", err)
	}
	var kbID *uuid.UUID
	if rawKBID := valuex.String(src["kb_id"]); rawKBID != "" {
		parsed, err := uuid.Parse(rawKBID)
		if err != nil {
			return models.RAGChunkSource{}, fmt.Errorf("invalid kb_id: %w", err)
		}
		kbID = &parsed
	}
	return models.RAGChunkSource{
		ChunkID:    chunkID,
		SourceID:   sourceID,
		KBID:       kbID,
		Name:       valuex.String(src["name"]),
		SourceType: valuex.String(src["source_type"]),
	}, nil
}

// buildDocumentChunkRecords 构建文档 chunk 写入记录。
func (r *RAGChunkRepository) buildDocumentChunkRecords(doc *models.Document, chunks []*ragchunker.Chunk, vectors [][]float64) ([]models.RAGChunkRecord, error) {
	if doc == nil {
		return nil, xerr.Internal("文档为空", nil)
	}
	texts := chunkTexts(chunks)
	if len(texts) != len(vectors) {
		return nil, xerr.Internal("文档 chunk 向量数量不匹配", nil)
	}
	tags := tagNames(doc.Tags)
	kbID := ""
	if doc.KBID != nil {
		kbID = doc.KBID.String()
	}
	out := make([]models.RAGChunkRecord, 0, len(texts))
	vectorIndex := 0
	for parentIndex, parent := range chunks {
		if parent == nil {
			continue
		}
		parentContent := strings.TrimSpace(parent.Content)
		parentID := deterministicChunkID(doc.ID, parentIndex, -1).String()
		if parentContent != "" {
			out = append(out, models.RAGChunkRecord{
				ChunkID:    parentID,
				SourceID:   doc.ID.String(),
				UserID:     doc.UserID.String(),
				KBID:       kbID,
				Name:       doc.FileName,
				SourceType: doc.SourceType,
				Content:    parentContent,
				Level:      "parent",
				Tags:       tags,
				Vector:     vectors[vectorIndex],
			})
			vectorIndex++
		}
		for childIndex, child := range parent.Children {
			childContent := strings.TrimSpace(child)
			if childContent == "" {
				continue
			}
			childID := deterministicChunkID(doc.ID, parentIndex, childIndex).String()
			out = append(out, models.RAGChunkRecord{
				ChunkID:    childID,
				ParentID:   parentID,
				SourceID:   doc.ID.String(),
				UserID:     doc.UserID.String(),
				KBID:       kbID,
				Name:       doc.FileName,
				SourceType: doc.SourceType,
				Content:    childContent,
				Level:      "child",
				Tags:       tags,
				Vector:     vectors[vectorIndex],
			})
			vectorIndex++
		}
	}
	return out, nil
}

func chunkTexts(chunks []*ragchunker.Chunk) []string {
	texts := make([]string, 0)
	for _, parent := range chunks {
		if parent == nil {
			continue
		}
		if content := strings.TrimSpace(parent.Content); content != "" {
			texts = append(texts, content)
		}
		for _, child := range parent.Children {
			if content := strings.TrimSpace(child); content != "" {
				texts = append(texts, content)
			}
		}
	}
	return texts
}

func chunkIndexMapping(embeddingDim int) map[string]any {
	return map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"chunk_id":    map[string]any{"type": "keyword"},
				"parent_id":   map[string]any{"type": "keyword"},
				"source_id":   map[string]any{"type": "keyword"},
				"user_id":     map[string]any{"type": "keyword"},
				"kb_id":       map[string]any{"type": "keyword"},
				"name":        map[string]any{"type": "keyword"},
				"source_type": map[string]any{"type": "keyword"},
				"level":       map[string]any{"type": "keyword"},
				"tags":        map[string]any{"type": "keyword"},
				"content":     map[string]any{"type": "text"},
				"vector": map[string]any{
					"type":       "dense_vector",
					"dims":       embeddingDim,
					"index":      true,
					"similarity": "cosine",
				},
			},
		},
	}
}

func tagNames(rows []models.Tag) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if name := strings.TrimSpace(row.Name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// deterministicChunkID 确定性 chunk ID
func deterministicChunkID(documentID uuid.UUID, parentIndex int, childIndex int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("boxify:document:%s:%d:%d", documentID.String(), parentIndex, childIndex)))
}

// deterministicImageChunkID 生成图片描述 chunk 的确定性 ID。
func deterministicImageChunkID(imageID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("boxify:image:%s:0", imageID.String())))
}
