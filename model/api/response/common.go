// Package response provides API response DTOs.
package response

// PagedResp 分页响应通用结构
type PagedResp[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// NewPagedResp 创建分页响应
func NewPagedResp[T any](items []T, total int64, page, pageSize int) PagedResp[T] {
	return PagedResp[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}
