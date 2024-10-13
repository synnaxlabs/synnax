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
) (string, error) {
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return "", err
	}
	err = fs.WalkDir(emb, ".", func(path string, d fs.DirEntry, err error) error {
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
		if err = os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
	})
	return dir, err
}
