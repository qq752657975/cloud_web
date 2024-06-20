package msstrings

import (
	"fmt"
	"reflect"
	"strings"
)

func JoinStrings(str ...any) string {
	var sb strings.Builder
	for _, v := range str {
		sb.WriteString(check(v))
	}
	return sb.String()
}

func check(v any) string {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.String:
		return v.(string)
	//case reflect.Int:
	//	vv := v.(int)
	//	return strconv.FormatInt(int64(vv), 10)
	//case reflect.Int64:
	//	vv := v.(int64)
	//	return strconv.FormatInt(vv, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}
