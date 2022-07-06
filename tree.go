package jellyfish_merkle

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/samber/lo"
)

type Key interface {
	comparable
}

type Value interface {
	CryptoHash
}

type ValueSet[K Key] struct {
	keyHash HashValue
	pair    *Pair[K]
}

type Pair[K Key] struct {
	valueHash HashValue
	key       K
}
type NodeBatch[K Key] map[NodeKey]Node[K]

type StateValueBatch[K Key, V Value] map[ValueIndex[K]]V

type StaleNodeIndexBatch map[StaleNodeIndex]byte

type TreeReader[K Key] interface {
	GetNode(nodeKey NodeKey) (Node[K], error)

	GetRightMostLeaf() (NodeKey, LeafNode[K], error)
}

type TreeWriter[K Key] interface {
	WriteNodeBatch(nodeBatch NodeBatch[K]) error
}

type StateValueWriter[K Key, V Value] interface {
	WriteKVBatch()
}

type StaleNodeIndex struct {
	StaleSinceVersion Version
	NodeKey           NodeKey
}

type NodeStats struct {
	NewNodes    uint
	NewLeaves   uint
	StaleNodes  uint
	StaleLeaves uint
}

type TreeUpdateBatch[K Key] struct {
	NodeBatch           NodeBatch[K]
	StaleNodeIndexBatch StaleNodeIndexBatch
	NodeStats           []NodeStats
}

type NibbleRangeIterator[K Key] struct {
	sortedKey []Pair[K]
	nibbleIdx uint
	pos       uint
}

type JellyfishMerkleTree[K Key, R TreeReader[K]] struct {
	reader R
}

func NewJellyfishMerkleTree[K Key, R TreeReader[K]](reader R) *JellyfishMerkleTree[K, R] {
	return &JellyfishMerkleTree[K, R]{reader: reader}
}

func (jmt *JellyfishMerkleTree[K, R]) getHash(nodeKey *NodeKey, node Node[K], hashCache map[NibblePath]HashValue) HashValue {
	if hashCache == nil {
		return node.Hash()
	} else {
		v, ok := hashCache[*nodeKey.NibblePath()]
		if ok {
			return v
		} else {
			panic(fmt.Sprintf("%v can not be found in hash cache", *nodeKey))
		}
	}
}

func (jmt *JellyfishMerkleTree[K, R]) BatchPutValueSets(valueSets [][]ValueSet[K], nodeHashes []map[NibblePath]HashValue, persistedVersion *Version, firstVersion Version) ([]HashValue, *TreeUpdateBatch[K], error) {
	var err error
	treeCache, err := NewTreeCache[K](jmt.reader, firstVersion, persistedVersion)
	var hastSets []map[NibblePath]HashValue
	if nodeHashes != nil {
		hastSets = nodeHashes
	} else {
		hastSets = make([]map[NibblePath]HashValue, len(valueSets))
	}

	if len(valueSets) != len(hastSets) {
		panic("valueSets and hashSets length must be equal.")
	}

	for idx, item := range lo.Zip2(valueSets, hastSets) {
		valueSet := item.A
		hashSet := item.B
		valueSetLength := len(valueSet)
		if valueSet == nil || valueSetLength < 1 {
			treeCache.Freeze()
			continue
		}

		version := firstVersion + Version(idx)

		sort.SliceStable(valueSet, func(i, j int) bool {
			return bytes.Compare(valueSet[i].keyHash.hash[:], valueSet[j].keyHash.hash[:]) < 0
		})

		prev := 1
		for curr := 1; curr < valueSetLength; curr++ {
			if !bytes.Equal(valueSet[curr-1].keyHash.hash[:], valueSet[curr].keyHash.hash[:]) {
				valueSet[prev] = valueSet[curr]
				prev++
			}
		}

		dedupedAndSortedKVs := valueSet[:prev]
		rootNodeKey := treeCache.GetRootNodeKey()
		var newRootNodeKey *NodeKey
		newRootNodeKey, _, err = jmt.batchInsertAt(rootNodeKey, version, dedupedAndSortedKVs, 0, hashSet, treeCache)
		if err != nil {
			return nil, nil, fmt.Errorf("batch put valueSet failed caused by error %v", err)
		}

		treeCache.SetRootNodeKey(*newRootNodeKey)

		treeCache.Freeze()
	}

	hashValues, treeUpdateBatch := From(treeCache)
	return hashValues, treeUpdateBatch, nil
}

func (jmt *JellyfishMerkleTree[K, R]) batchInsertAt(nodeKey *NodeKey, version Version, kvs []ValueSet[K], depth uint, hashCache map[NibblePath]HashValue, treeCache *TreeCache[K, R]) (*NodeKey, Node[K], error) {
	return nil, nil, nil
}
