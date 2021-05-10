package serialization

import (
	"encoding/binary"
	"reflect"
)

type customMarshaler interface {
	Marshal(*encodeState) error
	Unmarshal(FabricSerializationType, *decodeState) error
}

var customMarshalerType = reflect.TypeOf((*customMarshaler)(nil)).Elem()

func castToMarshaler(rv reflect.Value) (customMarshaler, bool) {
	var v interface{}
	if rv.Kind() != reflect.Ptr && reflect.PtrTo(rv.Type()).Implements(customMarshalerType) {

		v = rv.Addr().Interface()
	} else if rv.Type().Implements(customMarshalerType) {
		v = rv.Interface()
	}

	cm, ok := v.(customMarshaler)
	return cm, ok
}

type headerFlags uint8

const (
	headerFlagsEmpty                   headerFlags = 0x00
	headerFlagsContainsTypeInformation headerFlags = 0x01
	headerFlagsContainsExtensionData   headerFlags = 0x02
)

type objectHeader struct {
	Size    uint32
	Flag    headerFlags
	Padding [3]byte
}

var sizeOfobjectHeader = uint32(binary.Size(objectHeader{}))

func allFields(rv reflect.Value) []reflect.Value {
	if rv.Kind() != reflect.Struct {
		return nil
	}

	var fields []reflect.Value

	typ := rv.Type()

	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i)
		fv := rv.Field(i)

		if !fv.CanSet() {
			continue
		}

		if ft.Anonymous {
			fields = append(fields, allFields(fv)...)
		} else {
			fields = append(fields, fv)
		}
	}

	return fields
}
