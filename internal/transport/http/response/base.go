/**
 * @Time   : 2026/6/27 16:20
 * @Author : chenyangzhao542@gmail.com
 * @File   : base.go.go
 **/

package response

type ListResponse[T any] struct {
	List []T `json:"list"`
}
