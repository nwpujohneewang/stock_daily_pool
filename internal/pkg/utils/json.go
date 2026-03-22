package utils

import (
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func ToString(val interface{}) string {
	str, _ := json.Marshal(val)
	return string(str)
}
