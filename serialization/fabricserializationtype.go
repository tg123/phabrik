package serialization

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
