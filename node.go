package jellyfish_merkle

import "fmt"

const (
	Leaf     = NodeKind(0)
	Internal = NodeKind(1)
	Null     = NodeKind(2)
)

type NodeKind byte

type Version uint64

type Node[K Key] interface {
	Kind() NodeKind
	Hash() HashValue
	IsLeave() bool
}

type NullNode struct {
}

func (nn *NullNode) Kind() NodeKind {
	return Null
}

func (nn *NullNode) Hash() HashValue {
	v, _ := createLiteralHash("SPARSE_MERKLE_PLACEHOLDER_HASH")
	return *v
}

func (nn *NullNode) IsLeave() bool {
	return false
}

type Child[K Key] struct {
	Hash     HashValue
	Version  Version
	NodeType Node[K]
}

type ValueIndex[K Key] struct {
	key     K
	version Version
}

type LeafNode[K Key] struct {
	accountKey HashValue
	valueHash  HashValue
	valueIndex ValueIndex[K]
}

type InternalNode[K Key] struct {
	children  Children[K]
	leafCount uint
}

type NodeKey struct {
	version    Version
	nibblePath NibblePath
}

func NewNodeKey(version Version, nibblePath NibblePath) NodeKey {
	return NodeKey{version, nibblePath}
}

func (nk *NodeKey) NibblePath() *NibblePath {
	return &nk.nibblePath
}

func NewEmptyPathNodeKey(version Version) (*NodeKey, error) {
	np, err := newEvenNibblePath(make([]byte, 0))
	if err != nil {
		return nil, fmt.Errorf("new node path failed caused by nibble path error %v", err)
	}
	return &NodeKey{version: version, nibblePath: *np}, nil
}

type Children[K Key] map[Nibble]Child[K]
