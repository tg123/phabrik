package serialization

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// copy from https://github.com/microsoft/go-winio/blob/master/pkg/guid/guid.go
// but replace `type GUID windows.GUID` to local defined in order to build on all platform

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var emptyGUID = GUID{}

func (g GUID) String() string {
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		g.Data1,
		g.Data2,
		g.Data3,
		g.Data4[:2],
		g.Data4[2:])
}

func (g GUID) IsEmpty() bool {
	return g == emptyGUID
}

// https://github.com/microsoft/go-winio/blob/3fe4fa31662f/pkg/guid/guid.go
// NewV4 returns a new version 4 (pseudorandom) GUID, as defined by RFC 4122.
func NewGuidV4() (GUID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return GUID{}, err
	}

	var g GUID
	g.Data1 = binary.LittleEndian.Uint32(b[0:4])
	g.Data2 = binary.LittleEndian.Uint16(b[4:6])
	g.Data3 = binary.LittleEndian.Uint16(b[6:8])
	copy(g.Data4[:], b[8:16])

	g.Data3 = (g.Data3 & 0x0fff) | 0x4000   // Version 4 (randomly generated)
	g.Data4[0] = (g.Data4[0] & 0x3f) | 0x80 // RFC4122 variant
	return g, nil
}

func MustNewGuidV4() GUID {
	g, err := NewGuidV4()
	if err != nil {
		panic(err)
	}

	return g
}

var _ CustomMarshaler = (*GUID)(nil)

func (g *GUID) Marshal(s Encoder) error {
	if g.IsEmpty() {
		return s.WriteTypeMeta(FabricSerializationTypeGuid | FabricSerializationTypeEmptyValueBit)
	}

	if err := s.WriteTypeMeta(FabricSerializationTypeGuid); err != nil {
		return err
	}

	return s.WriteBinary(g)
}

func (g *GUID) Unmarshal(meta FabricSerializationType, s Decoder) error {

	if !IsBaseMeta(meta, FabricSerializationTypeGuid) {
		return fmt.Errorf("expect guid get %v", meta)
	}

	return s.ReadBinary(g)
}
