package kv

import (
	"bytes"
	"io"
)

// Flusher represents a type who can flush its contents to a Writer.
type Flusher interface {
	Flush(w io.Writer) error
}

// Loader represents a type who can load its contents from a Reader.
type Loader interface {
	Load(r io.Reader) error
}

// FlushLoader is a type that implements Flusher and Loader.
type FlushLoader interface {
	Flusher
	Loader
}

// Flush writes the contents of the Flusher to a writable key-value store.
func Flush(kv Writer, key []byte, flusher Flusher, opts ...interface{}) error {
	b := new(bytes.Buffer)
	if err := flusher.Flush(b); err != nil {
		return err
	}
	return kv.Set(key, b.Bytes(), opts...)
}

// Load loads the contents of the Loader  from a readable key-value store.
func Load(kv Reader, key []byte, loader Loader) error {
	b, err := kv.Get(key)
	if err != nil {
		return err
	}
	return LoadBytes(b, loader)
}

// LoadBytes loads the contents of a byte slice into a Loader.
func LoadBytes(b []byte, loader Loader) error { return loader.Load(bytes.NewReader(b)) }
