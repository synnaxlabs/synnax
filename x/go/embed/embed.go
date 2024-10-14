package embed

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

func Extract(
	emb embed.FS,
	dir string,
	dirPerm os.FileMode,
	filePerm os.FileMode,
) (err error) {
	if err = os.MkdirAll(dir, dirPerm); err != nil {
		return
	}
	return fs.WalkDir(emb, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := emb.ReadFile(path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dir, path)
		if err = os.MkdirAll(filepath.Dir(destPath), dirPerm); err != nil {
			return err
		}
		return os.WriteFile(destPath, data, filePerm)
	})
}
