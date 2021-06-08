package lease

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"syscall"
	"time"
	"unicode/utf16"
)

const (
	maxDuration = time.Duration(math.MaxInt32 - 1)
)

type leaseMessageHeader struct {
	MajorVersion                 uint8
	MinorVersion                 uint8
	_                            uint8
	_                            uint8
	MessageHeaderSize            uint32
	MessageSize                  uint32
	_                            uint32
	MessageIdentifier            int64
	MessageType                  LeaseMessageType
	_                            int32
	LeaseInstance                int64
	RemoteLeaseAgentInstance     int64
	Duration                     int32
	_                            int32
	Expiration                   int64
	LeaseSuspendDuration         int32
	ArbitrationDuration          int32
	IsTwoWayTermination          bool
	_                            uint8
	_                            uint8
	_                            uint8
	SubjectEstablishPendingList  listDesc
	SubjectFailedPendingList     listDesc
	MonitorFailedPendingList     listDesc
	SubjectPendingAcceptedList   listDesc
	SubjectFailedAcceptedList    listDesc
	MonitorFailedAcceptedList    listDesc
	SubjectPendingRejectedList   listDesc
	SubjectTerminatePendingList  listDesc
	SubjectTerminateAcceptedList listDesc
	MessageListenEndpoint        listDesc
	_                            uint32
	// LeaseListenEndPoint          leaseMessageBodyList
}

type leaseMessageExt struct {
	MsgLeaseAgentInstance uint64
}

var sizeofLeaseMessageHeader = uint32(binary.Size(leaseMessageHeader{}))

type listDesc struct {
	Count       uint32
	StartOffset uint32
	Size        uint32
}

type relationshipIdentifier struct {
	Local  string
	Remote string
}

type transportListenEndpoint struct {
	Address     string
	ResolveType uint16
	Port        uint16
}

// cannot call binary.Size
func marshalWithSize(w io.Writer, d interface{}) (uint32, error) {
	var b []byte

	switch d := d.(type) {
	case []byte:
		b = d
	default:
		var buf bytes.Buffer
		if err := binary.Write(&buf, binary.LittleEndian, d); err != nil {
			return 0, err
		}

		b = buf.Bytes()
	}

	size := uint32(len(b))
	if err := binary.Write(w, binary.LittleEndian, size); err != nil {
		return 0, err
	}

	if err := binary.Write(w, binary.LittleEndian, b); err != nil {
		return 0, err
	}

	return size + uint32(binary.Size(uint32(1))), nil
}

func marshalRelationshipList(w io.Writer, l []relationshipIdentifier) (uint32, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(l))); err != nil {
		return 0, err
	}

	for _, r := range l {
		if _, err := marshalWithSize(&buf, utf16.Encode([]rune(r.Local))); err != nil {
			return 0, err
		}

		if _, err := marshalWithSize(&buf, utf16.Encode([]rune(r.Remote))); err != nil {
			return 0, err
		}
	}

	return marshalWithSize(w, buf.Bytes())
}

func dataAtList(data []byte, desc *listDesc) ([]byte, error) {
	st := desc.StartOffset
	ed := desc.StartOffset + desc.Size

	if st >= uint32(len(data)) {
		return nil, fmt.Errorf("bad data start offset")
	}

	if ed >= uint32(len(data)) {
		return nil, fmt.Errorf("bad data start offset")
	}

	return data[st:ed], nil
}

const (
	sizeofUint16        = 2
	sizeofAddressFamily = 2
	sizeofUShort        = 2
)

func unmarshal(data []byte) (*Message, error) {
	header := leaseMessageHeader{}

	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	message := Message{}

	message.Type = header.MessageType
	message.Identifier = header.MessageIdentifier
	message.LeaseInstance = header.LeaseInstance
	message.RemoteLeaseAgentInstance = header.RemoteLeaseAgentInstance
	message.Duration = time.Duration(header.Duration) * time.Millisecond
	message.Expiration = time.Duration(header.Expiration) * time.Millisecond
	message.LeaseSuspendDuration = time.Duration(header.LeaseSuspendDuration) * time.Millisecond
	message.ArbitrationDuration = time.Duration(header.ArbitrationDuration) * time.Millisecond
	message.IsTwoWayTermination = header.IsTwoWayTermination

	// TODO body list
	{
		d, err := dataAtList(data, &header.MessageListenEndpoint)
		if err != nil {
			return nil, err
		}

		r := bytes.NewReader(d)
		s := make([]uint16, (header.MessageListenEndpoint.Size-sizeofAddressFamily-sizeofUShort)/sizeofUint16)
		if err := binary.Read(r, binary.LittleEndian, s); err != nil {
			return nil, err
		}

		if s[len(s)-1] == 0 {
			s = s[:len(s)-1]
		}

		// skip address family
		if _, err := r.Seek(sizeofAddressFamily, io.SeekCurrent); err != nil {
			return nil, err
		}

		var port uint16
		if err := binary.Read(r, binary.LittleEndian, &port); err != nil {
			return nil, err
		}

		message.MessageListenEndpoint = net.JoinHostPort(string(utf16.Decode(s)), strconv.Itoa(int(port)))
	}

	return &message, nil
}

type marshalContext struct {
	AppId   string
	Address string
	Port    uint16
}

func (m *marshalContext) marshalLeaseBody(header *leaseMessageHeader, body *messageBody) ([]byte, error) {

	type param struct {
		listDesc *listDesc
		list     []string
	}

	var buf bytes.Buffer
	offset := uint32(binary.Size(header))

	for _, d := range []param{
		{&header.SubjectEstablishPendingList, body.SubjectEstablishPendingList},
		{&header.SubjectFailedPendingList, body.SubjectFailedPendingList},
		{&header.MonitorFailedPendingList, body.MonitorFailedPendingList},
		{&header.SubjectPendingAcceptedList, body.SubjectPendingAcceptedList},
		{&header.SubjectPendingRejectedList, body.SubjectPendingRejectedList},
		{&header.SubjectFailedAcceptedList, body.SubjectFailedAcceptedList},
		{&header.MonitorFailedAcceptedList, body.MonitorFailedAcceptedList},
		{&header.SubjectTerminatePendingList, body.SubjectTerminatePendingList},
		{&header.SubjectTerminateAcceptedList, body.SubjectTerminateAcceptedList},
	} {
		var l []relationshipIdentifier

		for _, r := range d.list {
			l = append(l, relationshipIdentifier{m.AppId, r})
		}

		size, err := marshalRelationshipList(&buf, l)
		if err != nil {
			return nil, err
		}

		d.listDesc.Count = uint32(len(l))
		d.listDesc.Size = size
		d.listDesc.StartOffset = offset

		offset += size
	}

	{
		header.MessageListenEndpoint.StartOffset = offset
		ListenEndpoint := m.Address
		s := []rune(ListenEndpoint)
		s = append(s, 0)
		ss := utf16.Encode(s)

		size := uint32(len(ss) * sizeofUint16)

		if err := binary.Write(&buf, binary.LittleEndian, ss); err != nil {
			return nil, err
		}

		if err := binary.Write(&buf, binary.LittleEndian, uint16(syscall.AF_INET)); err != nil {
			return nil, err
		}

		if err := binary.Write(&buf, binary.LittleEndian, m.Port); err != nil {
			return nil, err
		}

		header.MessageListenEndpoint.Size = size + sizeofAddressFamily + sizeofUShort
	}

	return buf.Bytes(), nil
}

func (m *marshalContext) marshal(message *Message) ([]byte, error) {

	header := &leaseMessageHeader{}
	header.MajorVersion = 2
	header.MinorVersion = 1
	header.MessageHeaderSize = uint32(binary.Size(header))
	header.MessageType = message.Type
	header.Duration = int32(message.Duration.Milliseconds())
	header.Expiration = message.Expiration.Milliseconds()
	header.LeaseSuspendDuration = int32(message.LeaseSuspendDuration.Milliseconds())
	header.ArbitrationDuration = int32(message.ArbitrationDuration.Milliseconds())
	header.IsTwoWayTermination = message.IsTwoWayTermination

	header.LeaseInstance = message.LeaseInstance
	header.RemoteLeaseAgentInstance = message.RemoteLeaseAgentInstance

	header.MessageIdentifier = uniqId()

	body, err := m.marshalLeaseBody(header, &message.messageBody)
	if err != nil {
		return nil, err
	}

	header.MessageSize = header.MessageHeaderSize + uint32(len(body))

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	if err := binary.Write(&buf, binary.LittleEndian, body); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
