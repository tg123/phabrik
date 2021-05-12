package serialization

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type cm struct {
	v int
}

func (c *cm) Marshal(s Encoder) error {
	if err := s.WriteTypeMeta(FabricSerializationTypeChar); err != nil {
		return nil
	}
	return s.WriteBinary([]byte(strconv.Itoa(c.v))[0])
}

func (c *cm) Unmarshal(meta FabricSerializationType, s Decoder) error {
	if meta != FabricSerializationTypeChar {
		return fmt.Errorf("except char got %v", meta)
	}

	var b byte
	if err := s.ReadBinary(&b); err != nil {
		return err
	}

	v, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}

	c.v = v
	return nil
}

func TestCustomMarshaler(t *testing.T) {
	type S struct {
		F cm
	}

	var object S
	object.F.v = 9

	var object2 S
	marshalAndUnmarshal(t, &object, &object2)

	assert.Equal(t, object.F.v, object2.F.v)
}

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

func marshalAndUnmarshal(t *testing.T, from, to interface{}) {
	data, err := Marshal(from)
	if err != nil {
		t.Fatal(err)
	}

	err = Unmarshal(data, to)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMapSerialization(t *testing.T) {
	type mapObj struct {
		Map  map[string]int32
		Map2 map[int32]string
	}

	{
		var object mapObj
		var object2 mapObj
		marshalAndUnmarshal(t, &object, &object2)
		assert.Equal(t, object, object2)
	}

	{
		var object mapObj
		object.Map = map[string]int32{
			"1": 1,
			"2": 2,
			"3": 3,
		}

		object.Map2 = map[int32]string{
			1: "1",
			2: "2",
			3: "3",
		}

		var object2 mapObj
		marshalAndUnmarshal(t, &object, &object2)
		assert.Equal(t, object, object2)
	}
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

	{
		var object2 BasicObject
		marshalAndUnmarshal(t, &object, &object2)
		assert.Equal(t, object, object2)
	}

	{
		// test empty
		var empty BasicObject
		marshalAndUnmarshal(t, &BasicObject{}, &empty)
		assert.Equal(t, empty, BasicObject{})
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
		var object2 BasicObjectChild
		marshalAndUnmarshal(t, &object, &object2)
		assert.Equal(t, object, object2)
	}

	// make sure embedded obj
	{
		var object2 BasicObjectChild
		marshalAndUnmarshal(t, &struct {
			Char1   int8
			Uchar1  uint8
			Short1  int16
			Ushort1 uint16
		}{
			Char1:   42,
			Uchar1:  1,
			Ushort1: 9527,
		}, &object2)

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
	object1.Guid = MustNewGuidV4()

	var object2 BasicObjectVersion

	marshalAndUnmarshal(t, &object1, &object2)

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
		var object2 BasicObjectVersion

		marshalAndUnmarshal(t, &object1, &object2)

		assert.Equal(t, uint32(0xDDDD), object2.Ulong)
		assert.Equal(t, false, object2.Bool)
	}

	{
		var empty BasicChild2ObjectVersion
		marshalAndUnmarshal(t, &BasicChild2ObjectVersion{}, &empty)
		assert.Equal(t, BasicChild2ObjectVersion{}, empty)
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
		var parent2 BasicNestedObject
		marshalAndUnmarshal(t, &parent, &parent2)
		assert.Equal(t, parent, parent2)
	}

	{
		var empty BasicNestedObject
		marshalAndUnmarshal(t, &BasicNestedObject{}, &empty)
		assert.Equal(t, BasicNestedObject{}, empty)
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
		var parent2 BasicObjectWithPointers
		marshalAndUnmarshal(t, &parent, &parent2)

		assert.Equal(t, parent, parent2)
	}

	{
		var empty BasicObjectWithPointers
		marshalAndUnmarshal(t, &BasicObjectWithPointers{}, &empty)

		assert.Equal(t, BasicObjectWithPointers{}, empty)
	}
}

func TestPolymorphicObject(t *testing.T) {
	// TODO activator not supported
}

func TestPolymorphicObjectChild(t *testing.T) {
	// TODO activator not supported
}

type BasicObjectV1 struct {
	Char1 int8
}

type BasicUnknownNestedObject struct {
	Char1   int8
	Ulong64 uint64
}

type BasicObjectV2 struct {
	Char1                 int8
	Ulong64               uint64
	Short1                int16
	Guid                  GUID
	CharArray             []int8
	BasicUnknownNested    BasicUnknownNestedObject
	Long1                 int32
	BasicUnknownNestedPtr *BasicUnknownNestedObject
	Ulong1                uint32
}

func TestBasicObjectVersioningV1ToV2(t *testing.T) {
	var object1 BasicObjectV1
	object1.Char1 = 'F'

	var object2 BasicObjectV2
	marshalAndUnmarshal(t, &object1, &object2)

	assert.Equal(t, object1.Char1, object2.Char1)
	assert.Equal(t, object2.Ulong64, uint64(0))
}

func TestBasicObjectVersioningV2ToV1ToV2(t *testing.T) {
	var object1 BasicObjectV2

	object1.Char1 = 'F'
	object1.Ulong64 = 0xF00D
	object1.Short1 = 0xBAD
	object1.CharArray = []int8{'y', 'e', 's'}

	object1.BasicUnknownNested.Char1 = 'y'
	object1.BasicUnknownNestedPtr = &BasicUnknownNestedObject{}
	object1.BasicUnknownNestedPtr.Char1 = 'r'

	{
		var object2 BasicObjectV2
		marshalAndUnmarshal(t, &object1, &object2)
		assert.Equal(t, object1, object2)
	}

	{
		var empty BasicObjectV2
		marshalAndUnmarshal(t, &BasicObjectV2{}, &empty)
		assert.Equal(t, BasicObjectV2{}, empty)
	}

	{
		var object2 BasicObjectV1
		marshalAndUnmarshal(t, &object1, &object2)

		assert.Equal(t, object1.Char1, object2.Char1)

		object2b := object2
		var object3 BasicObjectV1
		marshalAndUnmarshal(t, &object2b, &object3)
		// unknown object cannot be carried
		// assert.Equal(t, object3, object1)
	}

}

type BasicObjectWithArraysV1 struct {
	Long64 int64
}

type BasicObjectWithArraysV2 struct {
	Long64                         int64
	BoolArray                      []bool
	CharArray                      []int8
	UcharArray                     []uint8
	ShortArray                     []int16
	UshortArray                    []uint16
	LongArray                      []int32
	UlongArray                     []uint32
	Long64Array                    []int64
	Ulong64Array                   []uint64
	GuidArray                      []GUID
	DoubleArray                    []float64
	BasicUnknownNestedObjectArray  []BasicUnknownNestedObject
	BasicUnknownNestedPointerArray []*BasicUnknownNestedObject
}

func TestObjectWithArraysVersioningV1ToV2(t *testing.T) {
	var object1 BasicObjectWithArraysV1
	object1.Long64 = 0x4234

	var object2 BasicObjectWithArraysV2
	marshalAndUnmarshal(t, &object1, &object2)

	assert.Equal(t, object1.Long64, object2.Long64)
}

func TestObjectWithArraysVersioningV2ToV1(t *testing.T) {
	fill := func(array interface{}, len int, v interface{}) {
		rav := reflect.Indirect(reflect.ValueOf(array))
		rvv := reflect.ValueOf(v)

		arr := reflect.MakeSlice(rav.Type(), len, len)

		for i := 0; i < len; i++ {
			arr.Index(i).Set(rvv)
		}

		rav.Set(arr)
	}

	var object1 BasicObjectWithArraysV2
	object1.Long64 = 0xfafafaf

	object1.BoolArray = []bool{true, false}

	fill(&object1.CharArray, 10, int8('a'))
	fill(&object1.UcharArray, 5, uint8(0x20))
	fill(&object1.ShortArray, 15, int16(1))
	fill(&object1.UshortArray, 5, uint16(0x23))
	fill(&object1.LongArray, 53, int32(0x2f0))
	fill(&object1.UlongArray, 12, uint32(0x20))
	fill(&object1.Long64Array, 53, int64(0x2f0))
	fill(&object1.Ulong64Array, 12, uint64(0x20))
	fill(&object1.DoubleArray, 20, float64(-5.33))

	for i := 0; i < 13; i++ {
		object1.GuidArray = append(object1.GuidArray, MustNewGuidV4())
	}

	for i := 0; i < 3; i++ {
		var nested BasicUnknownNestedObject
		nested.Char1 = 'y'
		nested.Ulong64 = uint64(i)

		object1.BasicUnknownNestedObjectArray = append(object1.BasicUnknownNestedObjectArray, nested)
	}

	for i := 0; i < 9; i++ {
		var nested *BasicUnknownNestedObject

		if i%3 == 0 {
			nested = &BasicUnknownNestedObject{}
			nested.Char1 = 'x'
			nested.Ulong64 = uint64(i * 3)
		}

		object1.BasicUnknownNestedPointerArray = append(object1.BasicUnknownNestedPointerArray, nested)
	}

	{
		var empty BasicObjectWithArraysV2
		marshalAndUnmarshal(t, &BasicObjectWithArraysV2{}, &empty)
		assert.Equal(t, BasicObjectWithArraysV2{}, empty)
	}

	{
		var object2 BasicObjectWithArraysV2
		marshalAndUnmarshal(t, &object1, &object2)
		assert.Equal(t, object1, object2)
	}

	{
		var object2 BasicObjectWithArraysV1
		marshalAndUnmarshal(t, &object1, &object2)
		assert.Equal(t, object1.Long64, object2.Long64)
	}
}
