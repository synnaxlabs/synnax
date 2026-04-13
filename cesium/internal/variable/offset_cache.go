// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable

import (
	"encoding/binary"
	"io"
	"sync"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/telem"
)

// offsetTable stores byte offsets for each sample in a domain. Offsets are uint32,
// matching domain.pointer.size. Domains larger than ~4GB are unsupported at the
// storage layer, so uint32 is sufficient here.
type offsetTable struct {
	offsets     []uint32
	sampleCount int64
}

func (t *offsetTable) byteOffsetAt(sampleIdx int64) telem.Size {
	return telem.Size(t.offsets[sampleIdx])
}

type offsetCache struct {
	mu     sync.RWMutex
	tables map[uint32]*offsetTable
}

func newOffsetCache() *offsetCache {
	return &offsetCache{tables: make(map[uint32]*offsetTable)}
}

func (c *offsetCache) get(domainIdx uint32) (*offsetTable, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.tables[domainIdx]
	return t, ok
}

func (c *offsetCache) set(domainIdx uint32, t *offsetTable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tables[domainIdx] = t
}

func (c *offsetCache) invalidate(domainIdx uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tables, domainIdx)
}

func (c *offsetCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tables = make(map[uint32]*offsetTable)
}

func buildOffsetTable(r *domain.Reader, domainSize telem.Size) (*offsetTable, error) {
	t := &offsetTable{}
	buf := make([]byte, 4)
	var pos int64
	for pos+4 <= int64(domainSize) {
		t.offsets = append(t.offsets, uint32(pos))
		n, err := r.ReadAt(buf, pos)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n < 4 {
			break
		}
		length := int64(binary.LittleEndian.Uint32(buf))
		pos += 4 + length
		t.sampleCount++
	}
	return t, nil
}

func buildOffsetTableFromBytes(data []byte) *offsetTable {
	t := &offsetTable{}
	offset := 0
	for offset+4 <= len(data) {
		t.offsets = append(t.offsets, uint32(offset))
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4 + length
		t.sampleCount++
	}
	return t
}
