package conf

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
)

// 从环境变量中读取字符串
func readString(key string) (string, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("Empty string from env %s", key)
	}
	return v, nil
}

// 从环境变量中读取整形
func readInt64(key string) (int64, error) {
	s, err := readString(key)
	if err != nil {
		return 0, fmt.Errorf("Empty int from env %s", key)
	}
	v, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("Convert int from env %s error. %w", key, err)
	}
	return v, nil
}

// 从环境变量中读取布尔值
func readBool(key string) (bool, error) {
	s, err := readString(key)
	if err != nil {
		return false, fmt.Errorf("Empty bool from env %s", key)
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("Convert bool from env %s error. %w", key, err)
	}
	return v, nil
}

// 从环境变量中读取[]string, 原始字符串需以,分割
func readStringSlice(key string) (reflect.Value, error) {
	s, err := readString(key)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("Empty string slice from env %s", key)
	}
	slice := strings.Split(s, ",")
	v := reflect.MakeSlice(reflect.TypeOf([]string{}), len(slice), len(slice))
	for i := range slice {
		v.Index(i).SetString(slice[i])
	}
	return v, nil
}

// 从环境变量中读取时间
func readDuration(key string) (int64, error) {
	s, err := readString(key)
	if err != nil {
		return 0, fmt.Errorf("Empty duration from env %s", key)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("Invalid duration from env %s. %w", key, err)
	}
	return d.Nanoseconds(), nil
}

// 从环境变量中读取kafka版本
func readKafkaVersion(key string) (reflect.Value, error) {
	s, err := readString(key)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("Empty 4 uint array from env %s", key)
	}
	value, err := sarama.ParseKafkaVersion(s)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("Invalid kafka version from env %s. %w", key, err)
	}
	return reflect.ValueOf(value), nil
}

// 从环境变量中读取float64
func readFloat64(key string) (float64, error) {
	s, err := readString(key)
	if err != nil {
		return 0, fmt.Errorf("Empty float from env %s", key)
	}
	v, err := strconv.ParseFloat(s, 0)
	if err != nil {
		return 0, fmt.Errorf("Convert float from env %s error. %w", key, err)
	}
	return v, nil
}
