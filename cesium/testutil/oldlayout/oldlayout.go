// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package oldlayout writes and reads the files of the old Cesium layout, the format of
// v0.62, for the specs of its conversion. It writes no sync, so a crash image of a
// fault file system can lose any of its writes. meta.json takes the encoding of v0.62.
package oldlayout

import (
	"cmp"
	"context"
	"encoding/binary"
	jsonv2 "encoding/json/v2"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

const (
	indexFile   = "index.domain"
	counterFile = "counter.domain"
	extension   = ".domain"
	// pointerSize is the size of a record of index.domain.
	pointerSize = 26
)

// codec is the encoding of meta.json in v0.62.
var codec = json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true))

// Pointer is a record of index.domain: the time range of a run of samples, and where
// its bytes lie in a data file.
type Pointer struct {
	// TimeRange is the range of the samples, its end exclusive.
	telem.TimeRange
	// File is the number N of the data file N.domain that holds the bytes.
	File uint16
	// Offset and Size are the place of the bytes in the file.
	Offset, Size uint32
}

// Key is the directory of one channel in the old layout.
type Key struct {
	// Channel is the channel that meta.json holds.
	Channel channel.Channel
	// Counter is the value of counter.domain: the number of data files that the
	// channel made.
	Counter uint32
	// Pointers are the records of index.domain, in file order.
	Pointers []Pointer
	// Files holds the bytes of each data file N.domain by N, empty files included.
	Files map[uint16][]byte
	// Leftovers holds the bytes of each other file of the directory by name, such as
	// N.domain_temp and N.domain_gc of an old compaction.
	Leftovers map[string][]byte
}

// Write writes the directory of each of keys to fs as the old layout does: meta.json,
// counter.domain, index.domain, then each data file in rising order of N, then each
// leftover in rising order of name. It syncs nothing.
func Write(fs xfs.FS, keys ...Key) error {
	for _, k := range keys {
		sub, err := fs.Sub(strconv.FormatUint(uint64(k.Channel.Key), 10))
		if err != nil {
			return err
		}
		b, err := codec.Encode(k.Channel)
		if err != nil {
			return err
		}
		if err = xfs.WriteFile(sub, meta.File, b); err != nil {
			return err
		}
		if err = write(sub, counterFile, binary.LittleEndian.AppendUint32(
			nil,
			k.Counter,
		)); err != nil {
			return err
		}
		b = make([]byte, 0, pointerSize*len(k.Pointers))
		for _, p := range k.Pointers {
			b = binary.LittleEndian.AppendUint64(b, uint64(p.Start))
			b = binary.LittleEndian.AppendUint64(b, uint64(p.End))
			b = binary.LittleEndian.AppendUint16(b, p.File)
			b = binary.LittleEndian.AppendUint32(b, p.Offset)
			b = binary.LittleEndian.AppendUint32(b, p.Size)
		}
		if err = write(sub, indexFile, b); err != nil {
			return err
		}
		for _, n := range slices.Sorted(maps.Keys(k.Files)) {
			if err = write(sub, fileName(n), k.Files[n]); err != nil {
				return err
			}
		}
		for _, name := range slices.Sorted(maps.Keys(k.Leftovers)) {
			if err = write(sub, name, k.Leftovers[name]); err != nil {
				return err
			}
		}
	}
	return nil
}

// Read reads the directory of each channel of the old layout in fs, in rising order of
// key.
func Read(ctx context.Context, fs xfs.FS) ([]Key, error) {
	infos, err := fs.List("")
	if err != nil {
		return nil, err
	}
	var keys []Key
	for _, info := range infos {
		if _, err := strconv.ParseUint(
			info.Name(),
			10,
			32,
		); err != nil ||
			!info.IsDir() {
			continue
		}
		sub, err := fs.Sub(info.Name())
		if err != nil {
			return nil, err
		}
		k, err := readKey(ctx, sub)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b Key) int {
		return cmp.Compare(a.Channel.Key, b.Channel.Key)
	})
	return keys, nil
}

// readKey reads the directory of one channel in fs.
func readKey(ctx context.Context, fs xfs.FS) (Key, error) {
	ch, err := meta.Read(ctx, fs, codec)
	if err != nil {
		return Key{}, err
	}
	k := Key{
		Channel:   ch,
		Files:     make(map[uint16][]byte),
		Leftovers: make(map[string][]byte),
	}
	b, err := xfs.ReadFile(fs, counterFile)
	if err != nil {
		return Key{}, err
	}
	if len(b) != 4 {
		return Key{}, errors.Newf("%s of key %d holds %d bytes, not 4", counterFile,
			ch.Key, len(b))
	}
	k.Counter = binary.LittleEndian.Uint32(b)
	if b, err = xfs.ReadFile(fs, indexFile); err != nil {
		return Key{}, err
	}
	for i := 0; i+pointerSize <= len(b); i += pointerSize {
		k.Pointers = append(k.Pointers, Pointer{
			Start:  telem.TimeStamp(binary.LittleEndian.Uint64(b[i:])),
			End:    telem.TimeStamp(binary.LittleEndian.Uint64(b[i+8:])),
			File:   binary.LittleEndian.Uint16(b[i+16:]),
			Offset: binary.LittleEndian.Uint32(b[i+18:]),
			Size:   binary.LittleEndian.Uint32(b[i+22:]),
		})
	}
	infos, err := fs.List("")
	if err != nil {
		return Key{}, err
	}
	for _, info := range infos {
		name := info.Name()
		if name == meta.File || name == counterFile || name == indexFile {
			continue
		}
		b, err := xfs.ReadFile(fs, name)
		if err != nil {
			return Key{}, err
		}
		num, ok := strings.CutSuffix(name, extension)
		if n, err := strconv.ParseUint(num, 10, 16); ok && err == nil {
			k.Files[uint16(n)] = b
		} else {
			k.Leftovers[name] = b
		}
	}
	return k, nil
}

// fileName returns the name of data file n.
func fileName(n uint16) string { return strconv.FormatUint(uint64(n), 10) + extension }

// write writes b to the file name of fs, which it creates or empties first.
func write(fs xfs.FS, name string, b []byte) error {
	f, err := fs.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return errors.Join(err, f.Close())
}
