package federation

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf16"
)

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

const (
	pearsonPrefix = "UTzJ"
	pearsonSuffix = "X3if"
)

var pearsonT = []uint8{
	1, 87, 49, 12, 176, 178, 102, 166, 121, 193, 6, 84, 249, 230, 44, 163,
	14, 197, 213, 181, 161, 85, 218, 80, 64, 239, 24, 226, 236, 142, 38, 200,
	110, 177, 104, 103, 141, 253, 255, 50, 77, 101, 81, 18, 45, 96, 31, 222,
	25, 107, 190, 70, 86, 237, 240, 34, 72, 242, 20, 214, 244, 227, 149, 235,
	97, 234, 57, 22, 60, 250, 82, 175, 208, 5, 127, 199, 111, 62, 135, 248,
	174, 169, 211, 58, 66, 154, 106, 195, 245, 171, 17, 187, 182, 179, 0, 243,
	132, 56, 148, 75, 128, 133, 158, 100, 130, 126, 91, 13, 153, 246, 216, 219,
	119, 68, 223, 78, 83, 88, 201, 99, 122, 11, 92, 32, 136, 114, 52, 10,
	138, 30, 48, 183, 156, 35, 61, 26, 143, 74, 251, 94, 129, 162, 63, 152,
	170, 7, 115, 167, 241, 206, 3, 150, 55, 59, 151, 220, 90, 53, 23, 131,
	125, 173, 15, 238, 79, 95, 89, 16, 105, 137, 225, 224, 217, 160, 37, 123,
	118, 73, 2, 157, 46, 116, 9, 145, 134, 228, 207, 212, 202, 215, 69, 229,
	27, 188, 67, 124, 168, 252, 42, 4, 29, 108, 21, 247, 19, 205, 39, 203,
	233, 40, 186, 147, 198, 192, 155, 33, 164, 191, 98, 204, 165, 180, 117, 76,
	140, 36, 210, 172, 41, 54, 159, 8, 185, 232, 113, 196, 231, 47, 146, 120,
	51, 65, 28, 144, 254, 221, 93, 189, 194, 139, 112, 43, 71, 109, 184, 209,
}

// this is not a standard pearson hash
func pearsonHash(v string) []byte {
	hash := make([]byte, 16)
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, utf16.Encode([]rune(pearsonPrefix+v+pearsonSuffix)))

	paddedInput := buf.Bytes()

	for j := byte(0); j < 16; j++ {
		h := pearsonT[(paddedInput[0]+j)&0xFF]

		for i := 1; i < len(paddedInput); i++ {
			h = pearsonT[h^paddedInput[i]]
		}
		hash[j] = h
	}

	return hash
}

func NodeIDFromV4Generator(v string) (nodeId NodeID) {

	binary.Read(bytes.NewBuffer(pearsonHash(v)), binary.LittleEndian, &nodeId)
	// save hi and lo for later use
	hi := nodeId.Hi
	low := nodeId.Lo

	// recalculate hi base on index in name
	idx := strings.LastIndexFunc(v, func(r rune) bool { return r == '.' || r == '_' })

	roleName := v[:idx]
	instance_s := v[idx+1:]

	instance_i, err := strconv.ParseUint(instance_s, 10, 64)
	if err != nil {
		return
	}

	binary.Read(bytes.NewBuffer(pearsonHash(roleName)), binary.LittleEndian, &nodeId)

	offset := nodeId.Lo & 0xffffff
	instance_x := (offset + instance_i) & 0xffffff
	instance_y := (instance_x * 14938617) & 0xffffff

	// recalculate hi
	nodeId.Hi = (hi & 0x000000FFFFFFFFFF) | (instance_y << 40)
	nodeId.Lo = low

	return
}
