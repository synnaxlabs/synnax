// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"encoding/json/v2"
	"reflect"
	"unsafe"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// RecordOf returns the resource of the record that one sample of a change channel
// describes. It returns an error when the sample does not hold a key.
type RecordOf func(sample []byte) (resource.ID, error)

// check returns what a subject receives of a sample, given whether the subject can
// retrieve each resource: the sample when send is true and replacement is nil, another
// sample when replacement is not nil, and nothing when send is false.
type check = func(allowed func(resource.ID) bool) (replacement []byte, send bool)

// Record finds the records that the samples of a change channel describe.
type Record struct {
	// Of returns the record of one sample.
	Of RecordOf
	// Check returns the check of one sample. A streamer sends the sample only to a
	// subject that passes it, in place of a check of the record that Of returns, and
	// checks that record when the check is nil. It returns an error when the check of
	// the sample is gone, which fails the stream. Nil for a channel that checks only
	// that record.
	Check func(sample []byte) (check, error)
	// Deleted is true when each sample describes a record that was deleted.
	Deleted bool
}

// Records returns the Record of each channel that PublishTable and PublishActions
// created. The caller must not modify the map.
func (p *Provider) Records() map[channel.Key]Record { return p.records }

// Publishes reports whether PublishTable or PublishActions created the channel.
func (p *Provider) Publishes(key channel.Key) bool {
	_, ok := p.records[key]
	return ok
}

// UnmarshalEntryKey returns the key of the entry that a JSON sample holds. Use it as
// TableConfig.SetKey when the key field of the JSON is not the key of the entry.
func UnmarshalEntryKey[K gorp.Key, E gorp.Entry[K]](sample []byte) (K, error) {
	var e E
	if err := json.Unmarshal(sample, &e); err != nil {
		return *new(K), err
	}
	return e.GorpKey(), nil
}

func recordOf[K gorp.Key](
	key func(sample []byte) (K, error),
	res func(K) resource.ID,
) RecordOf {
	return func(sample []byte) (resource.ID, error) {
		k, err := key(sample)
		if err != nil {
			return resource.ID{}, err
		}
		return res(k), nil
	}
}

// unmarshalKey returns the key field of a JSON sample.
func unmarshalKey[K gorp.Key](sample []byte) (K, error) {
	var v struct {
		Key *K `json:"key"`
	}
	if err := json.Unmarshal(sample, &v); err != nil {
		return *new(K), err
	}
	if v.Key == nil {
		return *new(K), errors.New("[signals] - sample has no key")
	}
	return *v.Key, nil
}

// unmarshalKeySample returns the key that keySample encodes as sample. It fails when a
// fixed-size key and sample differ in length.
func unmarshalKeySample[K gorp.Key](sample []byte) (K, error) {
	var k K
	if v := reflect.ValueOf(&k).Elem(); v.Kind() == reflect.String {
		v.SetString(string(sample))
		return k, nil
	}
	if size := int(unsafe.Sizeof(k)); len(sample) != size {
		return k, errors.Newf(
			"[signals] - delete sample has %d bytes, but a %T key has %d",
			len(sample), k, size,
		)
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(&k)), unsafe.Sizeof(k)), sample)
	return k, nil
}
