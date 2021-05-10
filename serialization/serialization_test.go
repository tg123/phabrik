package serialization

import (
	"reflect"
	"testing"
)

func TestBasicSerialization(t *testing.T) {

	type BasicObject struct {
		Char1     int8
		Uchar1    uint8
		Short1    int16
		Ushort1   uint16
		Bool1     bool
		Ulong64_1 uint64
		Long64_1  int64

		Double float64

		String string

		Ulong64ArraySize uint32
		Ulong64Array     []int64

		Guid GUID
	}

	var object BasicObject

	object.Short1 = -10
	object.Ushort1 = 10
	object.Bool1 = true
	object.Uchar1 = 0xF8
	object.Char1 = 'd'
	object.Ulong64_1 = 0xFFFFFFFFFFFFFFFF
	object.Long64_1 = 0x0FFFFFFFFFFFFFFF
	object.Double = 89.3
	object.String = "Hello object"
	object.Ulong64ArraySize = 16
	object.Ulong64Array = make([]int64, object.Ulong64ArraySize)

	for i := uint32(0); i < object.Ulong64ArraySize; i++ {
		object.Ulong64Array[i] = int64(i)
	}

	// {14E4F405-BA48-4B51-8084-0B6C5523F29E}
	object.Guid = GUID{0x14e4f405, 0xba48, 0x4b51, [8]byte{0x80, 0x84, 0xb, 0x6c, 0x55, 0x23, 0xf2, 0x9e}}

	data, err := Marshal(&object)
	if err != nil {
		t.Fatal(err)
	}

	var object2 BasicObject
	err = Unmarshal(data, &object2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(object, object2) {
		t.Fatal("not equal")
	}
}
