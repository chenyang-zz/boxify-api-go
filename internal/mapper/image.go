package mapper

import (
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/response"
)

func ImageToResponse(row *models.Image, tags []string, url string) *response.ImageResponse {
	if row == nil {
		return nil
	}
	if tags == nil {
		tags = imageTagNames(row.Tags)
	}
	return &response.ImageResponse{
		ID:          row.ID,
		KBID:        row.KBID,
		FileName:    row.FileName,
		FileExt:     row.FileExt,
		FileSize:    row.FileSize,
		Url:         url,
		Description: derefString(row.Description),
		Objects:     imageObjectsToMap(row.Objects),
		Scene:       row.Scene,
		Tags:        tags,
		Status:      row.Status,
		ErrorMsg:    row.ErrorMsg,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func imageTagNames(rows []models.Tag) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Name)
	}
	return out
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// imageObjectsToMap 将 ORM 中的物体列表稳定映射为响应 map。
func imageObjectsToMap(objects models.JSONMaps) map[string]any {
	if objects == nil {
		return map[string]any{"items": []map[string]any{}}
	}
	items := make([]map[string]any, 0, len(objects))
	for _, item := range objects {
		if item == nil {
			continue
		}
		items = append(items, item)
	}
	return map[string]any{"items": items}
}
