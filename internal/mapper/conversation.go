/**
 * @Time   : 2026/6/27 15:56
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package mapper

import (
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/response"
)

func ConversationToResponse(row *models.Conversation) *response.ConversationResponse {
	if row == nil {
		return nil
	}
	res := &response.ConversationResponse{
		ID:              row.ID,
		Title:           row.Title,
		IsGroup:         row.IsGroup,
		MemberPersonIDs: row.MemberPersonIDs,
		EnableTools:     row.EnableTools,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	if res.MemberPersonIDs == nil {
		res.MemberPersonIDs = []string{}
	}

	return res
}

func ConversationsToListResponse(rows []*models.Conversation) *response.ListResponse[*response.ConversationResponse] {
	out := make([]*response.ConversationResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, ConversationToResponse(row))
	}
	return &response.ListResponse[*response.ConversationResponse]{List: out}
}
