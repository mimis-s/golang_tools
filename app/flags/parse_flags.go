package flags

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"unsafe"

	"gopkg.in/yaml.v3"
)

// ParseWithStructPointers 启动参数解析，如果启动参数没有指定，会去env里查找同名参数
// flagStructPointers为结构体指针数组
func ParseWithStructPointers(flagStructPointers ...interface{}) {

	for _, st := range flagStructPointers {
		flagParseStruct2Flags(st)
	}

	flag.Parse()
}

func flagParseStruct2Flags(st interface{}) {
	if st == nil {
		return
	}

	var stTo = reflect.TypeOf(st)
	var stVo = reflect.ValueOf(st)
	switch stTo.Kind() {
	case reflect.Ptr:
		stTo = stTo.Elem()
		stVo = stVo.Elem()
	// case reflect.Struct:
	// 	break
	default:
		panic(fmt.Errorf("invalid flags parse struct(%+v), must be pointer or struct", st))
	}

	for i := 0; i < stTo.NumField(); i++ {
		field := stTo.Field(i)

		key, find := field.Tag.Lookup("env")
		if !find {
			continue
		}

		desc := field.Tag.Get("desc")

		defaultValue, find := os.LookupEnv(key)
		if !find {
			defaultValue, find = field.Tag.Lookup("default")
			if !find {
				defaultValue = fmt.Sprintf("not_found_env_%v", key)
			}
		}

		var fieldValuePointer = unsafe.Pointer(stVo.Field(i).Addr().Pointer())
		switch field.Type.Kind() {
		case reflect.String:
			flag.StringVar((*string)(fieldValuePointer), key, defaultValue, desc)
		case reflect.Int:
			defaultValue1, _ := strconv.Atoi(defaultValue)
			flag.IntVar((*int)(fieldValuePointer), key, defaultValue1, desc)
		case reflect.Int64:
			defaultValue1, _ := strconv.ParseInt(defaultValue, 10, 64)
			flag.Int64Var((*int64)(fieldValuePointer), key, defaultValue1, desc)
		case reflect.Bool:
			flag.BoolVar((*bool)(fieldValuePointer), key, defaultValue == "true", desc)
		default:
			panic(fmt.Errorf("parse flag kind invalid,must be string/int/int64/bool, not %+v", field.Type.Kind()))
		}
	}

	return
}

// 解析yaml配置文件
func ParseWithConfigFileContent(configPath string, configStruct interface{}) error {

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		newErr := fmt.Errorf("load boot config file %v error:%v", configPath, err)
		return newErr
	}

	err = yaml.Unmarshal(content, configStruct)
	if err != nil {
		newErr := fmt.Errorf("load boot config file %v content %v ok, but parse content error:%v",
			configStruct, string(content), err)
		return newErr
	}

	return nil
}
