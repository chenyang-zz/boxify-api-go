package repository

import (
	"context"

	ragchunker "github.com/boxify/api-go/internal/core/rag/chunker"
	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

type RAGChunkRepository interface {
	EnsureIndex(ctx context.Context, embeddingDim int) error
	IndexDocumentChunks(ctx context.Context, document *models.Document, chunks []*ragchunker.Chunk, vectors [][]float64) error
	IndexImageChunk(ctx context.Context, image *models.Image, content string, vector []float64) error
	DeleteBySource(ctx context.Context, userID uuid.UUID, sourceID uuid.UUID) error
	UpdateKnowledgeBase(ctx context.Context, userID uuid.UUID, sourceID uuid.UUID, kbID uuid.UUID) error
	UpdateTags(ctx context.Context, userID uuid.UUID, sourceID uuid.UUID, tags []string) error
	DecodeSource(src map[string]any) (models.RAGChunkSource, error)
}
