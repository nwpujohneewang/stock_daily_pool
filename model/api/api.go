// Package api provides unified API response types and helpers.
package api

// Response 统一 API 响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 成功响应
func OK(data interface{}) Response {
	return Response{Code: 0, Message: "ok", Data: data}
}

// Fail 失败响应
func Fail(code int, message string) Response {
	return Response{Code: code, Message: message}
}

// FailWithData 带数据的失败响应
func FailWithData(code int, message string, data interface{}) Response {
	return Response{Code: code, Message: message, Data: data}
}

// Created 创建成功响应 (HTTP 201)
func Created(data interface{}) Response {
	return Response{Code: 0, Message: "created", Data: data}
}
