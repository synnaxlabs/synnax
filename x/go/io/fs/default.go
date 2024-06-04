package fs

import "os"

type defaultFS struct {
	perm os.FileMode
}

var Default FS = &defaultFS{perm: defaultPerm}

func (d *defaultFS) Open(name string, flag int) (File, error) {
	return os.OpenFile(name, flag, d.perm)
}

func (d *defaultFS) Sub(name string) (FS, error) {
	if err := os.MkdirAll(name, d.perm); err != nil {
		return nil, err
	}
	return &subFS{dir: name, FS: d}, nil
}

func (d *defaultFS) Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (d *defaultFS) List(name string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(name)
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

func (d *defaultFS) Remove(name string) error {
	return os.RemoveAll(name)
}

func (d *defaultFS) Rename(name string, newName string) error {
	return os.Rename(name, newName)
}

func (d *defaultFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
