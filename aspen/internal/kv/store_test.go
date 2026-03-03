// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// White-box unit tests for kvStore. Package kv (not kv_test) is required to
// access unexported types.
package kv

import (
	"fmt"
	"sync"
	"testing"

	xkv "github.com/synnaxlabs/x/kv"
)

func newTestOp(key, value string) Operation {
	return Operation{Change: xkv.Change{Key: []byte(key), Value: []byte(value)}}
}

// TestKVStoreApply verifies that apply() writes operations into the map without
// a full copy.
func TestKVStoreApply(t *testing.T) {
	t.Run("writes all operations", func(t *testing.T) {
		s := newStore()
		s.apply([]Operation{newTestOp("a", "1"), newTestOp("b", "2"), newTestOp("c", "3")})

		state, release := s.PeekState()
		defer release()

		for k, want := range map[string]string{"a": "1", "b": "2", "c": "3"} {
			if got := string(state[k].Value); got != want {
				t.Errorf("state[%q].Value = %q, want %q", k, got, want)
			}
		}
	})

	t.Run("overwrites existing entries", func(t *testing.T) {
		s := newStore()
		s.apply([]Operation{newTestOp("k", "first")})
		s.apply([]Operation{newTestOp("k", "second")})

		state, release := s.PeekState()
		defer release()

		if got := string(state["k"].Value); got != "second" {
			t.Errorf("state[k].Value = %q, want \"second\"", got)
		}
	})

	t.Run("handles empty operation slice", func(t *testing.T) {
		s := newStore()
		s.apply([]Operation{newTestOp("k", "v")})
		s.apply([]Operation{})

		state, release := s.PeekState()
		defer release()

		if _, ok := state["k"]; !ok {
			t.Error("empty apply erased existing entries")
		}
	})
}

// TestKVStoreConcurrentApplyAndPeek runs concurrent writers and readers to
// verify there are no data races. Run with -race to exercise the detector.
func TestKVStoreConcurrentApplyAndPeek(t *testing.T) {
	s := newStore()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			s.apply([]Operation{newTestOp(fmt.Sprintf("key%d", i), "v")})
		}(i)
		go func() {
			defer wg.Done()
			state, release := s.PeekState()
			_ = len(state)
			release()
		}()
	}
	wg.Wait()
}
