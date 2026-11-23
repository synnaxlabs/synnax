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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
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
	// Truncated is true when the listing left out entries, past the first maxEntries
	// or past the size of an answer.
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

// exec answers a list command with its listing.
func (b browser) exec(
	ctx context.Context,
	cmd task.Command,
) (msgpack.EncodedJSON, error) {
	if cmd.Type != listCommand {
		return nil, driver.ErrUnsupportedCommand
	}
	var args struct {
		Path string `json:"path"`
	}
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return nil, failure("The path must be a string.")
	}
	l, err := b.list(ctx, args.Path)
	if err != nil {
		return nil, err
	}
	return msgpack.NewEncodedJSON(l)
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
	resolved, err := filepath.EvalSymlinks(dir)
	l.Secret = err != nil || secrets.holds(resolved, "")
	for _, e := range found[:min(len(found), maxEntries)] {
		if ctx.Err() != nil {
			return listing{}, failure(
				"The Core did not finish the list of %s in time.",
				dir,
			)
		}
		target, err := filepath.EvalSymlinks(e.Path)
		if l.Secret || err != nil || secrets.holds(target, resolved) {
			l.Entries = append(l.Entries, e)
			continue
		}
		if e, ok := describe(e, target); ok {
			l.Entries = append(l.Entries, e)
		}
	}
	entries := l.Entries
	l.Entries = []entry{}
	kept, truncated, err := driver.Fit(l, [][]entry{entries})
	if err != nil {
		return listing{}, err
	}
	l.Entries, l.Truncated = kept[0], l.Truncated || truncated
	return l, nil
}

// readDir returns the files and folders in dir, folders first, then files, each by
// name. It follows links, and leaves out a link that leads nowhere and an entry whose
// name is not valid UTF-8.
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
func readHead(path string) ([]byte, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, maxCertificateSize+1))
	if err = errors.Combine(err, f.Close()); err != nil {
		return nil, err
	}
	return data, nil
}

func (b browser) top() (listing, error) {
	rootDirs, err := roots()
	if err != nil {
		return listing{}, failure("Failed to read the drives: %s.", err)
	}
	l := listing{SecretsDir: b.secretsDir, Entries: []entry{}}
	add := func(name, path string) {
		if path == "" || slices.ContainsFunc(
			l.Entries,
			func(e entry) bool { return e.Path == path },
		) {
			return
		}
		l.Entries = append(
			l.Entries,
			entry{Name: name, Path: path, Kind: kindDirectory},
		)
	}
	for _, root := range rootDirs {
		add(root, root)
	}
	add(filepath.Base(b.home), b.home)
	add(filepath.Base(b.secretsDir), b.secretsDir)
	return l, nil
}

// secrets is the secrets folder of the Core's rack, as it is at the start of a list.
type secrets struct {
	// info is nil when no folder is set.
	info fs.FileInfo
	// unread is true when the folder is set but cannot be read.
	unread bool
}

func (b browser) secrets() secrets {
	if b.secretsDir == "" {
		return secrets{}
	}
	info, err := os.Stat(b.secretsDir)
	if err != nil {
		return secrets{unread: true}
	}
	return secrets{info: info}
}

// holds reports whether path, with its links resolved, is in the secrets folder. It
// compares the identity of path and of each parent with the folder, up to checked, a
// folder known to be outside it. It returns true when the folder, path, or a parent
// cannot be read.
func (s secrets) holds(path, checked string) bool {
	if s.unread {
		return true
	}
	if s.info == nil {
		return false
	}
	for ; path != checked; path = filepath.Dir(path) {
		info, err := os.Stat(path)
		if err != nil || os.SameFile(info, s.info) {
			return true
		}
		if filepath.Dir(path) == path {
			return false
		}
	}
	return false
}
