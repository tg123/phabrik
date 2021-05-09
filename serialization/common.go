package serialization

import "encoding/binary"

type customMarshaler interface {
	Marshal(*encodeState) error
	Unmarshal(FabricSerializationType, *decodeState) error
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
