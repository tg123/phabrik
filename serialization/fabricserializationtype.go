package serialization

import "reflect"

type FabricSerializationType uint8

const (
	FabricSerializationTypeEmptyValueBit FabricSerializationType = 0x40 // 0b0100 0000 - This bit set means the value is empty
	FabricSerializationTypeArray         FabricSerializationType = 0x80 // 0b1000 0000 - This bit set indicates an array
	FabricSerializationTypeBaseTypeMask  FabricSerializationType = 0x0F // 0b0000 1111
	FabricSerializationTypeBoolFalseFlag FabricSerializationType = 0x30 // 0b0011 0000

	FabricSerializationTypeObject  FabricSerializationType = 0x00
	FabricSerializationTypePointer FabricSerializationType = 0x01

	FabricSerializationTypeBool      FabricSerializationType = 0x02
	FabricSerializationTypeBoolTrue  FabricSerializationType = FabricSerializationTypeBool
	FabricSerializationTypeBoolFalse FabricSerializationType = FabricSerializationTypeBool | FabricSerializationTypeBoolFalseFlag

	FabricSerializationTypeChar  FabricSerializationType = 0x03
	FabricSerializationTypeUChar FabricSerializationType = 0x04

	FabricSerializationTypeShort  FabricSerializationType = 0x05
	FabricSerializationTypeUShort FabricSerializationType = 0x06
	FabricSerializationTypeInt32  FabricSerializationType = 0x07
	FabricSerializationTypeUInt32 FabricSerializationType = 0x08
	FabricSerializationTypeInt64  FabricSerializationType = 0x09
	FabricSerializationTypeUInt64 FabricSerializationType = 0x0A

	FabricSerializationTypeDouble FabricSerializationType = 0x0B
	FabricSerializationTypeGuid   FabricSerializationType = 0x0C

	FabricSerializationTypeWString FabricSerializationType = 0x0D

	FabricSerializationTypeByteArrayNoCopy FabricSerializationType = 0x0E | FabricSerializationTypeArray

	FabricSerializationTypeScopeBegin FabricSerializationType = 0x1F
	FabricSerializationTypeScopeEnd   FabricSerializationType = 0x2F
	FabricSerializationTypeObjectEnd  FabricSerializationType = 0x3F

	FabricSerializationTypeNotAMeta FabricSerializationType = 0xFF
)

func IsEmptyMeta(meta FabricSerializationType) bool {
	return meta&FabricSerializationTypeEmptyValueBit > 0
}

func IsArrayMeta(meta FabricSerializationType) bool {
	return meta&FabricSerializationTypeArray > 0
}

func IsBaseMeta(meta, base FabricSerializationType) bool {
	return (meta & FabricSerializationTypeBaseTypeMask) == base
}

func kindToFabricSerializationType(kind reflect.Kind) FabricSerializationType {
	switch kind {
	case reflect.Uint8:
		return FabricSerializationTypeUChar
	case reflect.Int8:
		return FabricSerializationTypeChar
	case reflect.Uint16:
		return FabricSerializationTypeUShort
	case reflect.Uint32:
		return FabricSerializationTypeUInt32
	case reflect.Uint64:
		return FabricSerializationTypeUInt64
	case reflect.Int16:
		return FabricSerializationTypeShort
	case reflect.Int32:
		return FabricSerializationTypeInt32
	case reflect.Int64:
		return FabricSerializationTypeInt64
	case reflect.Float32, reflect.Float64:
		return FabricSerializationTypeDouble
	case reflect.Bool:
		return FabricSerializationTypeBool
	case reflect.String:
		return FabricSerializationTypeWString
	case reflect.Struct:
		return FabricSerializationTypeObject
	case reflect.Ptr:
		return FabricSerializationTypePointer
	default:
	}

	// not support
	return FabricSerializationTypeNotAMeta
}
