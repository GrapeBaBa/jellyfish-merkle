package jellyfish_merkle

import (
	"errors"
	"fmt"
	"math"

	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

const PreGenesisVersion = math.MaxUint64

type frozenTreeCache[K Key] struct {
	nodeCache           NodeBatch[K]
	staleNodeIndexCache StaleNodeIndexBatch
	nodeStats           []NodeStats
	rootHashes          []HashValue
}

func newFrozenTreeCache[K Key]() *frozenTreeCache[K] {
	nodeCache := make(map[NodeKey]Node[K])
	staleNodeIndexCache := make(map[StaleNodeIndex]byte)
	return &frozenTreeCache[K]{
		nodeCache:           NodeBatch[K](nodeCache),
		staleNodeIndexCache: StaleNodeIndexBatch(staleNodeIndexCache),
		nodeStats:           make([]NodeStats, 0),
		rootHashes:          make([]HashValue, 0),
	}
}

type TreeCache[K Key, R TreeReader[K]] struct {
	rootNodeKey         NodeKey
	nextVersion         Version
	nodeCache           map[NodeKey]Node[K]
	numNewLeaves        uint
	staleNodeIndexCache map[NodeKey]byte
	numStaleLeaves      uint
	frozenTreeCache     *frozenTreeCache[K]
	reader              R
}

func NewTreeCache[K Key, R TreeReader[K]](reader R, nextVersion Version, persistedVersion *Version) (*TreeCache[K, R], error) {
	nodeCache := make(map[NodeKey]Node[K])
	staleNodeIndexCache := make(map[NodeKey]byte)
	var rootNodeKey *NodeKey
	var err error
	if persistedVersion != nil {
		version := *persistedVersion
		if version == PreGenesisVersion {
			if nextVersion == PreGenesisVersion {
				return nil, errors.New("invalid nextVersion")
			}
		} else {
			if nextVersion <= version {
				return nil, errors.New("invalid nextVersion")
			}
		}
		rootNodeKey, err = NewEmptyPathNodeKey(version)
		if err != nil {
			return nil, fmt.Errorf("new TreeCache failed caused by new NodeKey error %v", err)
		}
	} else {
		rootNodeKey, err = NewEmptyPathNodeKey(0)
		if err != nil {
			return nil, fmt.Errorf("new TreeCache failed caused by new NodeKey error %v", err)
		}
		nodeCache[*rootNodeKey] = &NullNode{}
	}

	return &TreeCache[K, R]{
		rootNodeKey:         *rootNodeKey,
		nodeCache:           nodeCache,
		nextVersion:         nextVersion,
		numNewLeaves:        0,
		staleNodeIndexCache: staleNodeIndexCache,
		numStaleLeaves:      0,
		frozenTreeCache:     newFrozenTreeCache[K](),
		reader:              reader,
	}, nil

}

func (tc *TreeCache[K, R]) GetNode(nodeKey *NodeKey) (Node[K], error) {
	v, ok := tc.nodeCache[*nodeKey]
	if ok {
		return v, nil
	}

	v, ok = tc.frozenTreeCache.nodeCache[*nodeKey]
	if ok {
		return v, nil
	}

	v, err := tc.reader.GetNode(*nodeKey)
	if err != nil {
		return nil, fmt.Errorf("get node failed caused by error %v", err)
	}

	return v, nil
}

func (tc *TreeCache[K, R]) GetRootNodeKey() *NodeKey {
	return &tc.rootNodeKey
}

func (tc *TreeCache[K, R]) SetRootNodeKey(rootNodeKey NodeKey) {
	tc.rootNodeKey = rootNodeKey
}

func (tc *TreeCache[K, R]) PutNode(nodeKey NodeKey, newNode Node[K]) error {
	_, ok := tc.nodeCache[nodeKey]
	if ok {
		return fmt.Errorf("node with key %v already exists in NodeBatch", nodeKey)
	}

	if newNode.IsLeave() {
		tc.numNewLeaves++
	}
	tc.nodeCache[nodeKey] = newNode
	return nil
}

func (tc *TreeCache[K, R]) RemoveNode(oldNodeKey *NodeKey, isLeaf bool) {
	_, ok := tc.nodeCache[*oldNodeKey]
	if !ok {
		_, ok = tc.staleNodeIndexCache[*oldNodeKey]
		if ok {
			panic("node gets stale twice unexpectedly.")
		} else {
			tc.staleNodeIndexCache[*oldNodeKey] = byte('0')
		}
		if isLeaf {
			tc.numStaleLeaves++
		}
	} else {
		delete(tc.nodeCache, *oldNodeKey)
		tc.numNewLeaves--
	}
}

func (tc *TreeCache[K, R]) Freeze() {
	rootNodeKey := tc.GetRootNodeKey()
	rootNode, err := tc.GetNode(rootNodeKey)
	if err != nil {
		panic(fmt.Sprintf("root node with key %v must exist", rootNodeKey))
	}
	rootHash := rootNode.Hash()
	tc.frozenTreeCache.rootHashes = append(tc.frozenTreeCache.rootHashes, rootHash)
	nodeStats := NodeStats{
		NewNodes:    uint(len(tc.nodeCache)),
		NewLeaves:   tc.numNewLeaves,
		StaleNodes:  uint(len(tc.staleNodeIndexCache)),
		StaleLeaves: tc.numStaleLeaves,
	}
	tc.frozenTreeCache.nodeStats = append(tc.frozenTreeCache.nodeStats, nodeStats)
	maps.Copy(tc.frozenTreeCache.nodeCache, tc.nodeCache)
	staleSinceVersion := tc.nextVersion

	staleNodeIndexBatch := lo.MapKeys[NodeKey, byte, StaleNodeIndex](tc.staleNodeIndexCache, func(_ byte, k NodeKey) StaleNodeIndex {
		return StaleNodeIndex{StaleSinceVersion: staleSinceVersion, NodeKey: k}
	})

	maps.Copy(tc.frozenTreeCache.staleNodeIndexCache, staleNodeIndexBatch)
	tc.numStaleLeaves = 0
	tc.numNewLeaves = 0
	tc.nextVersion++
}

func From[K Key, R TreeReader[K]](treeCache *TreeCache[K, R]) ([]HashValue, *TreeUpdateBatch[K]) {
	return treeCache.frozenTreeCache.rootHashes, &TreeUpdateBatch[K]{
		NodeBatch:           treeCache.frozenTreeCache.nodeCache,
		StaleNodeIndexBatch: treeCache.frozenTreeCache.staleNodeIndexCache,
		NodeStats:           treeCache.frozenTreeCache.nodeStats,
	}
}
