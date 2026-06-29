package mapper_test

import (
	"testing"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

func TestMCPServerToResponseMapsToolsCacheWithoutNilPrefix(t *testing.T) {
	row := &models.MCPServer{
		ID:   uuid.New(),
		Name: "demo",
		ToolsCache: models.MCPMetas{
			{Name: "search", Description: "web search"},
		},
	}

	got := mapper.MCPServerToResponse(row, "masked-secret")

	if len(got.ToolsCache) != 1 {
		t.Fatalf("ToolsCache len = %d, want 1; value=%#v", len(got.ToolsCache), got.ToolsCache)
	}
	if got.ToolsCache[0] == nil {
		t.Fatal("ToolsCache[0] = nil, want meta")
	}
	if got.ToolsCache[0].Name != "search" || got.ToolsCache[0].Description != "web search" {
		t.Fatalf("ToolsCache[0] = %#v", got.ToolsCache[0])
	}
	if got.AuthMasked != "masked-secret" {
		t.Fatalf("AuthMasked = %q, want masked-secret", got.AuthMasked)
	}
}
