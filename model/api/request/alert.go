// model/api/request/alert.go
package request

// GetAlertsReq 获取告警请求
type GetAlertsReq struct {
	Date string `form:"date"`
}
