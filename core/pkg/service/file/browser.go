// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package file

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

const (
	// maxEntries is the most entries that a listing keeps.
	maxEntries = 1000
	// maxCertificateSize is the largest file that a listing reads a certificate from.
	maxCertificateSize = 1 << 20
)

// certificateExtensions are the extensions, in lower case, of the files that a listing
// reads a certificate from.
var certificateExtensions = []string{".pem", ".crt", ".cer", ".der", ".cert"}

type kind string

const (
	kindFile      kind = "file"
	kindDirectory kind = "directory"
)

// listing is the data of the reply to a list command.
type listing struct {
	// Path is the listed folder, or empty at the top level.
	Path string `json:"path"`
	// Parent is the folder that holds Path, or empty at a root and at the top level.
	Parent string `json:"parent"`
	// SecretsDir is the secrets folder of the Core's rack, or empty when none is set.
	SecretsDir string `json:"secrets_dir"`
	// Secret is true when Path is in the secrets folder.
	Secret bool `json:"secret"`
	// Truncated is true when the listing kept only the first maxEntries entries.
	Truncated bool    `json:"truncated"`
	Entries   []entry `json:"entries"`
}

// entry is a file or folder in a listing. Size and Modified are nil at the top level
// and in the secrets folder, and Certificate is nil unless the file holds one.
type entry struct {
	Name        string               `json:"name"`
	Path        string               `json:"path"`
	Kind        kind                 `json:"kind"`
	Size        *int64               `json:"size,omitzero"`
	Modified    *telem.TimeStamp     `json:"modified,omitzero"`
	Certificate *certificate.Details `json:"certificate,omitzero"`
}

// browser lists the folders of the Core's computer.
type browser struct {
	// home is the home folder of the user that runs the Core, or empty when unknown.
	home string
	// secretsDir is the absolute secrets folder, or empty when none is set.
	secretsDir string
}

func failure(format string, args ...any) driver.Failure {
	return driver.Failure{
		Message:     "Failed to list files",
		Description: fmt.Sprintf(format, args...),
	}
}

func readFailure(err error, dir string) driver.Failure {
	if errors.Is(err, fs.ErrPermission) {
		return failure("The Core cannot read %s.", dir)
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return failure("Failed to read %s: %s.", dir, err)
}

// list lists the folder at path, or the roots, the home folder, and the secrets folder
// when path is empty. It returns a driver.Failure that names the path when path is
// relative, is not a folder, or cannot be read.
func (b browser) list(ctx context.Context, path string) (listing, error) {
	if path == "" {
		return b.top()
	}
	if !filepath.IsAbs(path) {
		return listing{}, failure("%s is not an absolute path.", path)
	}
	dir := filepath.Clean(path)
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return listing{}, failure("%s does not exist.", dir)
	case err != nil:
		return listing{}, readFailure(err, dir)
	case !info.IsDir():
		return listing{}, failure("%s is not a folder.", dir)
	}
	found, err := readDir(dir)
	if err != nil {
		return listing{}, err
	}
	l := listing{
		Path:       dir,
		SecretsDir: b.secretsDir,
		Truncated:  len(found) > maxEntries,
		Entries:    make([]entry, 0, min(len(found), maxEntries)),
	}
	if parent := filepath.Dir(dir); parent != dir {
		l.Parent = parent
	}
	secrets := b.secrets()
	l.Secret = secrets.holds(dir)
	for _, e := range found[:min(len(found), maxEntries)] {
		if err := ctx.Err(); err != nil {
			return listing{}, err
		}
		// The check and the reads use one resolved path, so a link that changes
		// between them cannot lead the reads into the secrets folder.
		target, err := filepath.EvalSymlinks(e.Path)
		if l.Secret || err != nil || secrets.holds(target) {
			l.Entries = append(l.Entries, e)
			continue
		}
		if e, ok := describe(e, target); ok {
			l.Entries = append(l.Entries, e)
		}
	}
	return l, nil
}

// readDir returns the files and folders in dir, folders first, then files, each by
// name.
// It follows links, and leaves out a link that leads nowhere and an entry whose name is
// not valid UTF-8.
func readDir(dir string) ([]entry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, readFailure(err, dir)
	}
	found := make([]entry, 0, len(dirEntries))
	for _, d := range dirEntries {
		if !utf8.ValidString(d.Name()) {
			continue
		}
		e := entry{Name: d.Name(), Path: filepath.Join(dir, d.Name())}
		mode := d.Type()
		if mode&fs.ModeSymlink != 0 {
			target, err := os.Stat(e.Path)
			if err != nil {
				continue
			}
			mode = target.Mode().Type()
		}
		switch {
		case mode.IsDir():
			e.Kind = kindDirectory
		case mode.IsRegular():
			e.Kind = kindFile
		default:
			continue
		}
		found = append(found, e)
	}
	slices.SortStableFunc(found, func(a, b entry) int {
		if a.Kind != b.Kind {
			if a.Kind == kindDirectory {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return found, nil
}

// describe adds the size, the modified time, and the certificate of the file or folder
// at target to e. It returns false when target cannot be read.
func describe(e entry, target string) (entry, bool) {
	info, err := os.Stat(target)
	if err != nil {
		return e, false
	}
	var size int64
	if e.Kind == kindFile {
		size = info.Size()
	}
	modified := telem.NewTimeStamp(info.ModTime())
	e.Size, e.Modified = &size, &modified
	ext := strings.ToLower(filepath.Ext(e.Name))
	if e.Kind == kindFile && size <= maxCertificateSize &&
		slices.Contains(certificateExtensions, ext) {
		e.Certificate = readCertificate(target)
	}
	return e, true
}

// readCertificate returns the details of the certificate in the file at path, or nil
// when the file holds no certificate.
func readCertificate(path string) *certificate.Details {
	data, err := readHead(path)
	if err != nil || len(data) > maxCertificateSize {
		return nil
	}
	details, err := certificate.ParseDetails(data)
	if err != nil {
		return nil
	}
	return &details
}

// readHead returns at most the first maxCertificateSize+1 bytes of the file at path.
func readHead(path string) (data []byte, err error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	return io.ReadAll(io.LimitReader(f, maxCertificateSize+1))
}

func (b browser) top() (listing, error) {
	rootDirs, err := roots()
	if err != nil {
		return listing{}, failure("Failed to read the drives: %s.", err)
	}
	l := listing{SecretsDir: b.secretsDir, Entries: []entry{}}
	add := func(name, path string) {
		l.Entries = append(
			l.Entries,
			entry{Name: name, Path: path, Kind: kindDirectory},
		)
	}
	for _, root := range rootDirs {
		add(root, root)
	}
	for _, dir := range []string{b.home, b.secretsDir} {
		if dir != "" {
			add(filepath.Base(dir), dir)
		}
	}
	return l, nil
}

// secrets is the secrets folder of one listing, by its name and with its links
// resolved.
type secrets struct {
	dir string
	// resolved is empty when dir does not resolve.
	resolved string
}

func (b browser) secrets() secrets {
	s := secrets{dir: b.secretsDir}
	if s.dir == "" {
		return s
	}
	if resolved, err := filepath.EvalSymlinks(s.dir); err == nil {
		s.resolved = resolved
	}
	return s
}

// holds reports whether path is in the folder, by its name or with its links resolved.
// It returns true when the folder or path does not resolve.
func (s secrets) holds(path string) bool {
	if s.dir == "" {
		return false
	}
	if s.resolved == "" || within(path, s.dir) {
		return true
	}
	resolved, err := filepath.EvalSymlinks(path)
	return err != nil || within(resolved, s.resolved)
}

// within reports whether path is dir or is in dir. Both must be clean and absolute.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
