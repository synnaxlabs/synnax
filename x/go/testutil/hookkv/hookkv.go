// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package hookkv provides a kv.DB whose transaction commits a test can intercept. Open
// every store of a test on one DB, so they share the indexes and write lock that a
// second store over the same kv.DB would not.
package hookkv

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/set"
)

// Hook runs in place of a commit that it intercepts. commit applies the transaction.
type Hook func(commit func() error) error

// DB is a kv.DB whose transaction commits run through a Hook when one intercepts them.
// The hook of the commit's context goes first, then the first key hook that matched.
// Writes made without a transaction are never intercepted.
type DB struct {
	kv.DB
	mu   sync.Mutex
	keys set.Set[*keyHook]
}

type keyHook struct {
	marker []byte
	hook   Hook
}

// Wrap returns a DB over db.
func Wrap(db kv.DB) *DB { return &DB{DB: db, keys: set.New[*keyHook]()} }

type contextKey struct{}

// WithHook returns a context whose transaction commits run through hook.
func WithHook(ctx context.Context, hook Hook) context.Context {
	return context.WithValue(ctx, contextKey{}, hook)
}

// HookKey runs each commit that sets a key containing marker through hook, until
// remove is called.
func (d *DB) HookKey(marker []byte, hook Hook) (remove func()) {
	h := &keyHook{marker: marker, hook: hook}
	d.mu.Lock()
	d.keys.Add(h)
	d.mu.Unlock()
	return func() {
		d.mu.Lock()
		d.keys.Remove(h)
		d.mu.Unlock()
	}
}

// Hold returns a Hook that holds the first commit it intercepts until release closes,
// and closes held once that commit waits. Later commits pass.
func Hold(held chan<- struct{}, release <-chan struct{}) Hook {
	var taken atomic.Bool
	return func(commit func() error) error {
		if taken.CompareAndSwap(false, true) {
			close(held)
			<-release
		}
		return commit()
	}
}

// OpenTx implements kv.DB.
func (d *DB) OpenTx() kv.Tx { return &tx{Tx: d.DB.OpenTx(), db: d} }

func (d *DB) match(key []byte) Hook {
	d.mu.Lock()
	defer d.mu.Unlock()
	for h := range d.keys {
		if bytes.Contains(key, h.marker) {
			return h.hook
		}
	}
	return nil
}

type tx struct {
	kv.Tx
	db      *DB
	keyHook Hook
}

// Set implements kv.Writer.
func (t *tx) Set(ctx context.Context, key, value []byte) error {
	if t.keyHook == nil {
		t.keyHook = t.db.match(key)
	}
	return t.Tx.Set(ctx, key, value)
}

// Commit implements kv.Tx.
func (t *tx) Commit(ctx context.Context, opts ...any) error {
	commit := func() error { return t.Tx.Commit(ctx, opts...) }
	if hook, ok := ctx.Value(contextKey{}).(Hook); ok {
		return hook(commit)
	}
	if t.keyHook != nil {
		return t.keyHook(commit)
	}
	return commit()
}
