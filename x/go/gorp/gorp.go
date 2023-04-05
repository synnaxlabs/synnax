// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"
	"github.com/synnaxlabs/x/kv"
)

// Wrap wraps the provided key-value database in a DB.
func Wrap(kv kv.DB, opts ...Option) *DB {
	return &DB{DB: kv, options: newOptions(opts...)}
}

// DB is a wrapper around a kv.DB that queries can be executed against. DB implements
// the TypedWriter interface, so it can be provided to Query.Write.
type DB struct {
	kv.DB
	options
}

// NewWriter begins a new TypedWriter against the DB.
func (db *DB) NewWriter(ctx context.Context) Writer {
	return writer{Writer: db.NewWriter(ctx), opts: db.options}
}

// NewReader begins a new Reader against the DB.
func (db *DB) NewReader(ctx context.Context) Reader {
	return reader{Reader: db.NewReader(ctx), opts: db.options}
}

type Reader interface {
	kv.Reader
	options() options
}

type Writer interface {
	kv.Writer
	options() options
}

type writer struct {
	opts options
	kv.Writer
}

func (w writer) options() options { return w.opts }

type reader struct {
	opts options
	kv.Reader
}

func (r reader) options() options { return r.opts }
