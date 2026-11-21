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
	"io"
	"os"
	"path"
	"strconv"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// channelFile is the name of the file in each channel directory of the pointer log
// layout that holds the channel: a body length, a checksum of the length and the body,
// and the encoded channel.
const channelFile = "channel"

// channelHeaderSize is the size of the length and the checksum before the body.
const channelHeaderSize = 8

// createLogChannel keeps ch in the channel list of the pointer log layout: in memory,
// and durable in the channel file of its directory.
func (db *DB) createLogChannel(ch Channel) error {
	body, err := db.metaCodec.Encode(ch)
	if err != nil {
		return err
	}
	b := make([]byte, channelHeaderSize, channelHeaderSize+len(body))
	binary.LittleEndian.PutUint32(b, uint32(len(body)))
	b = append(b, body...)
	binary.LittleEndian.PutUint32(b[4:], channelSum(db.engine.db, ch.Key, b))
	dir := keyToDirName(ch.Key)
	sub, err := db.fs.Sub(dir)
	if err != nil {
		return err
	}
	f, err := sub.Open(channelFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return err
	}
	if _, err = f.WriteAt(b, 0); err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	if err = db.fs.SyncDir(dir); err != nil {
		return err
	}
	if err = db.fs.SyncDir(""); err != nil {
		return err
	}
	db.mu.channels[ch.Key] = ch
	db.mu.torn.Remove(ch.Key)
	return nil
}

// channelSum returns the checksum of the channel file b of key, over its length and its
// body.
func channelSum(db uint64, key ChannelKey, b []byte) uint32 {
	crc := place.Update(place.Channel(db, key).Start(), b[:4])
	return place.Update(crc, b[channelHeaderSize:])
}

// loadChannels reads the channel list of the pointer log layout from the channel file
// of each channel directory. A file that fails its check twice is a cut-short create
// when the key holds no segment, and loadChannels returns its key as torn. On a key
// with segments, it refuses with an error that wraps place.ErrCorrupt.
func (db *DB) loadChannels() (channels map[ChannelKey]Channel, torn set.Set[ChannelKey], err error) {
	infos, err := db.fs.List("")
	if err != nil {
		return nil, nil, err
	}
	channels, torn = make(map[ChannelKey]Channel), set.New[ChannelKey]()
	for _, info := range infos {
		key, err := strconv.ParseUint(info.Name(), 10, 32)
		if err != nil || !info.IsDir() {
			continue
		}
		ch, ok, err := db.readChannel(ChannelKey(key))
		switch {
		case err != nil:
			return nil, nil, err
		case ok:
			channels[ch.Key] = ch
		case db.engine.stored.Contains(ChannelKey(key)):
			return nil, nil, errors.Wrapf(
				place.ErrCorrupt,
				"the channel file of key %d fails its check",
				key,
			)
		default:
			torn.Add(ChannelKey(key))
		}
	}
	return channels, torn, nil
}

// readChannel reads the channel file of key, and reads it once more when it fails its
// check. It returns false when the file fails both reads. A missing file is a channel
// directory with no channel, and readChannel skips it.
func (db *DB) readChannel(key ChannelKey) (Channel, bool, error) {
	name := path.Join(keyToDirName(key), channelFile)
	exists, err := db.fs.Exists(name)
	if err != nil || !exists {
		return Channel{}, false, err
	}
	for range 2 {
		b, err := db.readFile(name)
		if err != nil {
			return Channel{}, false, err
		}
		if !channelPasses(db.engine.db, key, b) {
			continue
		}
		var ch Channel
		if err = db.metaCodec.Decode(b[channelHeaderSize:], &ch); err != nil {
			return Channel{}, false, errors.Wrapf(
				place.ErrCorrupt,
				"the channel file of key %d passes its check but does not decode: %w",
				key,
				err,
			)
		}
		return ch, true, nil
	}
	return Channel{}, false, nil
}

// channelPasses reports whether b is a whole channel file of key that passes its check.
func channelPasses(db uint64, key ChannelKey, b []byte) bool {
	if len(b) < channelHeaderSize {
		return false
	}
	n := binary.LittleEndian.Uint32(b)
	if uint64(len(b)) != channelHeaderSize+uint64(n) {
		return false
	}
	return binary.LittleEndian.Uint32(b[4:]) == channelSum(db, key, b)
}

// readFile reads the whole file name.
func (db *DB) readFile(name string) (b []byte, err error) {
	f, err := db.fs.Open(name, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	b = make([]byte, info.Size())
	if _, err = f.ReadAt(b, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return b, nil
}
