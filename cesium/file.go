package cesium

import (
	"github.com/arya-analytics/cesium/internal/allocate"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/kv"
)

const (
	// maxFileSize is the default maximum Size of a cesium file.
	maxFileSize = allocate.DefaultMaxSize
	// maxFileDescriptors is the default maximum number of file descriptors
	// cesium can open at a time.
	maxFileDescriptors = allocate.DefaultMaxDescriptors
	// cesiumDirectory is the directory in which cesium files are stored.
	cesiumDirectory = "cesium"
	// kvDirectory is the directory in which kv files are stored.
	kvDirectory = "kv"
)

type fileCounter struct{ kv.PersistedCounter }

func newFileCounter(kve kv.DB, key []byte) (*fileCounter, error) {
	counter, err := kv.NewPersistedCounter(kve, key)
	return &fileCounter{PersistedCounter: *counter}, err
}

// Next implements allocate.NextDescriptor.
func (f *fileCounter) Next() core.FileKey {
	v, err := f.Increment()
	if err != nil {
		panic(err)
	}
	return core.FileKey(v)
}
