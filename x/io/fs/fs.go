package fs

import (
	"github.com/spf13/afero"
	"os"
	"path"
)

type File interface {
	afero.File
}

const defaultPerm = 0755

type FS interface {
	Open(name string, flag int) (File, error)
	Sub(name string) (FS, error)
	List() ([]os.FileInfo, error)
	Exists(name string) (bool, error)
}

type defaultFS struct {
	dir  string
	perm os.FileMode
}

var DefaultFS FS = &defaultFS{perm: defaultPerm}

func (d *defaultFS) Open(name string, flag int) (File, error) {
	return os.OpenFile(path.Join(d.dir, name), flag, d.perm)
}

func (d *defaultFS) Sub(name string) (FS, error) {
	return OSDirFS(path.Join(d.dir, name))
}

func (d *defaultFS) Exists(name string) (bool, error) {
	_, err := os.Stat(path.Join(d.dir, name))
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (d *defaultFS) List() ([]os.FileInfo, error) {
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		infos[i], err = e.Info()
		if err != nil {
			return nil, err
		}
	}
	return infos, nil
}

func OSDirFS(dir string) (FS, error) {
	err := os.MkdirAll(dir, defaultPerm)
	return &defaultFS{dir: dir, perm: defaultPerm}, err
}

func NewMemFS() FS {
	return &memFS{
		Fs:   afero.NewMemMapFs(),
		perm: defaultPerm,
	}
}

type memFS struct {
	afero.Fs
	perm os.FileMode
}

func (m *memFS) Open(name string, flag int) (File, error) {
	return m.Fs.OpenFile(name, flag, m.perm)
}

func (m *memFS) Sub(name string) (FS, error) {
	return &memFS{Fs: afero.NewBasePathFs(m.Fs, name)}, nil
}

func (m *memFS) Exists(name string) (bool, error) {
	return afero.Exists(m.Fs, name)
}

func (m *memFS) List() ([]os.FileInfo, error) {
	return afero.ReadDir(m.Fs, ".")
}
