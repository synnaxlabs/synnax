// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/x/errors"
)

// EtherCATChannelMapMigration re-keys the channel maps that released Consoles stored on
// EtherCAT slaves from PDO entry names to entry addresses. A name the slave repeats
// maps to its first entry, the one the Driver read. When two keys name one entry, the
// first in sorted order takes its address. A key that names no entry, or loses its
// entry to another key, is kept.
var EtherCATChannelMapMigration = migrate.NewEntry(
	"v60_ethercat_channel_map_addresses",
	func(_ context.Context, d Device) (Device, error) {
		if d.Make != "ethercat" {
			return d, nil
		}
		b, err := json.Marshal(d.Properties)
		if err != nil {
			return Device{}, errors.Wrapf(err, "device %s properties", d.Key)
		}
		var s slave
		if err = json.Unmarshal(b, &s); err != nil {
			return Device{}, errors.Wrapf(err, "device %s properties", d.Key)
		}
		rekeyChannels(d.Properties, "read", s.PDOs.Inputs)
		rekeyChannels(d.Properties, "write", s.PDOs.Outputs)
		return d, nil
	},
)

// slave is the part of an EtherCAT slave's properties that lists its PDO entries.
type slave struct {
	// PDOs holds the entries the slave's scan discovered.
	PDOs struct {
		// Inputs are the entries the slave sends to the master.
		Inputs []pdoEntry `json:"inputs"`
		// Outputs are the entries the master sends to the slave.
		Outputs []pdoEntry `json:"outputs"`
	} `json:"pdos"`
}

// pdoEntry is one PDO entry a slave's scan discovered.
type pdoEntry struct {
	// Name is the entry's name, which the slave may repeat across PDOs.
	Name string `json:"name"`
	// Index is the object dictionary index of the entry.
	Index uint16 `json:"index"`
	// SubIndex is the object dictionary subindex of the entry.
	SubIndex uint8 `json:"sub_index"`
}

func rekeyChannels(props map[string]any, dir string, entries []pdoEntry) {
	section, ok := props[dir].(map[string]any)
	if !ok {
		return
	}
	channels, ok := section["channels"].(map[string]any)
	if !ok {
		return
	}
	renames := make(map[string]string, 4*len(entries))
	add := func(old, to string) {
		if _, taken := renames[caseless(old)]; !taken {
			renames[caseless(old)] = to
		}
	}
	for _, e := range entries {
		addr := fmt.Sprintf("%d_%d", e.Index, e.SubIndex)
		for _, old := range []string{"auto_" + e.Name, "manual_" + addr} {
			add(old, addr)
			add(old+"_state", "state_"+addr)
		}
	}
	rekeyed := make(map[string]any, len(channels))
	for _, key := range slices.Sorted(maps.Keys(channels)) {
		to := key
		if addr, ok := renames[caseless(key)]; ok {
			if _, taken := rekeyed[addr]; !taken {
				to = addr
			}
		}
		rekeyed[to] = channels[key]
	}
	section["channels"] = rekeyed
}

// caseless folds the forms the Console's snake and camel case conversions give one key
// to a single string. The conversions only change letter case and add or drop an
// underscore before a letter.
func caseless(key string) string {
	runes := []rune(strings.ToLower(key))
	out := make([]rune, 0, len(runes))
	for i, r := range runes {
		if r == '_' && i+1 < len(runes) && unicode.IsLetter(runes[i+1]) {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
