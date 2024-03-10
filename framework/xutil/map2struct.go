package xutil

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

func FillStruct(data map[string]interface{}, obj interface{}) error {
	for k, v := range data {
		//fmt.Println("k: ", k, " v: ", v)
		err := SetField(obj, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetField(obj interface{}, k string, v interface{}) error {
	//结构体属性值
	structValue := reflect.ValueOf(obj).Elem()
	//fmt.Println("structValue: ", structValue)
	//结构体单个属性值
	structFieldValue := structValue.FieldByName(k)
	//fmt.Println("structFieldValue: ", structFieldValue)
	if !structFieldValue.IsValid() {
		return fmt.Errorf("no such field: %s in obj", k)
	}
	if !structFieldValue.CanSet() {
		return fmt.Errorf("cannot set %s field value", k)
	}
	//结构体属性类型
	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(v)
	var err error
	if structFieldType != val.Type() {
		//类型转换
		val, err = TypeConversion(fmt.Sprintf("%v", v), structFieldValue.Type().Name())
		if err != nil {
			return err
		}
	}
	structFieldValue.Set(val)
	return nil
}

func TypeConversion(value string, ntype string) (reflect.Value, error) {
	fmt.Println("call TypeConversion...")
	if ntype == "string" {
		return reflect.ValueOf(value), nil
	} else if ntype == "time.Time" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		return reflect.ValueOf(t), err
	} else if ntype == "Time" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		return reflect.ValueOf(t), err
	} else if ntype == "int" {
		i, err := strconv.Atoi(value)
		return reflect.ValueOf(i), err
	} else if ntype == "int8" {
		i, err := strconv.ParseInt(value, 10, 64)
		return reflect.ValueOf(int8(i)), err
	} else if ntype == "int32" {
		i, err := strconv.ParseInt(value, 10, 64)
		return reflect.ValueOf(int64(i)), err
	} else if ntype == "int64" {
		i, err := strconv.ParseInt(value, 10, 64)
		return reflect.ValueOf(i), err
	} else if ntype == "float32" {
		i, err := strconv.ParseFloat(value, 64)
		return reflect.ValueOf(float32(i)), err
	} else if ntype == "float64" {
		i, err := strconv.ParseFloat(value, 64)
		return reflect.ValueOf(i), err
	}

	//else if .......增加其他一些类型的转换

	return reflect.ValueOf(value), errors.New("未知的类型：" + ntype)
}
