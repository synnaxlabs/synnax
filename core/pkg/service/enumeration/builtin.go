// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/set"
)

var (
	valveKey             = uuid.MustParse("0fdb5bf9-8929-4839-84f4-5bc0cda63344")
	normallyOpenValveKey = uuid.MustParse("18dd0627-e014-4c38-9fc8-cd077bdf3888")
	pumpKey              = uuid.MustParse("21709d6c-c7a8-41ca-89d0-6e946236491e")
	switchKey            = uuid.MustParse("d396f0f9-46ba-45a3-b9b5-36fc8b610625")
	lightKey             = uuid.MustParse("ed3c515d-10eb-468a-95bf-15b6204bb5f2")
)

func isBuiltin(key Key) bool {
	return lo.ContainsBy(Builtins(), func(e Enumeration) bool { return e.Key == key })
}

// shippedEntry is an entry of a built-in enumeration, with no color.
type shippedEntry struct{ key, name string }

// twoState returns a built-in enumeration whose entries hold the values 0 and 1.
func twoState(key Key, name string, zero, one shippedEntry) Enumeration {
	return Enumeration{
		Key:  key,
		Name: name,
		Entries: []Entry{
			{Key: zero.key, Value: 0, Name: zero.name},
			{Key: one.key, Value: 1, Name: one.name},
		},
	}
}

// Builtins returns the enumerations that the Core ships, with their shipped values. A
// stored built-in can differ from its shipped values, because users can edit it.
func Builtins() []Enumeration {
	var (
		closed  = shippedEntry{"closed", "Closed"}
		open    = shippedEntry{"open", "Open"}
		off     = shippedEntry{"off", "Off"}
		on      = shippedEntry{"on", "On"}
		stopped = shippedEntry{"stopped", "Stopped"}
		running = shippedEntry{"running", "Running"}
	)
	return []Enumeration{
		twoState(valveKey, "Valve", closed, open),
		twoState(normallyOpenValveKey, "Normally open valve", open, closed),
		twoState(pumpKey, "Pump", stopped, running),
		twoState(switchKey, "Switch", off, on),
		twoState(lightKey, "Light", off, on),
	}
}

// createBuiltins creates each built-in enumeration that is missing, with its shipped
// values, and never changes one that exists.
func (s *Service) createBuiltins(ctx context.Context, db *gorp.DB) error {
	builtins := Builtins()
	keys := lo.Map(builtins, func(e Enumeration, _ int) Key { return e.Key })
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		var existing []Enumeration
		if err := s.NewRetrieve().
			Where(MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&existing).
			Exec(ctx, tx); err != nil {
			return err
		}
		stored := set.New(
			lo.Map(existing, func(e Enumeration, _ int) Key { return e.Key })...,
		)
		missing := lo.Reject(builtins, func(e Enumeration, _ int) bool {
			return stored.Contains(e.Key)
		})
		return s.NewWriter(tx).CreateMany(ctx, missing)
	})
}
