package cesium

import (
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/kv"
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

type fileCounter struct{ kv.PersistedCounter }

func openFileCounter(kve kv.DB) (*fileCounter, error) {
	counter, err := kv.NewPersistedCounter(kve, []byte(fileCounterKey))
	return &fileCounter{PersistedCounter: *counter}, err
}

// Next implements allocate.NextDescriptor.
func (f *fileCounter) Next() core.FileKey {
	v, err := f.Add()
	if err != nil {
		panic(err)
	}
	return core.FileKey(v)
}
