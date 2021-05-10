package serialization

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/go-ole/go-ole"
)

type GUID ole.GUID

var _ customMarshaler = (*GUID)(nil)

func (g *GUID) Marshal(s *encodeState) error {
	if g.IsEmpty() {
		return s.writeTypeMeta(FabricSerializationTypeGuid | FabricSerializationTypeEmptyValueBit)
	}

	if err := s.writeTypeMeta(FabricSerializationTypeGuid); err != nil {
		return err
	}

	return binary.Write(s.buf, binary.LittleEndian, g)
}

func (g *GUID) Unmarshal(meta FabricSerializationType, s *decodeState) error {

	if !isBaseMeta(meta, FabricSerializationTypeGuid) {
		return fmt.Errorf("expect guid get %v", meta)
	}

	return binary.Read(s.inner, binary.LittleEndian, g)
}

func (g *GUID) IsEmpty() bool {
	return ole.IsEqualGUID((*ole.GUID)(g), ole.IID_NULL)
}

// https://github.com/microsoft/go-winio/blob/3fe4fa31662f/pkg/guid/guid.go
// NewV4 returns a new version 4 (pseudorandom) GUID, as defined by RFC 4122.
func NewGuidV4() (*GUID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}

	var g GUID
	g.Data1 = binary.LittleEndian.Uint32(b[0:4])
	g.Data2 = binary.LittleEndian.Uint16(b[4:6])
	g.Data3 = binary.LittleEndian.Uint16(b[6:8])
	copy(g.Data4[:], b[8:16])

	g.Data3 = (g.Data3 & 0x0fff) | 0x4000   // Version 4 (randomly generated)
	g.Data4[0] = (g.Data4[0] & 0x3f) | 0x80 // RFC4122 variant
	return &g, nil
}

func MustNewGuidV4() *GUID {
	g, err := NewGuidV4()
	if err != nil {
		panic(err)
	}

	return g
}
