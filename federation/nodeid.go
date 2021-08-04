package federation

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/tg123/phabrik/serialization"
)

type uint128 struct {
	Hi uint64
	Lo uint64
}

var u128zero = uint128{0, 0}
var u128max = uint128{math.MaxUint64, math.MaxUint64}

// NodeID is an unique identifier for each node in a federacy ring
// a NodeID is a 128 bit number, unit128_t
type NodeID uint128

// MinNodeID is the start of NodeID of a federacy ring
var MinNodeID = NodeID{0, 0}

func (n NodeID) String() string {
	return fmt.Sprintf("%016x%016x", n.Hi, n.Lo)
}

// github.com/lukechampine/Uint128/
func (u uint128) cmp(v uint128) int {
	if u == v {
		return 0
	} else if u.Hi < v.Hi || (u.Hi == v.Hi && u.Lo < v.Lo) {
		return -1
	} else {
		return 1
	}
}

type NodeInstance struct {
	Id         NodeID
	InstanceId uint64
}

var sizeOfNodeID = uint32(binary.Size(NodeID{}))
var _ serialization.CustomMarshaler = (*NodeID)(nil)
var nodeIdMetaType = serialization.FabricSerializationTypeUChar | serialization.FabricSerializationTypeArray

func (n *NodeID) Marshal(s serialization.Encoder) error {

	if err := s.WriteTypeMeta(nodeIdMetaType); err != nil {
		return err
	}

	if err := s.WriteCompressedUInt32(sizeOfNodeID); err != nil {
		return err
	}

	return s.WriteBinary(n)
}

func (n *NodeID) Unmarshal(meta serialization.FabricSerializationType, s serialization.Decoder) error {

	if meta != nodeIdMetaType {
		return fmt.Errorf("expect %v got %v", nodeIdMetaType, meta)
	}

	c, err := s.ReadCompressedUInt32()
	if err != nil {
		return err
	}

	if c != sizeOfNodeID {
		return fmt.Errorf("wrong node id size")
	}

	return s.ReadBinary(n)
}
