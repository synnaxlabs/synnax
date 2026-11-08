// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

const archiveExt = ".tar"

// writeArchive writes the directories dirs and the files files of fsys to a new tar
// archive at name in fsys, then syncs it once. The directory entries keep an empty
// directory in the archive.
func writeArchive(fsys xfs.FS, name string, dirs, files []string) (err error) {
	f, err := fsys.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	tw := tar.NewWriter(f)
	for _, dir := range dirs {
		if err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     dir + "/",
			Mode:     0o700,
		}); err != nil {
			return err
		}
	}
	for _, file := range files {
		if err = writeEntry(tw, fsys, file); err != nil {
			return err
		}
	}
	if err = tw.Close(); err != nil {
		return err
	}
	return f.Sync()
}

func writeEntry(tw *tar.Writer, fsys xfs.FS, name string) (err error) {
	f, err := fsys.Open(name, os.O_RDONLY)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if err = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Mode:     0o600,
		Size:     info.Size(),
	}); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

// extractArchive writes each entry of the tar archive at name in fsys to dst. It
// returns an error for an entry that is not a directory or a file, or whose name is not
// a local path.
func extractArchive(fsys xfs.FS, name string, dst xfs.FS) (err error) {
	f, err := fsys.Open(name, os.O_RDONLY)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		entry := strings.TrimSuffix(hdr.Name, "/")
		if !filepath.IsLocal(entry) {
			return errors.Newf("archive entry %q is not a local path", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			_, err = dst.Sub(entry)
		case tar.TypeReg:
			err = extractFile(tr, dst, entry)
		default:
			err = errors.Newf("archive entry %q is not a directory or a file", hdr.Name)
		}
		if err != nil {
			return err
		}
	}
}

func extractFile(r io.Reader, dst xfs.FS, name string) (err error) {
	if _, err = dst.Sub(path.Dir(name)); err != nil {
		return err
	}
	f, err := dst.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	_, err = io.Copy(f, r)
	return err
}

// listFiles returns the paths in fsys of the files in the directory dir and in its
// subdirectories.
func listFiles(fsys xfs.FS, dir string) ([]string, error) {
	infos, err := fsys.List(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, info := range infos {
		name := path.Join(dir, info.Name())
		if !info.IsDir() {
			files = append(files, name)
			continue
		}
		nested, err := listFiles(fsys, name)
		if err != nil {
			return nil, err
		}
		files = append(files, nested...)
	}
	return files, nil
}
