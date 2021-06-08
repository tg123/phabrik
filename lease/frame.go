package lease

import (
	"encoding/binary"
	"fmt"
	"io"
)

type ltFrameType uint8

const (
	ltFrametypeMessage ltFrameType = 1
	ltFrametypeConnect ltFrameType = 2
)

type ltFrameHeader struct {
	FrameType ltFrameType
	Reserved1 uint8
	Reserved2 uint16
	FrameSize uint32
	Reserved3 uint32
}

var sizeOfLtFrameheader = binary.Size(ltFrameHeader{})

func nextLtFrame(r io.Reader) ([]byte, error) {
	header := ltFrameHeader{}
	err := binary.Read(r, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	switch header.FrameType {
	case ltFrametypeConnect:
		// ignore connect message
		return nextLtFrame(r)
	case ltFrametypeMessage:
		body := make([]byte, header.FrameSize-uint32(sizeOfLtFrameheader))

		_, err = io.ReadFull(r, body)
		if err != nil {
			return nil, err
		}

		return body, nil
	default:
		return nil, fmt.Errorf("unexpected frame type %v", header.FrameType)
	}
}

func writeConnectFrame(w io.Writer) error {
	var h ltFrameHeader

	h.FrameType = ltFrametypeConnect
	h.FrameSize = uint32(sizeOfLtFrameheader)
	return binary.Write(w, binary.LittleEndian, &h)
}

func writeDataWithFrame(w io.Writer, message []byte) error {
	var h ltFrameHeader

	h.FrameType = ltFrametypeMessage
	h.FrameSize = uint32(sizeOfLtFrameheader) + uint32(len(message))

	err := binary.Write(w, binary.LittleEndian, &h)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, message)
	if err != nil {
		return err
	}

	return nil
}
