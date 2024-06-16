package web

import (
	"strings"
	"unicode"
	"unsafe"
)

func SubStringLast(str string, substr string) string {
	// 查找子字符串在主字符串中第一次出现的位置
	index := strings.Index(str, substr)
	// 如果子字符串没有找到，返回空字符串
	if index == -1 {
		return ""
	}
	// 获取子字符串的长度
	len := len(substr)
	// 返回从子字符串结束位置开始的剩余部分
	return str[index+len:]
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
