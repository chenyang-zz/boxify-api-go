/**
 * @Time   : 2026/6/27 15:41
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package request

type CreateConversationRequest struct {
	Title *string `json:"title" binding:"omitempty,min=1,max=256"`
}
