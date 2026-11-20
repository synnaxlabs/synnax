// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"sync"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

const (
	// intentsFile is the root file that makes the delete intents durable. It is append
	// only, and never renamed.
	intentsFile = "delete-intents"
	// batchHeaderSize is the size of the header of one append: the record count, then
	// a CRC32C of the batch's offset and its records.
	batchHeaderSize = 8
	// intentRecordSize is the size of one record: the key, then the flags.
	intentRecordSize = 8
	// flagClear marks a record that clears the intent of its key.
	flagClear uint32 = 1
)

// castagnoli is the CRC32C table of the intent batches.
var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// intents holds the keys that a delete intends to drop, and the file that makes them
// durable. It is safe for concurrent use.
type intents struct {
	fs   xfs.FS
	file xfs.File
	keys set.Set[ChannelKey]
	// err is the error of a failed append. The state of the file is then unknown, so
	// the intents close the file and fail every later call until a restart.
	err  error
	mu   sync.Mutex
	size int64
}

// replay is the state that the batches of an intents file hold.
type replay struct {
	keys set.Set[ChannelKey]
	// size is the end of the last good batch.
	size int64
	// corrupt is the number of bad batches before the last good one.
	corrupt int
}

// openIntents opens the intents file in fs, creating it if it is missing, and replays
// its batches. A batch whose checksum fails holds nothing. The bytes after the last
// good batch are the torn tail of an append that never returned, and openIntents
// truncates them. It also returns the number of bad batches before the last good one,
// which a torn append cannot leave.
func openIntents(fs xfs.FS) (*intents, int, error) {
	file, err := fs.Open(intentsFile, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, 0, err
	}
	// A created file is durable only once its entry in the root is.
	if err = fs.SyncDir(""); err != nil {
		return nil, 0, errors.Join(err, file.Close())
	}
	data, err := readAll(file)
	if err != nil {
		return nil, 0, errors.Join(err, file.Close())
	}
	r := replayIntents(data)
	if r.size < int64(len(data)) || r.corrupt > 0 {
		// One bad read must not cost an intent, so a second read comes before the
		// verdict. A failed checksum never passes by chance, so the read that keeps
		// more is the right one.
		again, err := readAll(file)
		if err != nil {
			return nil, 0, errors.Join(err, file.Close())
		}
		if r2 := replayIntents(again); r2.size > r.size ||
			(r2.size == r.size && r2.corrupt < r.corrupt) {
			r, data = r2, again
		}
	}
	if r.size < int64(len(data)) {
		if err = file.Truncate(r.size); err == nil {
			err = file.Sync()
		}
		if err != nil {
			return nil, 0, errors.Join(err, file.Close())
		}
	}
	return &intents{fs: fs, file: file, keys: r.keys, size: r.size}, r.corrupt, nil
}

func readAll(f xfs.File) ([]byte, error) {
	return io.ReadAll(io.NewSectionReader(f, 0, 1<<62))
}

// replayIntents replays the batches of data, in order.
func replayIntents(data []byte) replay {
	var (
		r   = replay{keys: set.New[ChannelKey]()}
		bad int
	)
	for off := 0; off+batchHeaderSize <= len(data); {
		count := uint64(binary.LittleEndian.Uint32(data[off:]))
		// A count past the bytes, or zero, leaves no way to find the next batch.
		if count == 0 ||
			count > uint64(len(data)-off-batchHeaderSize)/intentRecordSize {
			break
		}
		end := off + batchHeaderSize + int(count)*intentRecordSize
		records := data[off+batchHeaderSize : end]
		if binary.LittleEndian.Uint32(data[off+4:]) != batchCRC(int64(off), records) ||
			!knownFlags(records) {
			bad++
			off = end
			continue
		}
		for at := 0; at < len(records); at += intentRecordSize {
			key := binary.LittleEndian.Uint32(records[at:])
			if binary.LittleEndian.Uint32(records[at+4:])&flagClear != 0 {
				r.keys.Remove(key)
			} else {
				r.keys.Add(key)
			}
		}
		r.size = int64(end)
		r.corrupt = bad
		off = end
	}
	return r
}

func knownFlags(records []byte) bool {
	for at := 0; at < len(records); at += intentRecordSize {
		if binary.LittleEndian.Uint32(records[at+4:])&^flagClear != 0 {
			return false
		}
	}
	return true
}

// batchCRC returns the checksum of the records of a batch at off.
func batchCRC(off int64, records []byte) uint32 {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(off))
	return crc32.Update(crc32.Update(0, castagnoli, b[:]), castagnoli, records)
}

// encode returns the batch of keys with flags at off.
func encode(off int64, keys []ChannelKey, flags uint32) []byte {
	buf := make([]byte, batchHeaderSize+len(keys)*intentRecordSize)
	records := buf[batchHeaderSize:]
	for i, k := range keys {
		binary.LittleEndian.PutUint32(records[i*intentRecordSize:], k)
		binary.LittleEndian.PutUint32(records[i*intentRecordSize+4:], flags)
	}
	binary.LittleEndian.PutUint32(buf, uint32(len(keys)))
	binary.LittleEndian.PutUint32(buf[4:], batchCRC(off, records))
	return buf
}

// add makes intents for keys durable, with one append and one sync.
func (in *intents) add(keys []ChannelKey) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if err := in.append(keys, 0); err != nil {
		return err
	}
	in.keys.Add(keys...)
	return nil
}

// append appends one batch of the records of keys with flags and syncs the file. The
// caller must hold in.mu.
func (in *intents) append(keys []ChannelKey, flags uint32) error {
	if in.err != nil {
		return in.err
	}
	if _, err := in.file.WriteAt(encode(in.size, keys, flags), in.size); err != nil {
		return in.fail(err)
	}
	if err := in.file.Sync(); err != nil {
		return in.fail(err)
	}
	in.size += int64(batchHeaderSize + len(keys)*intentRecordSize)
	return nil
}

// fail closes the file after the failed write err, and makes every later call fail.
// The caller must hold in.mu.
func (in *intents) fail(err error) error {
	in.err = errors.Wrap(err, "delete intents failed to append, and no delete can run "+
		"until a restart")
	return errors.Join(in.err, in.file.Close())
}

// has reports whether key has an intent.
func (in *intents) has(key ChannelKey) bool {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.keys.Contains(key)
}

// all returns the keys with an intent.
func (in *intents) all() []ChannelKey {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.keys.Slice()
}

// clear clears the intents of keys with one batch of clear records and one sync. Keys
// with no intent are ignored.
func (in *intents) clear(keys ...ChannelKey) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	cleared := make([]ChannelKey, 0, len(keys))
	for _, k := range keys {
		if in.keys.Contains(k) {
			cleared = append(cleared, k)
		}
	}
	if len(cleared) == 0 {
		return nil
	}
	if err := in.append(cleared, flagClear); err != nil {
		return err
	}
	in.keys.Remove(cleared...)
	return nil
}

// truncateIfEmpty empties the file when no key has an intent, so the file does not
// grow with every delete.
func (in *intents) truncateIfEmpty() error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.err != nil || len(in.keys) > 0 || in.size == 0 {
		return in.err
	}
	if err := in.file.Truncate(0); err != nil {
		return in.fail(err)
	}
	if err := in.file.Sync(); err != nil {
		return in.fail(err)
	}
	in.size = 0
	return nil
}

// Close closes the intents file, unless a failed append closed it.
func (in *intents) Close() error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.err != nil {
		return nil
	}
	return in.file.Close()
}
