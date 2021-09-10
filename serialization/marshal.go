package serialization

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"unicode/utf16"
)

type encodeState struct {
	bufStack []*bytes.Buffer
	buf      *bytes.Buffer
}

func (s *encodeState) WriteTypeMeta(t FabricSerializationType) error {
	return s.writeTypeMeta(t)
}

func (s *encodeState) WriteBinary(v interface{}) error {
	return binary.Write(s.buf, binary.LittleEndian, v)
}

func (s *encodeState) WriteCompressedUInt32(v uint32) error {
	return s.writeCompressedUint32(v)
}

func (s *encodeState) pushBuffer() {
	buf := bytes.NewBuffer(nil)
	s.bufStack = append(s.bufStack, buf)
	s.buf = buf
}

func (s *encodeState) popBuffer() *bytes.Buffer {
	n := len(s.bufStack) - 1
	top := s.bufStack[n]

	s.bufStack = s.bufStack[:n]
	s.buf = s.bufStack[len(s.bufStack)-1]

	return top
}

func (s *encodeState) objectScopeBegin() error {
	s.pushBuffer()
	return nil
}

func (s *encodeState) objectScopeEnd() error {
	objbuf := s.popBuffer()

	err := s.writeTypeMeta(FabricSerializationTypeObject)
	if err != nil {
		return err
	}

	var objectheader objectHeader
	objectheader.Size = uint32(objbuf.Len()) + 3 + sizeOfobjectHeader
	// 3 == FabricSerializationTypeScopeBegin + FabricSerializationTypeScopeEnd + FabricSerializationTypeObjectEnd

	err = binary.Write(s.buf, binary.LittleEndian, &objectheader)
	if err != nil {
		return err
	}

	err = s.writeTypeMeta(FabricSerializationTypeScopeBegin)
	if err != nil {
		return err
	}

	_, err = s.buf.Write(objbuf.Bytes())
	if err != nil {
		return err
	}

	err = s.writeTypeMeta(FabricSerializationTypeScopeEnd)
	if err != nil {
		return err
	}

	err = s.writeTypeMeta(FabricSerializationTypeObjectEnd)
	if err != nil {
		return err
	}

	return nil
}

func (s *encodeState) writeTypeMeta(meta FabricSerializationType) error {
	return s.buf.WriteByte(byte(meta))
}

func (s *encodeState) writeEmpty(rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | FabricSerializationTypeBool)
		} else {
			return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | FabricSerializationTypeBoolFalse)
		}

	case reflect.Uint8, reflect.Int8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64:

		basetyp := kindToFabricSerializationType(rv.Kind())

		if basetyp == FabricSerializationTypeNotAMeta {
			return fmt.Errorf("bad base type meta")
		}

		return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | basetyp)
	case reflect.String:
		return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | FabricSerializationTypeArray | FabricSerializationTypeWString)
	case reflect.Ptr:
		return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | FabricSerializationTypePointer)
	case reflect.Slice:
		elmTyp := rv.Type().Elem()
		switch elmTyp.Kind() {
		case reflect.String, reflect.Ptr:
			return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | FabricSerializationTypeUInt32)
		case reflect.Struct:
			return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | FabricSerializationTypeObject | FabricSerializationTypeArray)
		default:
			basetyp := kindToFabricSerializationType(elmTyp.Kind())

			if basetyp == FabricSerializationTypeNotAMeta {
				return fmt.Errorf("unsupported marshal empty slice type %v", elmTyp)
			}

			return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | basetyp | FabricSerializationTypeArray)
		}
	case reflect.Map:
		return s.writeTypeMeta(FabricSerializationTypeEmptyValueBit | FabricSerializationTypeArray)
	default:
	}

	return fmt.Errorf("unsupported marshal empty type %v", rv.String())
}

func (s *encodeState) writeCompressedUint32(value uint32) error {
	return s.writeCompressedUnsigned(binary.Size(uint32(1)), uint64(value))
}

func (s *encodeState) value(rv reflect.Value) error {

	if rv.Kind() != reflect.Struct && (rv.IsZero() || rv.Kind() == reflect.Bool) {
		return s.writeEmpty(rv)
	}

	switch rv.Kind() {
	case reflect.Int8:
		err := s.writeTypeMeta(FabricSerializationTypeChar)
		if err != nil {
			return err
		}
		return binary.Write(s.buf, binary.LittleEndian, int8(rv.Int()))

	case reflect.Uint8:
		err := s.writeTypeMeta(FabricSerializationTypeUChar)
		if err != nil {
			return err
		}
		return binary.Write(s.buf, binary.LittleEndian, uint8(rv.Uint()))

	case reflect.Uint16, reflect.Uint32, reflect.Uint64:
		basetyp := kindToFabricSerializationType(rv.Kind())

		if basetyp == FabricSerializationTypeNotAMeta {
			return fmt.Errorf("bad base type meta")
		}

		err := s.writeTypeMeta(basetyp)
		if err != nil {
			return err
		}

		return s.writeCompressedUnsigned(int(rv.Type().Size()), rv.Uint())
	case reflect.Int16, reflect.Int32, reflect.Int64:
		basetyp := kindToFabricSerializationType(rv.Kind())

		if basetyp == FabricSerializationTypeNotAMeta {
			return fmt.Errorf("bad base type meta")
		}

		err := s.writeTypeMeta(basetyp)
		if err != nil {
			return err
		}

		return s.writeCompressedSigned(int(rv.Type().Size()), rv.Int())
	case reflect.Float32, reflect.Float64:
		if err := s.writeTypeMeta(FabricSerializationTypeDouble); err != nil {
			return err
		}
		return binary.Write(s.buf, binary.LittleEndian, rv.Float())

	case reflect.String:

		if err := s.writeTypeMeta(FabricSerializationTypeWString | FabricSerializationTypeArray); err != nil {
			return err
		}

		str := utf16.Encode([]rune(rv.String()))
		if err := s.writeCompressedUint32(uint32(len(str))); err != nil {
			return err
		}

		return binary.Write(s.buf, binary.LittleEndian, str)
	case reflect.Ptr:
		if err := s.writeTypeMeta(FabricSerializationTypePointer); err != nil {
			return err
		}

		if err := s.value(reflect.Indirect(rv)); err != nil {
			return err
		}
	case reflect.Struct:
		if cm, ok := castToMarshaler(rv); ok {
			return cm.Marshal(s)
		}

		if err := s.objectScopeBegin(); err != nil {
			return err
		}

		for _, field := range allFields(rv) {
			if err := s.value(field); err != nil {
				return err
			}
		}

		if err := s.objectScopeEnd(); err != nil {
			return err
		}
	case reflect.Slice:

		elmTyp := rv.Type().Elem().Kind()
		switch elmTyp {
		case reflect.String, reflect.Ptr:
			if err := s.writeTypeMeta(FabricSerializationTypeUInt32); err != nil {
				return err
			}
		default:
			baseTyp := kindToFabricSerializationType(elmTyp)

			if baseTyp == FabricSerializationTypeNotAMeta {
				return fmt.Errorf("unsupported slice type %v", elmTyp)
			}
			if err := s.writeTypeMeta(baseTyp | FabricSerializationTypeArray); err != nil {
				return err
			}
		}

		if err := s.writeCompressedUint32(uint32(rv.Len())); err != nil {
			return err
		}

		for i := 0; i < rv.Len(); i++ {
			if err := s.value(rv.Index(i)); err != nil {
				return err
			}
		}
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
		iter := rv.MapRange()
		for iter.Next() {
			entry := reflect.Indirect(reflect.New(sliceTyp))
			entry.Field(0).Set(iter.Key())
			entry.Field(1).Set(iter.Value())
			entries = reflect.Append(entries, entry)
		}

		if err := s.value(entries); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported marshal type %v", rv.String())
	}

	return nil
}

func Marshal(v interface{}) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}

	s := &encodeState{}
	pv := reflect.ValueOf(v)
	if pv.Kind() != reflect.Ptr || pv.IsNil() {
		return nil, fmt.Errorf("marshal type must be ptr")
	}

	rv := reflect.Indirect(pv)
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("marshal type must be ptr to struct")
	}

	s.pushBuffer() // root buf

	if err := s.value(rv); err != nil {
		return nil, err
	}

	return s.buf.Bytes(), nil
}
