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

// Records returns the RecordOf of each channel that PublishTable and PublishActions
// created. The caller must not modify the map.
func (p *Provider) Records() map[channel.Key]RecordOf { return p.records }

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

// unmarshalKeySample returns the key that keySample encodes as sample.
func unmarshalKeySample[K gorp.Key](sample []byte) (K, error) {
	var k K
	if v := reflect.ValueOf(&k).Elem(); v.Kind() == reflect.String {
		v.SetString(string(sample))
		return k, nil
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(&k)), unsafe.Sizeof(k)), sample)
	return k, nil
}
