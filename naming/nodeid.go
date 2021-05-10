package naming

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

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

// NodeIDFromHex convert string in 16 base from to NodeID
func NodeIDFromHex(v string) (NodeID, error) {
	u := NodeID{}

	i, ok := new(big.Int).SetString(v, 16)

	if !ok {
		return u, fmt.Errorf("fail to convert %v to Uint128", v)
	}

	u.Lo = i.Uint64()
	u.Hi = new(big.Int).Rsh(i, 64).Uint64()

	return u, nil
}

// NodeIDFromMD5 hash any string into a NodeID using MD5
func NodeIDFromMD5(v string) NodeID {
	h := md5.Sum([]byte(v))

	return NodeID{
		binary.LittleEndian.Uint64(h[:8]),
		binary.LittleEndian.Uint64(h[8:]),
	}
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
