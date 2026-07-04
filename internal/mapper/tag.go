package mapper

import (
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

func TagToResponse(row *models.Tag, docCount int64, imageCount int64) *response.TagResponse {
	if row == nil {
		return nil
	}
	return &response.TagResponse{
		ID:         row.ID,
		Name:       row.Name,
		Color:      row.Color,
		DocCount:   docCount,
		ImageCount: imageCount,
	}
}

func TagsToListResponse(rows []*models.Tag, docCounts map[uuid.UUID]int64, imageCounts map[uuid.UUID]int64) *response.ListResponse[*response.TagResponse] {
	out := make([]*response.TagResponse, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, TagToResponse(row, docCounts[row.ID], imageCounts[row.ID]))
	}
	return &response.ListResponse[*response.TagResponse]{List: out}
}

func TagsToPageListResponse(rows []*models.Tag, total int64, page int64, pageSize int64, docCounts map[uuid.UUID]int64, imageCounts map[uuid.UUID]int64) *response.PageListResponse[*response.TagResponse] {
	out := TagsToListResponse(rows, docCounts, imageCounts)
	return &response.PageListResponse[*response.TagResponse]{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     out.List,
	}
}
