package jellyfish_merkle

import "errors"

const RootNibbleHeight = HashValueLength * 2

type Nibble struct {
	b byte
}

type NibblePath struct {
	numNibbles uint
	bytes      [HashValueLength]byte
}

func newEvenNibblePath(bytes []byte) (*NibblePath, error) {
	length := len(bytes)
	if length > RootNibbleHeight/2 {
		return nil, errors.New("invalid bytes len")
	}
	bytesArray := *(*[HashValueLength]byte)(bytes)
	return &NibblePath{numNibbles: uint(length * 2), bytes: bytesArray}, nil
}

func newOddNibblePath(bytes []byte) (*NibblePath, error) {
	length := len(bytes)
	if length > RootNibbleHeight/2 {
		return nil, errors.New("invalid bytes len")
	}

	if bytes[length-1]&0x0f != 0 {
		return nil, errors.New("last nibble must be 0")
	}

	bytesArray := *(*[HashValueLength]byte)(bytes)
	return &NibblePath{numNibbles: uint(length*2 - 1), bytes: bytesArray}, nil
}
