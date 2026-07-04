/**
 * @Time   : 2026/7/4 20:48
 * @Author : chenyangzhao542@gmail.com
 * @File   : tag.go
 **/

package request

type UriTagServerIDRequest struct {
	ID string `uri:"tag_id" binding:"required,uuid"`
}

type ListTagsRequest struct {
	PageRequest
	Scope *string `form:"scope" json:"scope" binding:"omitempty,oneof=all document image"`
}

type TagMergeRequest struct {
	SourceID string `json:"source_id" binding:"required,uuid"`
	TargetID string `json:"target_id" binding:"required,uuid"`
}

type TagUpdateRequest struct {
	UriTagServerIDRequest
	Name  *string `json:"name" binding:"omitempty,min=1,max=64"`
	Color *string `json:"color" binding:"omitempty,min=1,max=16"`
}
