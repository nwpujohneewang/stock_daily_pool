// model/api/request/stock.go
package request

// SearchStockReq 股票搜索请求
type SearchStockReq struct {
	Query    string `form:"q"`
	Topic    string `form:"topic"`
	Category string `form:"category"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"pageSize,default=20"`
}

// Validate 校验并设置默认值
func (r *SearchStockReq) Validate() {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.PageSize <= 0 {
		r.PageSize = 20
	}
}
