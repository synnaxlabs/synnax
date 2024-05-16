// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
//
// channelGroup implements tools for constructing a set of channels. The set of
// channels is constructed as a sorted array of channelRanges. Each channelRange
// consists of the local key that the range starts at and the size of the
// channelRange (how many consecutive keys it is).

package channel

import (
	"slices"
	"sort"
)

// channelRange is the basic struct for ChannelGroup, and is defined as a set of
// consecutive channel keys.
type channelRange struct {
	// start is the local key that the channelRange starts at.
	start uint16
	// size is how many consecutive keys there are in a row.
	size uint16
}

// end returns the key that is at the end of channelRange r
func (r channelRange) end() uint16 {
	return r.start + r.size - 1
}

// ChannelGroup extends []channelRange with a few convenience methods.
type ChannelGroup []channelRange

// Len returns the length of c. Implements sort.interface.
func (c *ChannelGroup) Len() int {
	return len(*c)
}

// Swap switches the places of (*c)[i] and (*c)[j]. Implements sort.interface.
func (c *ChannelGroup) Swap(i, j int) {
	temp := (*c)[i]
	(*c)[i] = (*c)[j]
	(*c)[j] = temp
}

// Less returns true if the key of (*c)[i] is before the key of (*c)[j].
// Implements sort.interface.
func (c *ChannelGroup) Less(i, j int) bool {
	return (*c)[i].start < (*c)[j].start
}

// InsertKey allows you to insert one single key into the channelGroup c.
func (c *ChannelGroup) InsertKey(key Key) {
	c.InsertKeys(Keys{key})
}

// InsertKeys allows you to insert several keys into the channelGroup c.
func (c *ChannelGroup) InsertKeys(keys Keys) {
	if len(keys) == 0 {
		return
	}

	localKeys := make([]uint16, len(keys))
	for i, key := range keys {
		localKeys[i] = key.LocalKey()
	}

	r := channelRange{
		start: localKeys[0],
		size:  0,
	}
	for i, key := range localKeys {
		if r.start+r.size == key {
			r.size++
		} else {
			c.insert(r)
			r.start = localKeys[i]
			r.size = 1
		}
	}
	c.insert(r)
}

// RemoveKey removes key from c.
func (c *ChannelGroup) RemoveKey(key Key) {
	c.RemoveKeys(Keys{key})
}

// RemoveKeys removes all keys in keys from c.
func (c *ChannelGroup) RemoveKeys(keys Keys) {
	if len(keys) == 0 {
		return
	}

	localKeys := make([]uint16, len(keys))
	i := 0
	for _, key := range keys {
		if c.IsKeyIn(key) {
			localKeys[i] = key.LocalKey()
			i++
		}
	}
	localKeys = localKeys[0:i]
	if len(localKeys) == 0 {
		return
	}

	r := channelRange{
		start: localKeys[0],
		size:  0,
	}
	for i, key := range localKeys {
		if r.start+r.size == key {
			r.size++
		} else {
			c.remove(r)
			r.start = localKeys[i]
			r.size = 1
		}
	}
	c.remove(r)
}

// GetChannelsBeforeKey returns the number of channels in c before key. If key
// is in c, that does not count towards the number of channels returned.
func (c *ChannelGroup) GetChannelsBeforeKey(key Key) uint16 {
	localKey := key.LocalKey()
	counter := uint16(0)
	for i := 0; i < len(*c) && (*c)[i].start < localKey; i++ {
		if (*c)[i].end() >= localKey {
			counter += localKey - (*c)[i].start
		} else {
			counter += (*c)[i].size
		}
	}
	return counter
}

// IsKeyIn return true if the key is in c and false if key is not in c
func (c ChannelGroup) IsKeyIn(key Key) bool {
	localKey := key.LocalKey()
	i := sort.Search(len(c), func(i int) bool { return c[i].start > localKey })
	if i == 0 || c[i-1].end() < localKey {
		return false
	}
	return true
}

// delete removes the element c[i] from c.
func (c *ChannelGroup) delete(i int) {
	(*c) = slices.Delete(*c, i, i+1)
}

// insert inserts r into c. insert also makes sure that c
// is compressed to the minimum size possible.
func (c *ChannelGroup) insert(r channelRange) {
	if r.size == 0 {
		return
	}
	if len(*c) == 0 {
		*c = ChannelGroup{r}
		return
	}

	// this is the first channelRange in c whose start is later than r's
	i := sort.Search(len(*c), func(i int) bool { return (*c)[i].start > r.start })

	// check if the channelRange to the left overlaps with r. If so, we remove the
	// overlap from r and insert it again.
	if i != 0 {
		endOfBefore := (*c)[i-1].end()
		if endOfBefore >= r.start {
			overlap := endOfBefore - r.start + 1
			if int(r.size)-int(overlap) <= 0 {
				return
			}
			r.size -= overlap
			r.start += overlap
			c.insert(r)
			return
		}
	}

	// check if the channelRange to the right overlaps with r.
	if i != len(*c) {
		startOfNext := (*c)[i].start
		if r.end() >= startOfNext {
			if r.end() >= (*c)[i].end() {
				(*c).delete(i)
				c.insert(r)
				return
			}
			overlap := r.end() - startOfNext + 1
			r.size -= overlap
			c.insert(r)
			return
		}
	}

	// At this point, r does not overlap at all with any of the
	// channelRanges in c.

	// This checks if we need to compress c
	if i != 0 && (*c)[i-1].end() == r.start-1 {
		// we need to left compress
		(*c)[i-1].size += r.size
		// we may need to compress on the right
		if i != len(*c) && r.end() == (*c)[i].start-1 {
			(*c)[i-1].size += (*c)[i].size
			(*c).delete(i)
		}
		return
	} else if i != len(*c) && r.end() == (*c)[i].start-1 {
		r.size += (*c)[i].size
		(*c).delete(i)
	}

	*c = slices.Insert(*c, i, r)
}

// remove removes the channelRange r from c.
func (c *ChannelGroup) remove(r channelRange) {
	if len(*c) == 0 || r.size == 0 {
		return
	}
	i := sort.Search(len(*c), func(i int) bool { return (*c)[i].end() >= r.start })
	if i == len(*c) || r.end() < (*c)[i].start {
		return
	}

	if r.start < (*c)[i].start {
		r.start = (*c)[i].start
		r.size -= (*c)[i].start - r.start
	}
	if r.end() > (*c)[i].end() {
		newRemove := channelRange{
			start: (*c)[i].end() + 1,
			size:  r.end() - (*c)[i].end(),
		}
		c.remove(newRemove)
		r.size -= newRemove.size
	}

	// now, we know r is fully in c[i]
	if r.start == (*c)[i].start {
		if r.size == (*c)[i].size {
			c.delete(i)
			return
		}
		(*c)[i].size -= r.size
		(*c)[i].start += r.size
		return
	}

	if r.end() == (*c)[i].end() {
		(*c)[i].size -= r.size
		return
	}

	// r is in the middle of c, we need to split c. (*c)[i] will be to the left
	// of r, newChannelRange will be to the right of r.
	newChannelRange := channelRange{
		start: r.end() + 1,
		size:  (*c)[i].end() - r.end(),
	}
	(*c)[i].size = r.start - (*c)[i].start
	c.insert(newChannelRange)
}
