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
	// intentRecordSize is the size of one record: the key, the flags, then a CRC32C of
	// the record's offset, key, and flags.
	intentRecordSize = 12
	// flagClear marks a record that clears the intent of its key.
	flagClear uint32 = 1
)

// castagnoli is the CRC32C table of the intent records.
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

// openIntents opens the intents file in fs, creating it if it is missing, and replays
// its records. A record whose checksum fails holds nothing. The records after the last
// good one are the torn tail of an append that never returned, and openIntents
// truncates them. It also returns the number of bad records before the last good one,
// which a torn append cannot leave.
func openIntents(fs xfs.FS) (*intents, int, error) {
	file, err := fs.Open(intentsFile, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, 0, err
	}
	// A created file is durable only once its entry in the root is.
	if err = fs.SyncDir(""); err != nil {
		return nil, 0, joinClose(err, file)
	}
	in := &intents{fs: fs, file: file, keys: set.New[ChannelKey]()}
	data, err := io.ReadAll(io.NewSectionReader(file, 0, 1<<62))
	if err != nil {
		return nil, 0, joinClose(err, file)
	}
	var bad, corrupt int
	for off := 0; off+intentRecordSize <= len(data); off += intentRecordSize {
		key := binary.LittleEndian.Uint32(data[off:])
		flags := binary.LittleEndian.Uint32(data[off+4:])
		if binary.LittleEndian.Uint32(
			data[off+8:],
		) != recordCRC(
			int64(off),
			key,
			flags,
		) {
			bad++
			continue
		}
		if flags&flagClear != 0 {
			in.keys.Remove(key)
		} else {
			in.keys.Add(key)
		}
		in.size = int64(off + intentRecordSize)
		corrupt = bad
	}
	if in.size < int64(len(data)) {
		if err = file.Truncate(in.size); err == nil {
			err = file.Sync()
		}
		if err != nil {
			return nil, 0, joinClose(err, file)
		}
	}
	return in, corrupt, nil
}

func joinClose(err error, c io.Closer) error {
	if cErr := c.Close(); cErr != nil {
		return cErr
	}
	return err
}

// recordCRC returns the checksum of a record of key and flags at off.
func recordCRC(off int64, key ChannelKey, flags uint32) uint32 {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[:], uint64(off))
	binary.LittleEndian.PutUint32(b[8:], key)
	binary.LittleEndian.PutUint32(b[12:], flags)
	return crc32.Checksum(b[:], castagnoli)
}

// encode returns the records of keys with flags from off.
func encode(off int64, keys []ChannelKey, flags uint32) []byte {
	buf := make([]byte, len(keys)*intentRecordSize)
	for i, k := range keys {
		at := i * intentRecordSize
		binary.LittleEndian.PutUint32(buf[at:], k)
		binary.LittleEndian.PutUint32(buf[at+4:], flags)
		binary.LittleEndian.PutUint32(buf[at+8:], recordCRC(off+int64(at), k, flags))
	}
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

// append appends the records of keys with flags and syncs the file. The caller must
// hold in.mu.
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
	in.size += int64(len(keys) * intentRecordSize)
	return nil
}

// fail closes the file after the failed append err, and makes every later call fail.
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

// clear clears the intents of keys with one append of clear records and one sync. Keys
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

// Close closes the intents file, unless a failed append closed it.
func (in *intents) Close() error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.err != nil {
		return nil
	}
	return in.file.Close()
}
