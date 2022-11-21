package kfs

import (
	"sync"
	"time"
)

type entry[T comparable] struct {
	BaseFile
	sync.Mutex
	ls  time.Time
	key T
}

func (e *entry[T]) Age() time.Duration {
	return time.Since(e.ls)
}

func (e *entry[T]) Sync() error {
	e.ls = time.Now()
	return e.BaseFile.Sync()
}

func (e *entry[T]) Key() T {
	return e.key
}

func newEntry[T comparable](key T, f BaseFile) File[T] {
	return &entry[T]{
		Mutex:    sync.Mutex{},
		BaseFile: f,
		ls:       time.Now(),
		key:      key,
	}
}
