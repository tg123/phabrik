package serialization

import (
	"fmt"
)

const (
	valueCompressMask7Bit     = 0x7F /*0b 0111 1111 */
	valueCompressMaskMoredata = 0x80 /*0b 1000 0000 */
	valueCompressMaskNegative = 0x40 /*0b 0100 0000 */
)

// TODO find a way to merge signed and unsigned

func (s *decodeState) readCompressedSigned(size int) (int64, error) {
	var value int64

	byteValue, err := s.inner.ReadByte()
	if err != nil {
		return 0, err
	}

	if (byteValue & valueCompressMaskNegative) != 0 {
		value = ^value
	}

	maxSize := ((size*8 + 6) / 7)

	for readSize := 1; readSize <= maxSize; readSize++ {
		b := byteValue & valueCompressMask7Bit

		value <<= 7
		value |= int64(b)

		if byteValue&valueCompressMaskMoredata == 0 {
			break
		}

		if readSize == maxSize {
			return 0, fmt.Errorf("format err 0")
		}

		byteValue, err = s.inner.ReadByte()
		if err != nil {
			return 0, err
		}
	}

	return value, nil
}

func (s *decodeState) readCompressedUnsigned(size int) (uint64, error) {
	var value uint64

	byteValue, err := s.inner.ReadByte()
	if err != nil {
		return 0, err
	}

	maxSize := ((size*8 + 6) / 7)

	for readSize := 1; readSize <= maxSize; readSize++ {
		b := byteValue & valueCompressMask7Bit

		value <<= 7
		value |= uint64(b)

		if byteValue&valueCompressMaskMoredata == 0 {
			break
		}

		if readSize == maxSize {
			return 0, fmt.Errorf("format err 0")
		}

		byteValue, err = s.inner.ReadByte()
		if err != nil {
			return 0, err
		}
	}

	return value, nil
}

func (s *encodeState) writeCompressedSigned(size int, value int64) error {
	if value == 0 {
		return nil
	}

	size = ((size*8 + 6) / 7)
	buffer := make([]byte, size)
	index := size - 1

	var target int64
	if value < 0 {
		target = ^target
	}

	signBit := byte(target & valueCompressMaskNegative)
	temp := value
	isFirst := true

	for {
		b := byte(valueCompressMask7Bit & temp) // grab
		if !isFirst {
			b |= valueCompressMaskMoredata
		}

		buffer[index] = b
		isFirst = false
		index--

		temp >>= 7

		//done for unsigned value, or unsigned value if sign bit has been compressed.
		if (temp == target) && ((b & valueCompressMaskNegative) == signBit) {
			break
		}
	}

	_, err := s.buf.Write(buffer[index+1:])
	return err
}

func (s *encodeState) writeCompressedUnsigned(size int, value uint64) error {
	if value == 0 {
		return nil
	}

	size = ((size*8 + 6) / 7)
	buffer := make([]byte, size)
	index := size - 1

	var target uint64

	temp := value
	isFirst := true

	for {
		b := byte(valueCompressMask7Bit & temp) // grab
		if !isFirst {
			b |= valueCompressMaskMoredata
		}

		buffer[index] = b

		isFirst = false
		index--

		temp >>= 7

		//done for unsigned value, or unsigned value if sign bit has been compressed.
		if temp == target {
			break
		}
	}

	_, err := s.buf.Write(buffer[index+1:])
	return err
}
