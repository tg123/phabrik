// Code generated by "stringer -type FabricSerializationType"; DO NOT EDIT.

package serialization

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[FabricSerializationTypeEmptyValueBit-64]
	_ = x[FabricSerializationTypeArray-128]
	_ = x[FabricSerializationTypeBaseTypeMask-15]
	_ = x[FabricSerializationTypeBoolFalseFlag-48]
	_ = x[FabricSerializationTypeObject-0]
	_ = x[FabricSerializationTypePointer-1]
	_ = x[FabricSerializationTypeBool-2]
	_ = x[FabricSerializationTypeBoolTrue-2]
	_ = x[FabricSerializationTypeBoolFalse-50]
	_ = x[FabricSerializationTypeChar-3]
	_ = x[FabricSerializationTypeUChar-4]
	_ = x[FabricSerializationTypeShort-5]
	_ = x[FabricSerializationTypeUShort-6]
	_ = x[FabricSerializationTypeInt32-7]
	_ = x[FabricSerializationTypeUInt32-8]
	_ = x[FabricSerializationTypeInt64-9]
	_ = x[FabricSerializationTypeUInt64-10]
	_ = x[FabricSerializationTypeDouble-11]
	_ = x[FabricSerializationTypeGuid-12]
	_ = x[FabricSerializationTypeWString-13]
	_ = x[FabricSerializationTypeByteArrayNoCopy-142]
	_ = x[FabricSerializationTypeScopeBegin-31]
	_ = x[FabricSerializationTypeScopeEnd-47]
	_ = x[FabricSerializationTypeObjectEnd-63]
	_ = x[FabricSerializationTypeNotAMeta-255]
}

const (
	_FabricSerializationType_name_0 = "FabricSerializationTypeObjectFabricSerializationTypePointerFabricSerializationTypeBoolFabricSerializationTypeCharFabricSerializationTypeUCharFabricSerializationTypeShortFabricSerializationTypeUShortFabricSerializationTypeInt32FabricSerializationTypeUInt32FabricSerializationTypeInt64FabricSerializationTypeUInt64FabricSerializationTypeDoubleFabricSerializationTypeGuidFabricSerializationTypeWString"
	_FabricSerializationType_name_1 = "FabricSerializationTypeBaseTypeMask"
	_FabricSerializationType_name_2 = "FabricSerializationTypeScopeBegin"
	_FabricSerializationType_name_3 = "FabricSerializationTypeScopeEndFabricSerializationTypeBoolFalseFlag"
	_FabricSerializationType_name_4 = "FabricSerializationTypeBoolFalse"
	_FabricSerializationType_name_5 = "FabricSerializationTypeObjectEndFabricSerializationTypeEmptyValueBit"
	_FabricSerializationType_name_6 = "FabricSerializationTypeArray"
	_FabricSerializationType_name_7 = "FabricSerializationTypeByteArrayNoCopy"
	_FabricSerializationType_name_8 = "FabricSerializationTypeNotAMeta"
)

var (
	_FabricSerializationType_index_0 = [...]uint16{0, 29, 59, 86, 113, 141, 169, 198, 226, 255, 283, 312, 341, 368, 398}
	_FabricSerializationType_index_3 = [...]uint8{0, 31, 67}
	_FabricSerializationType_index_5 = [...]uint8{0, 32, 68}
)

func (i FabricSerializationType) String() string {
	switch {
	case i <= 13:
		return _FabricSerializationType_name_0[_FabricSerializationType_index_0[i]:_FabricSerializationType_index_0[i+1]]
	case i == 15:
		return _FabricSerializationType_name_1
	case i == 31:
		return _FabricSerializationType_name_2
	case 47 <= i && i <= 48:
		i -= 47
		return _FabricSerializationType_name_3[_FabricSerializationType_index_3[i]:_FabricSerializationType_index_3[i+1]]
	case i == 50:
		return _FabricSerializationType_name_4
	case 63 <= i && i <= 64:
		i -= 63
		return _FabricSerializationType_name_5[_FabricSerializationType_index_5[i]:_FabricSerializationType_index_5[i+1]]
	case i == 128:
		return _FabricSerializationType_name_6
	case i == 142:
		return _FabricSerializationType_name_7
	case i == 255:
		return _FabricSerializationType_name_8
	default:
		return "FabricSerializationType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
