package serialization

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"testing"
)

// TODO test binary values

func TestCompressedEmpty(t *testing.T) {
	ec := &encodeState{}
	ec.pushBuffer()
	err := ec.writeCompressedSigned(int(binary.Size(int64(1))), 0)
	if err != nil {
		t.Errorf("compress error = %v", err)
		return
	}

	err = ec.writeCompressedUnsigned(int(binary.Size(uint64(1))), 0)
	if err != nil {
		t.Errorf("compress error = %v", err)
		return
	}

	if ec.buf.Len() != 0 {
		t.Errorf("buf should be empty")
	}
}

func TestCompressedSigned(t *testing.T) {
	tests := []struct {
		value interface{}
	}{
		{
			value: int16(1),
		},
		{
			value: int32(1),
		},
		{
			value: int16(42),
		},
		{
			value: int32(42),
		},
		{
			value: int32(-42),
		},
		{
			value: int32(math.MaxInt32),
		},
		{
			value: int32(math.MinInt32),
		},
		{
			value: int16(math.MaxInt16),
		},
		{
			value: int16(math.MinInt16),
		},
		{
			value: int64(math.MaxInt64),
		},
		{
			value: int64(math.MinInt64),
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("compress %v", tt.value), func(t *testing.T) {

			ec := &encodeState{}
			ec.pushBuffer()
			rv := reflect.ValueOf(tt.value)
			err := ec.writeCompressedSigned(int(rv.Type().Size()), rv.Int())

			if err != nil {
				t.Errorf("compress error = %v", err)
				return
			}

			dc := decodeState{bytes.NewReader(ec.buf.Bytes())}

			v, err := dc.readCompressedSigned(int(rv.Type().Size()))
			if err != nil {
				t.Errorf("decompress error = %v", err)
				return
			}

			if v != rv.Int() {
				t.Errorf("value changed got %v, expect %v", v, rv.Int())
				return
			}
		})
	}
}

func TestCompressedUnsigned(t *testing.T) {
	tests := []struct {
		value interface{}
	}{
		{
			value: uint16(1),
		},
		{
			value: uint16(42),
		},
		{
			value: uint32(42),
		},
		{
			value: uint32(math.MaxUint32),
		},
		{
			value: uint16(math.MaxUint16),
		},
		{
			value: uint64(math.MaxUint64),
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("compress %v", tt.value), func(t *testing.T) {

			ec := &encodeState{}
			ec.pushBuffer()
			rv := reflect.ValueOf(tt.value)
			err := ec.writeCompressedUnsigned(int(rv.Type().Size()), rv.Uint())

			if err != nil {
				t.Errorf("compress error = %v", err)
				return
			}

			dc := decodeState{bytes.NewReader(ec.buf.Bytes())}

			v, err := dc.readCompressedUnsigned(int(rv.Type().Size()))
			if err != nil {
				t.Errorf("decompress error = %v", err)
				return
			}

			if v != rv.Uint() {
				t.Errorf("value changed got %v, expect %v", v, rv.Uint())
				return
			}
		})
	}
}
