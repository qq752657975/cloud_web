package binding

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"sync"
)

type StructValidator interface {
	//结构体验证，如果错误返回对应的错误信息
	ValidateStruct(any) error
	//返回对应使用的验证器
	Engine() any
}

var Validator StructValidator = &defaultValidator{}

type defaultValidator struct {
	one      sync.Once
	validate *validator.Validate
}

type SliceValidationError []error

func (err SliceValidationError) Error() string {
	n := len(err)
	switch n {
	case 0:
		return ""
	default:
		var b strings.Builder
		if err[0] != nil {
			fmt.Fprintf(&b, "[%d]: %s", 0, err[0].Error())
		}
		if n > 1 {
			for i := 1; i < n; i++ {
				if err[i] != nil {
					b.WriteString("\n")
					fmt.Fprintf(&b, "[%d]: %s", i, err[i].Error())
				}
			}
		}
		return b.String()
	}
}

func (d *defaultValidator) ValidateStruct(obj any) error {
	of := reflect.ValueOf(obj)
	switch of.Kind() {
	case reflect.Pointer:
		return d.ValidateStruct(of.Elem().Interface())
	case reflect.Struct:
		return d.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := of.Len()
		sliceValidationError := make(SliceValidationError, 0)
		for i := 0; i < count; i++ {
			if err := d.validateStruct(of.Index(i).Interface()); err != nil {
				sliceValidationError = append(sliceValidationError, err)
			}
		}
		if len(sliceValidationError) == 0 {
			return nil
		}
		return sliceValidationError
	default:
		panic("unhandled default case")
	}
	return nil
}

func (d *defaultValidator) Engine() any {
	d.lazyInit()
	return d.validate
}

func (d *defaultValidator) lazyInit() {
	d.one.Do(func() {
		d.validate = validator.New()
	})
}

func (d *defaultValidator) validateStruct(obj any) error {
	d.lazyInit()
	return d.validate.Struct(obj)
}

func validate(obj any) error {
	return Validator.ValidateStruct(obj)
}
