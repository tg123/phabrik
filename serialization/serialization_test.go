package serialization

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	Ulong64Array     []uint64

	Guid GUID
}

func TestBasicSerialization(t *testing.T) {
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
	object.Ulong64Array = make([]uint64, object.Ulong64ArraySize)

	for i := uint32(0); i < object.Ulong64ArraySize; i++ {
		object.Ulong64Array[i] = uint64(i)
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

func TestBasicSerializationWithChild(t *testing.T) {
	type BasicObjectChild struct {
		BasicObject
		Clong1 int32
	}

	var object BasicObjectChild

	object.Clong1 = 0xfab61c

	object.Short1 = 1000
	object.Ushort1 = 10454
	object.Bool1 = false
	object.Uchar1 = 0x0b
	object.Char1 = 'v'
	object.Ulong64_1 = 0xFFFFFFFFFF
	object.Long64_1 = -1
	object.Double = -9.343

	object.Ulong64ArraySize = 100
	object.Ulong64Array = make([]uint64, object.Ulong64ArraySize)

	for i := uint32(0); i < object.Ulong64ArraySize; i++ {
		object.Ulong64Array[i] = 0xFFFFFFFFF - uint64(i)*13
	}

	object.String = "striiiing"

	// {14E4F405-BA48-4B51-8084-0B6C5523F29E}
	object.Guid = GUID{0x14e4f405, 0xba48, 0x4b51, [8]byte{0x80, 0x84, 0xb, 0x6c, 0x55, 0x23, 0xf2, 0x9e}}

	{
		data, err := Marshal(&object)
		if err != nil {
			t.Fatal(err)
		}

		var object2 BasicObjectChild
		err = Unmarshal(data, &object2)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(object, object2) {
			t.Fatal("not equal")
		}
	}

	// make sure embedded obj
	{
		data, err := Marshal(&struct {
			Char1   int8
			Uchar1  uint8
			Short1  int16
			Ushort1 uint16
		}{
			Char1:   42,
			Uchar1:  1,
			Ushort1: 9527,
		})
		if err != nil {
			t.Fatal(err)
		}

		var object2 BasicObjectChild
		err = Unmarshal(data, &object2)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, int8(42), object2.Char1)
		assert.Equal(t, uint8(1), object2.Uchar1)
		assert.Equal(t, uint16(9527), object2.Ushort1)
	}
}

type BasicObjectVersion struct {
	Ulong uint32
	Bool  bool
}

type BasicChildObjectVersion struct {
	BasicObjectVersion
	Short int16
	Guid  GUID
}

func TestBasicVersioningChildReadBase(t *testing.T) {

	var object1 BasicChildObjectVersion

	object1.Bool = false
	object1.Short = 999
	object1.Ulong = 0xDDDD
	object1.Guid = *MustNewGuidV4()

	data, err := Marshal(&object1)
	if err != nil {
		t.Fatal(err)
	}

	var object2 BasicObjectVersion
	err = Unmarshal(data, &object2)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint32(0xDDDD), object2.Ulong)
	assert.Equal(t, false, object2.Bool)
}

func TestBasicVersioningChild2(t *testing.T) {
	type BasicChild2ObjectVersion struct {
		BasicChildObjectVersion
		Char   int8
		Long64 int64
	}

	var object1 BasicChild2ObjectVersion

	object1.Bool = false
	object1.Short = 999
	object1.Ulong = 0xDDDD
	object1.Char = 0
	object1.Long64 = 0xF123456

	{
		data, err := Marshal(&object1)
		if err != nil {
			t.Fatal(err)
		}

		var object2 BasicObjectVersion
		err = Unmarshal(data, &object2)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, uint32(0xDDDD), object2.Ulong)
		assert.Equal(t, false, object2.Bool)
	}

	{
		// TODO unknown object doest not support
	}
}

func TestBasicNestedObject(t *testing.T) {
	type BasicNestedObject struct {
		Char1            int8
		Short1           int16
		BasicObject      BasicObject
		BasicObjectArray []BasicObjectVersion
	}

	var parent BasicNestedObject

	parent.Char1 = 'z'
	parent.Short1 = -10324

	parent.BasicObject.Short1 = 1000
	parent.BasicObject.Ushort1 = 10454
	parent.BasicObject.Bool1 = false
	parent.BasicObject.Uchar1 = 0x0b
	parent.BasicObject.Char1 = 'v'
	parent.BasicObject.Ulong64_1 = 0xFFFFFFFFFF
	parent.BasicObject.Long64_1 = 0x0FFFFFFFFFFFF
	parent.BasicObject.Double = -9.343

	parent.BasicObject.Ulong64ArraySize = 100
	parent.BasicObject.Ulong64Array = make([]uint64, parent.BasicObject.Ulong64ArraySize)

	for i := uint32(0); i < parent.BasicObject.Ulong64ArraySize; i++ {
		parent.BasicObject.Ulong64Array[i] = 0xFFFFFFFFF - uint64(i)*13
	}

	for i := 0; i < 10; i++ {
		var object BasicObjectVersion
		object.Ulong = uint32(i)
		object.Bool = ((i & 1) == 1)
		parent.BasicObjectArray = append(parent.BasicObjectArray, object)
	}

	parent.BasicObject.String = "striiiing"

	// {14E4F405-BA48-4B51-8084-0B6C5523F29E}
	parent.BasicObject.Guid = GUID{0x14e4f405, 0xba48, 0x4b51, [8]byte{0x80, 0x84, 0xb, 0x6c, 0x55, 0x23, 0xf2, 0x9e}}

	{
		data, err := Marshal(&parent)
		if err != nil {
			t.Fatal(err)
		}

		var parent2 BasicNestedObject
		err = Unmarshal(data, &parent2)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(parent, parent2) {
			t.Fatal("not equal")
		}
	}
}

func TestBasicObjectWithPointers(t *testing.T) {
	type BasicObjectWithPointers struct {
		BasicObject1 *BasicObject
		BasicObject2 *BasicObject
	}

	var parent BasicObjectWithPointers

	parent.BasicObject1 = &BasicObject{}
	parent.BasicObject1.Short1 = 1000
	parent.BasicObject1.Ushort1 = 10454
	parent.BasicObject1.Bool1 = false
	parent.BasicObject1.Uchar1 = 0x0b
	parent.BasicObject1.Char1 = 'v'
	parent.BasicObject1.Ulong64_1 = 0xFFFFFFFFFF
	parent.BasicObject1.Long64_1 = 0x0FFFFFFFFFFFF
	parent.BasicObject1.Double = -9.343

	// {14E4F405-BA48-4B51-8084-0B6C5523F29E}
	parent.BasicObject1.Guid = GUID{0x14e4f405, 0xba48, 0x4b51, [8]byte{0x80, 0x84, 0xb, 0x6c, 0x55, 0x23, 0xf2, 0x9e}}

	{
		data, err := Marshal(&parent)
		if err != nil {
			t.Fatal(err)
		}

		var parent2 BasicObjectWithPointers
		err = Unmarshal(data, &parent2)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(parent, parent2) {
			t.Fatal("not equal")
		}
	}
}
