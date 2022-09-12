// Package memkv implements an in-memory key value store using cockroachdb's pebble storage engine.
// It's particularly useful for testing scenarios.
package memkv

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
)

// New opens a new in-memory key-value store implementing the kv.db interface.
func New() kv.DB {
	db, err := pebble.Open("", &pebble.Options{FS: vfs.NewMem()})
	if err != nil {
		panic(err)
	}
	return pebblekv.Wrap(db)
}
