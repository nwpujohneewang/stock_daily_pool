package utils

import "github.com/bytedance/sonic"

func ToString(val interface{}) string {
	str, _ := sonic.MarshalString(val)
	return str
}
