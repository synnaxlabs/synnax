package kv

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/kv"
)

type fileCounter struct{ kv.PersistedCounter }

func openFileCounter(kve kv.DB, key []byte) (*fileCounter, error) {
	counter, err := kv.OpenCounter(kve, key)
	return &fileCounter{PersistedCounter: *counter}, err
}

func (f *fileCounter) NextFile() (core.FileKey, error) {
	v, err := f.Add()
	return core.FileKey(v), err
}
