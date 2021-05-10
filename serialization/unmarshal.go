package serialization

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"unicode/utf16"
)

type decodeState struct {
	inner *bytes.Reader
}

func (s *decodeState) ReadBinary(v interface{}) error {
	return binary.Read(s.inner, binary.LittleEndian, v)
}

func (s *decodeState) ReadCompressedUInt32() (uint32, error) {
	return s.readCompressedUInt32()
}

func IsEmptyMeta(meta FabricSerializationType) bool {
	return meta&FabricSerializationTypeEmptyValueBit > 0
}

func IsArrayMeta(meta FabricSerializationType) bool {
	return meta&FabricSerializationTypeArray > 0
}

func IsBaseMeta(meta, base FabricSerializationType) bool {
	return (meta & FabricSerializationTypeBaseTypeMask) == base
}

// func (s *decodeState) dumpCurrentPos() {
// 	c, _ := s.inner.Seek(0, io.SeekCurrent)
// 	x := make([]byte, 10)
// 	s.inner.ReadAt(x, c)
// 	log.Printf("pos %v %v", c, hex.EncodeToString(x))
// }

// func (s *decodeState) readCompressedUInt64() (uint64, error) {
// 	v, err := s.readCompressedUnsigned(binary.Size(uint64(1)))
// 	return uint64(v), err
// }

func (s *decodeState) readCompressedUInt32() (uint32, error) {
	v, err := s.readCompressedUnsigned(binary.Size(uint32(1)))
	return uint32(v), err
}

// func (s *decodeState) readCompressedUInt16() (uint16, error) {
// 	v, err := s.readCompressedUnsigned(binary.Size(uint16(1)))
// 	return uint16(v), err
// }

// func (s *decodeState) readCompressedInt64() (int64, error) {
// 	v, err := s.readCompressedSigned(binary.Size(int64(1)))
// 	return int64(v), err
// }

// func (s *decodeState) readCompressedInt32() (int32, error) {
// 	v, err := s.readCompressedSigned(binary.Size(int32(1)))
// 	return int32(v), err
// }

// func (s *decodeState) readCompressedInt16() (int16, error) {
// 	v, err := s.readCompressedSigned(binary.Size(int16(1)))
// 	return int16(v), err
// }

func (s *decodeState) readTypeMeta() (FabricSerializationType, error) {
	var meta FabricSerializationType
	err := binary.Read(s.inner, binary.LittleEndian, &meta)
	if err != nil {
		return FabricSerializationTypeNotAMeta, err
	}

	return meta, nil
}

func (s *decodeState) expectTypeMeta(expectMeta FabricSerializationType) error {
	meta, err := s.readTypeMeta()
	if err != nil {
		return err
	}

	if meta != expectMeta {
		return fmt.Errorf("got %v expect %v", meta, expectMeta)
	}

	return nil
}

func (s *decodeState) readObjectBegin(meta FabricSerializationType) (int64, error) {
	if meta != FabricSerializationTypeObject {
		return -1, nil
	}

	var objectheader objectHeader

	headerPosition, err := s.inner.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}

	binary.Read(s.inner, binary.LittleEndian, &objectheader)
	if objectheader.Flag&headerFlagsContainsTypeInformation == headerFlagsContainsTypeInformation {

		// TODO no obj activator in go, discard type info at the moment
		// struct must set exact type info
		len, err := s.readCompressedUInt32()
		if err != nil {
			return -1, err
		}

		if len == 0 {
			return -1, fmt.Errorf("typeinfo len must > 0")
		}

		if _, err := io.CopyN(io.Discard, s.inner, int64(len)); err != nil {
			return -1, err
		}
	}

	if err := s.expectTypeMeta(FabricSerializationTypeScopeBegin); err != nil {
		return -1, err
	}

	return headerPosition + int64(objectheader.Size) - 2, nil
}

func (s *decodeState) consumeObjectEnd(meta FabricSerializationType, endpos int64) error {
	if meta != FabricSerializationTypeObject {
		return nil
	}

	_, err := s.inner.Seek(endpos, io.SeekStart)
	if err != nil {
		return err
	}

	if err := s.expectTypeMeta(FabricSerializationTypeScopeEnd); err != nil {
		return err
	}

	if err := s.expectTypeMeta(FabricSerializationTypeObjectEnd); err != nil {
		return err
	}

	return nil
}

func (s *decodeState) value(meta FabricSerializationType, rv reflect.Value) error {
	if IsEmptyMeta(meta) {

		// bool is alway empty
		if rv.Kind() == reflect.Bool {
			if meta == FabricSerializationTypeBool|FabricSerializationTypeEmptyValueBit {
				rv.SetBool(true)
			} else if meta == FabricSerializationTypeBoolFalse|FabricSerializationTypeEmptyValueBit {
				rv.SetBool(false)
			} else {
				return fmt.Errorf("expect bool got %v", meta)
			}
		} else {
			// other kind
			// TODO basetype check
			rv.Set(reflect.Zero(rv.Type()))
		}

		return nil
	}

	switch rv.Kind() {
	case reflect.Uint8, reflect.Int8:
		v, err := s.inner.ReadByte()

		if err != nil {
			return err
		}

		switch meta {
		case FabricSerializationTypeChar:
			rv.SetInt(int64(v))
		case FabricSerializationTypeUChar:
			rv.SetUint(uint64(v))
		default:
			return fmt.Errorf("expect char/uchar got %v", meta)
		}

	case reflect.Uint16, reflect.Uint32, reflect.Uint64:

		switch meta {
		case FabricSerializationTypeUShort, FabricSerializationTypeUInt32, FabricSerializationTypeUInt64:
			v, err := s.readCompressedUnsigned(int(rv.Type().Size()))
			if err != nil {
				return err
			}

			rv.SetUint(v)
		default:
			return fmt.Errorf("expect uint got %v", meta)
		}
	case reflect.Int16, reflect.Int32, reflect.Int64:
		switch meta {
		case FabricSerializationTypeShort, FabricSerializationTypeInt32, FabricSerializationTypeInt64:
			v, err := s.readCompressedSigned(int(rv.Type().Size()))
			if err != nil {
				return err
			}

			rv.SetInt(v)
		default:
			return fmt.Errorf("expect int got %v", meta)
		}
	case reflect.Float32, reflect.Float64:
		if meta != FabricSerializationTypeDouble {
			return fmt.Errorf("expect double got %v", meta)
		}

		var v float64

		err := binary.Read(s.inner, binary.LittleEndian, &v)
		if err != nil {
			return err
		}

		rv.SetFloat(v)

	case reflect.String:

		if meta != FabricSerializationTypeWString|FabricSerializationTypeArray {
			return fmt.Errorf("expect []string got %v", meta)
		}

		len, err := s.readCompressedUInt32()
		if err != nil {
			return err
		}

		body := make([]uint16, len) // wchar

		err = binary.Read(s.inner, binary.LittleEndian, &body)
		if err != nil {
			return err
		}

		rv.SetString(string(utf16.Decode(body)))

	case reflect.Ptr:
		ptr := reflect.New(rv.Type().Elem())

		objmeta, err := s.readTypeMeta()
		if err != nil {
			return err
		}

		if err := s.value(objmeta, reflect.Indirect(ptr)); err != nil {
			return err
		}

		rv.Set(ptr)

	case reflect.Struct:
		if cm, ok := castToMarshaler(rv); ok {
			return cm.Unmarshal(meta, s)
		}

		endPos, err := s.readObjectBegin(meta)
		if err != nil {
			return err
		}

		for _, field := range allFields(rv) {
			meta, err := s.readTypeMeta()
			if err != nil {
				return err
			}

			if meta == FabricSerializationTypeScopeEnd {
				break
			}

			err = s.value(meta, field)
			if err != nil {
				return err
			}
		}

		err = s.consumeObjectEnd(meta, endPos)
		if err != nil {
			return err
		}

	case reflect.Slice:

		switch rv.Type().Elem().Kind() {
		case reflect.String:
			if meta != FabricSerializationTypeUInt32 {
				return fmt.Errorf("[]string count expect uint32 got %v", meta)
			}
		case reflect.Struct:
			if meta != FabricSerializationTypeObject|FabricSerializationTypeArray {
				return fmt.Errorf("[]struct{} expect array got %v", meta)
			}
		}

		len0, err := s.readCompressedUInt32()
		len := int(len0)
		if err != nil {
			return err
		}

		objs := reflect.MakeSlice(reflect.SliceOf(rv.Type().Elem()), len, len)

		for i := 0; i < len; i++ {

			meta, err := s.readTypeMeta()
			if err != nil {
				return err
			}

			err = s.value(meta, objs.Index(i))

			if err != nil {
				return err
			}
		}

		rv.Set(objs)

	case reflect.Map:
		keytyp := rv.Type().Key()
		valtyp := rv.Type().Elem()
		sliceTyp := reflect.StructOf([]reflect.StructField{
			{
				Name: "Key",
				Type: keytyp,
			},
			{
				Name: "Value",
				Type: valtyp,
			},
		})

		entries := reflect.Indirect(reflect.New(reflect.SliceOf(sliceTyp)))
		err := s.value(meta, entries)
		if err != nil {
			return err
		}

		m := reflect.MakeMap(reflect.MapOf(keytyp, valtyp))

		for i := 0; i < entries.Len(); i++ {
			kv := entries.Index(i)
			m.SetMapIndex(kv.Field(0), kv.Field(1))
		}

		rv.Set(m)

	default:
		return fmt.Errorf("unsupported unmarshal type %v", rv.String())
	}

	return nil
}

func Unmarshal(data []byte, v interface{}) error {
	pv := reflect.ValueOf(v)
	if pv.Kind() != reflect.Ptr || pv.IsNil() {
		return fmt.Errorf("unmarshal type must be ptr")
	}

	rv := reflect.Indirect(pv)
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal type must be ptr to struct")
	}

	d := decodeState{bytes.NewReader(data)}
	meta, err := d.readTypeMeta()
	if err != nil {
		return err
	}

	return d.value(meta, rv)
}
