// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import "github.com/synnaxlabs/x/telem"

// Record is a decoded index pointer exposed for read-only inspection tooling. It
// mirrors the on-disk layout of a pointer without granting access to the live index.
type Record struct {
	// TimeRange is the time interval occupied by the domain. Start is inclusive and
	// End is exclusive. Two domains sharing an end and start are continuous.
	telem.TimeRange
	// FileKey is the numeric key of the data file holding the domain.
	FileKey uint16
	// Offset is the byte offset of the domain within its data file.
	Offset uint32
	// Size is the byte length of the domain within its data file.
	Size uint32
}

// RecordSize is the encoded size, in bytes, of one Record in the index file.
const RecordSize = pointerByteSize

const (
	// IndexFileName is the on-disk name of the domain index file.
	IndexFileName = indexFile
	// CounterFileName is the on-disk name of the file-key counter file.
	CounterFileName = counterFile
	// Extension is the suffix shared by every domain file.
	Extension = extension
)

// DataFileName returns the on-disk name of the data file with the given key.
func DataFileName(key uint16) string { return fileKeyToName(key) }

// DecodeRecords decodes raw index-file bytes into records. A trailing partial record
// is dropped, matching the engine's decoder.
func DecodeRecords(b []byte) []Record {
	ptrs := (&pointerCodec{}).decode(b)
	records := make([]Record, len(ptrs))
	for i, p := range ptrs {
		records[i] = Record{
			TimeRange: p.TimeRange,
			FileKey:   p.fileKey,
			Offset:    p.offset,
			Size:      p.size,
		}
	}
	return records
}
