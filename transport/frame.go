package transport

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"

	"github.com/sigurn/crc8"
)

type securityProvider uint8

const (
	securityProviderNone      securityProvider = 0
	securityProviderSsl       securityProvider = 0x1 // Certs
	securityProviderKerberos  securityProvider = 0x2 // Windows auth
	securityProviderNegotiate securityProvider = 0x3 // Windows auth
	securityProviderClaims    securityProvider = 0x4 // Can be for AAD for DSTS
	securityProviderLast      securityProvider = 0x4 // Maximum size for this enum is 3 bits.
)

type frameheader struct {
	FrameLength          uint32
	SecurityProviderMask uint8
	FrameHeaderCRC       uint8
	HeaderLength         uint16
	FrameBodyCRC         uint32
}

var sizeOfFrameheader = binary.Size(frameheader{})

type frameReadConfig struct {
	CheckFrameHeaderCRC bool
	CheckFrameBodyCRC   bool
}

func nextFrame(r io.Reader, config frameReadConfig) (*frameheader, []byte, error) {
	header := frameheader{}
	err := binary.Read(r, binary.LittleEndian, &header)
	if err != nil {
		return nil, nil, err
	}

	var b bytes.Buffer
	err = binary.Write(&b, binary.LittleEndian, &frameheader{
		FrameLength:          header.FrameLength,
		SecurityProviderMask: header.SecurityProviderMask,
		HeaderLength:         header.HeaderLength,
		FrameHeaderCRC:       0,
		FrameBodyCRC:         0,
	})
	if err != nil {
		return nil, nil, err
	}

	if config.CheckFrameHeaderCRC {
		if header.FrameHeaderCRC != crc8.Checksum(b.Bytes(), crc8.MakeTable(crc8.CRC8)) {
			return nil, nil, fmt.Errorf("frame header crc8 check fail")
		}
	}

	body := make([]byte, header.FrameLength-uint32(sizeOfFrameheader))

	_, err = io.ReadFull(r, body)
	if err != nil {
		return nil, nil, err
	}

	if config.CheckFrameBodyCRC {
		if header.FrameBodyCRC != crc32.Checksum(body, crc32.IEEETable) {
			return nil, nil, fmt.Errorf("frame body crc32 check fail")
		}
	}

	return &header, body, nil
}

type frameWriteConfig struct {
	SecurityProviderMask securityProvider
	FrameHeaderCRC       bool
	FrameBodyCRC         bool
}

func writeMessageWithFrame(w io.Writer, message *Message, config frameWriteConfig) error {
	headerLen, msg, err := message.marshal()
	if err != nil {
		return err
	}

	return writeFrame(w, headerLen, msg, config)
}

func writeFrame(w io.Writer, headerLen int, msg []byte, config frameWriteConfig) error {

	tcpheader := &frameheader{
		FrameLength:          uint32(sizeOfFrameheader + len(msg)),
		SecurityProviderMask: uint8(config.SecurityProviderMask),
		FrameHeaderCRC:       0,
		HeaderLength:         uint16(headerLen),
		FrameBodyCRC:         0,
	}

	var b bytes.Buffer
	err := binary.Write(&b, binary.LittleEndian, tcpheader)
	if err != nil {
		return err
	}

	if config.FrameHeaderCRC {
		tcpheader.FrameHeaderCRC = crc8.Checksum(b.Bytes(), crc8.MakeTable(crc8.CRC8))
	}

	if config.FrameBodyCRC {
		tcpheader.FrameBodyCRC = crc32.Checksum(msg, crc32.IEEETable)
	}

	err = binary.Write(w, binary.LittleEndian, tcpheader)
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	return err
}
