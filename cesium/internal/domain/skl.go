package domain

import (
	"github.com/synnaxlabs/x/telem"
	"math"
	"math/rand"
)

// adapted from https://github.com/cloudcentricdev/golang-tutorials/blob/main/03/skiplist/skiplist.go

const (
	MaxHeight = 16
	PValue    = 0.5 // p = 1/2
)

var probabilities [MaxHeight]uint32

type node struct {
	ptr  pointer
	next [MaxHeight]*node
	prev [MaxHeight]*node
}

type SkipList struct {
	head   *node
	tail   *node
	height int
}

func NewSkipList() *SkipList {
	skl := &SkipList{}
	// Both head and tail are nodes, however, they do not carry a value and are only
	// sentinels.
	skl.head = &node{}
	skl.tail = &node{}
	for level := 0; level < MaxHeight; level++ {
		skl.head.next[level] = skl.tail
		skl.tail.prev[level] = skl.head
	}

	skl.height = 1
	return skl
}

func init() {
	probability := 1.0

	for level := 0; level < MaxHeight; level++ {
		probabilities[level] = uint32(probability * float64(math.MaxUint32))
		probability *= PValue
	}
}

func randomHeight() int {
	seed := rand.Uint32()

	height := 1
	for height < MaxHeight && seed <= probabilities[height] {
		height++
	}

	return height
}

// search returns a pointer indicating the largest node no larger than the requested
// value along with a list of pointers indicating the largest node no larger than the
// requested value at each level.
// While the first return is simply the first element of the second, it's much more
// convenient to work with.
//
// If the requested node is smaller than all nodes in the skiplist, skl.head is returned.
// If the requested node is larger than all nodes in the skiplist, the last element in
// the skiplist is returned.
func (skl *SkipList) search(tr telem.TimeRange) (prev *node, journey [MaxHeight]*node) {
	prev = skl.head
	var next *node

	for level := skl.height - 1; level >= 0; level-- {
		for next = prev.next[level]; next != skl.tail; next = prev.next[level] {
			if next == nil {
				panic("skl reached nil pointer before tail")
			}

			if next.ptr.Start.After(tr.Start) {
				// Move down a level if next value in current level is larger than v.
				break
			}
			prev = next
		}
		journey[level] = prev
	}

	return journey[0], journey
}

func (skl *SkipList) GetLE(tr telem.TimeRange) (pointer, bool) {
	n, _ := skl.search(tr)
	if n == skl.head {
		return pointer{}, false
	}
	return n.ptr, n.ptr.OverlapsWith(tr)
}

func (skl *SkipList) Insert(ptr pointer) {
	_, journey := skl.search(ptr.TimeRange)

	height := randomHeight()
	newNode := &node{ptr: ptr}

	for level := 0; level < height; level++ {
		prev := journey[level]
		if prev == nil {
			// If prev is nil, this means the level is not yet unlocked in the skiplist.
			prev = skl.head
		}
		next := prev.next[level]

		newNode.next[level] = next
		newNode.prev[level] = prev

		next.prev[level] = newNode
		prev.next[level] = newNode
	}

	if height > skl.height {
		skl.height = height
	}
}

func (skl *SkipList) shrink() {
	for level := skl.height - 1; level >= 0; level-- {
		if skl.head.next[level] == skl.tail {
			skl.height--
		}
	}
}
