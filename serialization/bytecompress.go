package serialization

import (
	"fmt"
)

const (
	valueCompressMask7Bit     = 0x7F /*0b 0111 1111 */
	valueCompressMaskMoredata = 0x80 /*0b 1000 0000 */
	valueCompressMaskNegative = 0x40 /*0b 0100 0000 */
)

func readCompressed[T int64 | uint64](s *decodeState, size int) (T, error) {
	var value T

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
		value |= T(b)

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

func writeCompressed[T int64 | uint64](s *encodeState, size int, value T) error {
	if value == 0 {
		return nil
	}

	size = ((size*8 + 6) / 7)
	buffer := make([]byte, size)
	index := size - 1

	var target T
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

func (s *decodeState) readCompressedSigned(size int) (int64, error) {
	return readCompressed[int64](s, size)
}

func (s *decodeState) readCompressedUnsigned(size int) (uint64, error) {
	return readCompressed[uint64](s, size)
}

func (s *encodeState) writeCompressedSigned(size int, value int64) error {
	return writeCompressed(s, size, value)
}

func (s *encodeState) writeCompressedUnsigned(size int, value uint64) error {
	return writeCompressed(s, size, value)
}
