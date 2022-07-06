package jellyfish_merkle

import "errors"

const (
	HashValueLength       = 32
	HashValueLengthInBits = HashValueLength * 8
)

type CryptoHash interface {
	Hash() HashValue
}

type HashValue struct {
	hash [HashValueLength]byte
}

func createLiteralHash(word string) (*HashValue, error) {
	wordBytes := []byte(word)
	length := len(wordBytes)
	if length > HashValueLength {
		return nil, errors.New("word length is too large")
	}

	wordBytes = append(wordBytes, make([]byte, HashValueLength-length)...)
	hashValue := newHashValueFromSlice(wordBytes)

	return &hashValue, nil
}

func newHashValueFromSlice(word []byte) HashValue {
	hash := *(*[HashValueLength]byte)(word)
	return HashValue{hash: hash}
}
