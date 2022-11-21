package fs

import (
	"github.com/spf13/afero"
	"os"
	"path"
)

type File interface {
	afero.File
}

type FS interface {
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
	Sub(name string) (FS, error)
}

type defaultFS struct {
	dir string
}

var DefaultFS FS = &defaultFS{}

func (d *defaultFS) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return os.OpenFile(path.Join(d.dir, name), flag, perm)
}

func (d *defaultFS) Sub(name string) (FS, error) {
	return OSDirFS(path.Join(d.dir, name))
}

func OSDirFS(dir string) (FS, error) {
	err := os.MkdirAll(dir, 0755)
	return &defaultFS{dir: dir}, err
}
