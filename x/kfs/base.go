package kfs

import (
	"github.com/spf13/afero"
	"os"
)

// NewOS returns a new BaseFS that uses the os package.
func NewOS() BaseFS {
	return &osFS{}
}

// NewMem returns a new BaseFS that uses an afero memory filesystem.
func NewMem() BaseFS {
	return &memFS{fs: afero.NewMemMapFs()}
}

type osFS struct{}

func (o *osFS) Open(name string) (BaseFile, error) {
	return os.OpenFile(name, os.O_RDWR, 0644)

}

func (o *osFS) Create(name string) (BaseFile, error) {
	return os.Create(name)
}

func (o *osFS) Remove(name string) error {
	return os.Remove(name)
}

func (o *osFS) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(name, perm)
}

func (o *osFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

type memFS struct {
	fs afero.Fs
}

func (m *memFS) Open(name string) (BaseFile, error) {
	return m.fs.OpenFile(name, os.O_RDWR, 0644)
}

func (m *memFS) Create(name string) (BaseFile, error) {
	return m.fs.Create(name)
}

func (m *memFS) Remove(name string) error {
	return m.fs.Remove(name)
}

func (m *memFS) MkdirAll(name string, perm os.FileMode) error {
	return m.fs.MkdirAll(name, perm)
}

func (m *memFS) Stat(name string) (os.FileInfo, error) {
	return m.fs.Stat(name)
}
