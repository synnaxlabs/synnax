package cesium

import (
	"github.com/synnaxlabs/cesium/internal/file"
	"github.com/synnaxlabs/x/kv"
)

const (
	// cesiumDirectory is the directory in which cesium files are stored.
	cesiumDirectory = "cesium"
	// kvDirectory is the directory in which kv files are stored.
	kvDirectory = "kv"
)

const (
	// fileCounterKey is the key for the counter that keeps track of the number of files
	// the DB has created.
	fileCounterKey = "cesium.nextFile"
)

type Counter struct{ kv.PersistedCounter }

func openFileCounter(kve kv.DB) (*Counter, error) {
	counter, err := kv.NewPersistedCounter(kve, []byte(fileCounterKey))
	return &Counter{PersistedCounter: *counter}, err
}

// Next implements allocate.NextDescriptor.
func (f *Counter) Next() file.Key {
	v, err := f.Add()
	if err != nil {
		panic(err)
	}
	return file.Key(v)
}
