/**
 * @Time   : 2026/7/4 20:50
 * @Author : chenyangzhao542@gmail.com
 * @File   : tag.go
 **/

package response

import "github.com/google/uuid"

type TagResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
	DocCount   int64     `json:"doc_count"`
	ImageCount int64     `json:"image_count"`
}
