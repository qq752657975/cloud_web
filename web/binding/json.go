package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

type jsonBinding struct {
	DisallowUnknownFields bool
	IsValidate            bool
}

func (j *jsonBinding) Name() string {
	return "json"
}

func (j *jsonBinding) Bind(r *http.Request, obj any) error {
	body := r.Body
	if body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	if j.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if j.IsValidate {
		err := validateRequireParam(obj, decoder)
		if err != nil {
			return err
		}
	} else {
		err := decoder.Decode(obj)
		if err != nil {
			return err
		}
	}
	return validate(obj)
}

func validateRequireParam(data any, decoder *json.Decoder) error {
	if data == nil {
		return nil
	}
	//反射
	valueOf := reflect.ValueOf(data)
	//判断其是否为指针类型
	if valueOf.Kind() != reflect.Pointer {
		return errors.New("no ptr type")
	}
	t := valueOf.Elem().Interface()
	of := reflect.ValueOf(t)
	switch of.Kind() {
	case reflect.Struct:
		return checkParam(of, data, decoder)
	case reflect.Slice, reflect.Array:
		elem := of.Type().Elem()
		elemType := elem.Kind()
		if elemType == reflect.Struct {
			return checkParamSlice(elem, data, decoder)
		}
	default:
		err := decoder.Decode(data)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkParamSlice(elem reflect.Type, data any, decoder *json.Decoder) error {
	mapData := make([]map[string]interface{}, 0)
	_ = decoder.Decode(&mapData)
	if len(mapData) <= 0 {
		return nil
	}
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		required := field.Tag.Get("web")
		tag := field.Tag.Get("json")
		for _, v := range mapData {
			value := v[field.Name]
			if value == nil && required == "required" {
				return errors.New(fmt.Sprintf("filed [%s] is required", tag))
			}
		}
	}
	if data != nil {
		marshal, _ := json.Marshal(mapData)
		_ = json.Unmarshal(marshal, data)
	}
	return nil
}

func checkParam(of reflect.Value, data any, decoder *json.Decoder) error {
	//解析为map，然后根据map中的key，进行对比
	//判断类型结构体，才能解析map
	mapData := make(map[string]interface{})
	_ = decoder.Decode(&mapData)
	for i := 0; i < of.NumField(); i++ {
		field := of.Type().Field(i)
		tag := field.Tag.Get("json")
		//添加对自定义web标签的支持
		required := field.Tag.Get("web")
		value := mapData[tag]
		if value == nil && required == "required" {
			return errors.New(fmt.Sprintf("filed [%s] is required", tag))
		}
	}
	marshal, _ := json.Marshal(mapData)
	_ = json.Unmarshal(marshal, data)
	return nil
}
