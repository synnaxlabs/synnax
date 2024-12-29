// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package embed

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// Extract extracts the contents of an embed.FS to a directory on disk.
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
