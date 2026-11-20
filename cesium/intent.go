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
	"slices"
	"sync"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

const (
	// intentsFile is the root file that makes the delete intents durable.
	intentsFile = "delete-intents"
	// intentsTempFile holds a rewrite of intentsFile until it renames over it.
	intentsTempFile = intentsFile + ".tmp"
	// intentRecordSize is the size of one record: the key, then a CRC32C of the
	// record's offset and key.
	intentRecordSize = 8
)

// castagnoli is the CRC32C table of the intent records.
var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// intents holds the keys that a delete intends to drop, and the file that makes them
// durable. It is safe for concurrent use.
type intents struct {
	fs   xfs.FS
	file xfs.File
	keys set.Set[ChannelKey]
	mu   sync.Mutex
	size int64
}

// openIntents opens the intents file in fs, creating it if it is missing. A record
// whose checksum fails holds no intent. The records after the last good one are the
// torn tail of an append that never returned, and openIntents truncates them. It also
// returns the number of bad records before the last good one, which a torn append
// cannot leave.
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
		if binary.LittleEndian.Uint32(data[off+4:]) != recordCRC(int64(off), key) {
			bad++
			continue
		}
		in.keys.Add(key)
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

// recordCRC returns the checksum of a record of key at off.
func recordCRC(off int64, key ChannelKey) uint32 {
	var b [12]byte
	binary.LittleEndian.PutUint64(b[:], uint64(off))
	binary.LittleEndian.PutUint32(b[8:], key)
	return crc32.Checksum(b[:], castagnoli)
}

// encode returns the records of keys from off.
func encode(off int64, keys []ChannelKey) []byte {
	buf := make([]byte, len(keys)*intentRecordSize)
	for i, k := range keys {
		at := i * intentRecordSize
		binary.LittleEndian.PutUint32(buf[at:], k)
		binary.LittleEndian.PutUint32(buf[at+4:], recordCRC(off+int64(at), k))
	}
	return buf
}

// add makes intents for keys durable, with one write and one sync.
func (in *intents) add(keys []ChannelKey) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if _, err := in.file.WriteAt(encode(in.size, keys), in.size); err != nil {
		return err
	}
	if err := in.file.Sync(); err != nil {
		return err
	}
	in.size += int64(len(keys) * intentRecordSize)
	in.keys.Add(keys...)
	return nil
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

// clear removes the intents of keys. It rewrites the file through a temporary file and
// a rename inside the root, then syncs the root, so a crash leaves either the old file
// or the new one.
func (in *intents) clear(keys ...ChannelKey) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	kept := in.keys.Copy()
	kept.Remove(keys...)
	if len(kept) == len(in.keys) {
		return nil
	}
	remaining := kept.Slice()
	slices.Sort(remaining)
	if err := in.rewrite(remaining); err != nil {
		return err
	}
	in.keys = kept
	return nil
}

func (in *intents) rewrite(keys []ChannelKey) error {
	tmp, err := in.fs.Open(intentsTempFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return err
	}
	data := encode(0, keys)
	if _, err = tmp.WriteAt(data, 0); err == nil {
		err = tmp.Sync()
	}
	if err = joinClose(err, tmp); err != nil {
		return err
	}
	if err = in.file.Close(); err != nil {
		return err
	}
	// The handle opens again on whichever file the rename leaves in place.
	renamed := in.fs.Rename(intentsTempFile, intentsFile)
	if renamed == nil {
		in.size = int64(len(data))
		err = in.fs.SyncDir("")
	}
	file, openErr := in.fs.Open(intentsFile, os.O_RDWR)
	if openErr == nil {
		in.file = file
	}
	return errors.Join(renamed, err, openErr)
}

// Close closes the intents file.
func (in *intents) Close() error { return in.file.Close() }
