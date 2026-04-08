// model/api/request/pool.go
package request

// GetPoolReq 获取池数据请求
type GetPoolReq struct {
	Date string `form:"date"`
}

// ReclassifyReq 重分类请求
type ReclassifyReq struct {
	Date string `form:"date"`
}
