package reconcile

import (
	"fmt"
	"reflect"
	"strings"
)

type MyType struct {
	A int
	B struct {
		c string
		d []int
		E *MyType
	}
	D interface{}
}

type ReflectWrapper struct {
	Obj interface{}
}

func (w *ReflectWrapper) Set(path string, value interface{}) *ReflectWrapper {
	src := reflect.ValueOf(w.Obj)
	val := reflect.ValueOf(value)
	fields := strings.Split(path, ".")
	set(src, val, fields)
	return w
}

func set(src reflect.Value, val reflect.Value, fields []string) {
	if len(fields) == 0 {
		src.Set(val)
		return
	}
	if src.Kind() == reflect.Ptr {
		if src.IsNil() {
			e := reflect.New(src.Type().Elem())
			src.Set(e)
		}
		src = src.Elem()
	}
	if src.Kind() == reflect.Interface {
		tmp := reflect.New(src.Elem().Type()).Elem()
		tmp.Set(src.Elem())
		set(tmp, val, fields)
		src.Set(tmp)
		return
	}

	if v := src.FieldByName(fields[0]); v.IsValid() {
		set(v, val, fields[1:])
	} else {
		panic(fmt.Sprintf("field %s not found on type %v", fields[0], src.Type()))
	}
}
