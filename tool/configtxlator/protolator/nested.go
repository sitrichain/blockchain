package protolator

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

func nestedFrom(value interface{}, destType reflect.Type) (reflect.Value, error) {
	tree := value.(map[string]interface{}) // Safe, already checked
	result := reflect.New(destType.Elem())
	nMsg := result.Interface().(proto.Message) // Safe, already checked
	if err := recursivelyPopulateMessageFromTree(tree, nMsg); err != nil {
		return reflect.Value{}, err
	}
	return result, nil
}

func nestedTo(value reflect.Value) (interface{}, error) {
	nMsg := value.Interface().(proto.Message) // Safe, already checked
	return recursivelyCreateTreeFromMessage(nMsg)
}

var timestampType = reflect.TypeOf(&types.Timestamp{})

type nestedFieldFactory struct{}

func (nff nestedFieldFactory) Handles(_ proto.Message, _ string, fieldType reflect.Type, _ reflect.Value) bool {
	// Note, we skip recursing into the field if it is a proto native timestamp, because there is other custom marshaling this conflicts with
	// this should probably be revisited more generally to prevent custom marshaling of 'well known messages'
	return fieldType.Kind() == reflect.Ptr && fieldType.AssignableTo(protoMsgType) && !fieldType.AssignableTo(timestampType)
}

func (nff nestedFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &plainField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: nestedFrom,
		populateTo:   nestedTo,
	}, nil
}

type nestedMapFieldFactory struct{}

func (nmff nestedMapFieldFactory) Handles(_ proto.Message, _ string, fieldType reflect.Type, _ reflect.Value) bool {
	return fieldType.Kind() == reflect.Map && fieldType.Elem().AssignableTo(protoMsgType) && !fieldType.Elem().AssignableTo(timestampType)
}

func (nmff nestedMapFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &mapField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(k string, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return nestedFrom(v, dT)
		},
		populateTo: func(k string, v reflect.Value) (interface{}, error) {
			return nestedTo(v)
		},
	}, nil
}

type nestedSliceFieldFactory struct{}

func (nmff nestedSliceFieldFactory) Handles(_ proto.Message, _ string, fieldType reflect.Type, _ reflect.Value) bool {
	return fieldType.Kind() == reflect.Slice && fieldType.Elem().AssignableTo(protoMsgType) && !fieldType.Elem().AssignableTo(timestampType)
}

func (nmff nestedSliceFieldFactory) NewProtoField(msg proto.Message, fieldName string, fieldType reflect.Type, fieldValue reflect.Value) (protoField, error) {
	return &sliceField{
		baseField: baseField{
			msg:   msg,
			name:  fieldName,
			fType: mapStringInterfaceType,
			vType: fieldType,
			value: fieldValue,
		},
		populateFrom: func(i int, v interface{}, dT reflect.Type) (reflect.Value, error) {
			return nestedFrom(v, dT)
		},
		populateTo: func(i int, v reflect.Value) (interface{}, error) {
			return nestedTo(v)
		},
	}, nil
}
