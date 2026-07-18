/*
 * @Time   : 2026-07-12 20:22:22
 * @Author : chenyang
 * @File   : image.go
 */

package request

import "mime/multipart"

type SearchImageRequest struct {
	Query string `json:"query" binding:"required,min=1"`
	TopK  int64  `json:"top_k" binding:"omitempty,gte=1,lte=50"` // default 12
}

type UploadImageRequest struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
	KBID *string               `form:"kb_id" binding:"omitempty,uuid"`
}

type UriImageIDRequest struct {
	ImageID string `uri:"image_id" binding:"required,uuid"`
}

type ListImagesRequest struct {
	PageRequest
	Tag  *string `json:"tag" form:"tag" binding:"omitempty"`          // 按标签名筛选
	KBID *string `json:"kb_id" form:"kb_id" binding:"omitempty,uuid"` // 按知识库筛选
}

type MoveImageRequest struct {
	UriImageIDRequest
	KBID string `json:"kb_id" form:"kb_id" binding:"required,uuid"`
}
