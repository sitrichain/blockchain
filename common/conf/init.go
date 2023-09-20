package conf

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/rongzer/blockchain/common/log"
)

// 环境变量名前缀
const _prefix = "BLOCKCHAIN"

// Initialize 初始化配置
func Initialize() error {
	v := reflect.ValueOf(&V).Elem()
	t := reflect.TypeOf(V)
	if t.Kind() != reflect.Struct {
		return errors.New("default config value expect struct")
	}
	if err := walkFileds(_prefix, v, t); err != nil {
		return fmt.Errorf("reflect config fields error %w", err)
	}
	return nil
}

// 遍历字段
func walkFileds(prefix string, v reflect.Value, t reflect.Type) error {
	for i := 0; i < t.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := t.Field(i)
		key := strings.ToUpper(prefix + "_" + fieldType.Name)
		switch fieldType.Type.Kind() {
		case reflect.Bool:
			value, err := readBool(key)
			if err != nil {
				log.Logger.Debug(err)
				continue
			}
			if fieldValue.CanSet() {
				fieldValue.SetBool(value)
			}
		case reflect.Int:
			fallthrough
		case reflect.Int64:
			var value int64
			var err error
			if fieldType.Type.String() == "time.Duration" {
				value, err = readDuration(key)
			} else {
				value, err = readInt64(key)
			}
			if err != nil {
				log.Logger.Debug(err)
				continue
			}
			if fieldValue.CanSet() {
				fieldValue.SetInt(value)
			}
		case reflect.Float64:
			value, err := readFloat64(key)
			if err != nil {
				log.Logger.Debug(err)
				continue
			}
			if fieldValue.CanSet() {
				fieldValue.SetFloat(value)
			}
		case reflect.Ptr:
			if !v.Field(i).IsNil() {
				if err := walkFileds(key, v.Field(i).Elem(), fieldType.Type.Elem()); err != nil {
					return err
				}
			}
		case reflect.Slice:
			if fieldType.Type.String() == "[]string" {
				value, err := readStringSlice(key)
				if err != nil {
					log.Logger.Debug(err)
					continue
				}
				if fieldValue.CanSet() {
					fieldValue.Set(value)
				}
			}

		case reflect.String:
			value, err := readString(key)
			if err != nil {
				log.Logger.Debug(err)
				continue
			}
			if fieldValue.CanSet() {
				fieldValue.SetString(value)
			}
		case reflect.Struct:
			if fieldType.Type.String() == "sarama.KafkaVersion" {
				value, err := readKafkaVersion(key)
				if err != nil {
					log.Logger.Debug(err)
					continue
				}
				fieldValue.Set(value)
				continue
			}
			if err := walkFileds(key, v.Field(i), fieldType.Type); err != nil {
				return err
			}
		default:
			log.Logger.Debugf("Unsupported config field: %s %v %d", fieldType.Name, fieldType.Type, fieldType.Type.Kind())
		}
	}

	return nil
}
