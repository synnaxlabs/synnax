package skl

import (
	"math"
	"math/rand"
	"sync"
)

// adapted from https://github.com/cloudcentricdev/golang-tutorials/blob/main/03/skiplist/skiplist.go

const (
	MaxHeight = 16
	PValue    = 0.5
)

var probabilities [MaxHeight]uint32

type node struct {
	sync.RWMutex
	v    int
	next [MaxHeight]*node
	prev [MaxHeight]*node
}

type SkipList struct {
	head   *node
	tail   *node
	height int
	length int
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
func (skl *SkipList) search(v int) (prev *node, journey [MaxHeight]*node) {
	prev = skl.head
	prev.RLock()

	for level := skl.height - 1; level >= 0; level-- {
		next := prev.next[level]
		next.RLock()
		for next != skl.tail {
			if next == nil {
				panic("skl reached nil pointer before tail")
			}

			if v < next.v {
				// Move down a level if next value in current level is larger than v.
				break
			}

			prev.RUnlock()
			prev = next
			next = prev.next[level]
		}
		next.RUnlock()
		journey[level] = prev
	}

	return journey[0], journey
}

func (skl *SkipList) SearchLE(val int) (int, bool) {
	v, _ := skl.search(val)
	if v == skl.head {
		return 0, false
	}
	return v.v, true
}

func (skl *SkipList) Insert(val int) {
	_, journey := skl.search(val)

	height := randomHeight()
	newNode := &node{v: val}

	for level := 0; level < height; level++ {
		prev := journey[level]
		if prev == nil {
			// If prev is nil, this means the level is not yet unlocked in the skiplist.
			prev = skl.head
		}
		next := prev.next[level]

		prev.Lock()
		next.Lock()

		newNode.next[level] = next
		newNode.prev[level] = prev

		next.prev[level] = newNode
		prev.next[level] = newNode

		prev.Unlock()
		next.Unlock()
	}

	if height > skl.height {
		skl.height = height
	}

	skl.length += 1
}

func (skl *SkipList) Delete(start int, end int) {
	target, _ := skl.search(start)
	if target.v != start {
		target = target.next[0]
	}

	for target.v < end && target != skl.tail {
		for level := 0; level < skl.height; level++ {
			if target.prev[level] != nil {
				target.next[level].prev[level] = target.prev[level]
				target.prev[level].next[level] = target.next[level]
			}
		}
		skl.length -= 1
		target = target.next[0]
	}

	skl.shrink()
}

func (skl *SkipList) Len() int { return skl.length }

func (skl *SkipList) shrink() {
	for level := skl.height - 1; level >= 0; level-- {
		if skl.head.next[level] == skl.tail {
			skl.height--
		}
	}
}
