// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package codegen wraps external code generation tools (currently
// `buf generate`) with hash-based caching so generation only runs when
// its inputs change.
package codegen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"golang.org/x/sync/errgroup"
)

// BufGenerateStampKey is the cache key under which the proto-input
// stamp is stored.
const BufGenerateStampKey = "buf-generate"

// RunBufGenerate runs `buf generate` if and only if the hashed input
// (proto files + buf.yaml + buf.gen.yaml) has changed since the
// previous run. When changedProtos is non-empty, the run is scoped
// to those files via repeated `--path` flags so protoc only runs over
// the slice that changed instead of the entire repo. When the cache
// indicates no input change, the call is a no-op.
//
// changedProtos contains repo-relative paths to generated `.proto`
// files that the format step actually wrote (or that we already know
// must be regenerated). Pass nil to force a full repo regeneration.
// RunBufGenerateResult carries information the caller needs to log
// the outcome.
type RunBufGenerateResult struct {
	// Cached is true when the input-content stamp matched and no
	// `buf generate` invocation was needed.
	Cached bool
}

func RunBufGenerate(
	ctx context.Context,
	repoRoot string,
	changedProtos []string,
	cache *format.Cache,
) (RunBufGenerateResult, error) {
	stamp, err := bufInputStamp(ctx, repoRoot)
	if err != nil {
		return RunBufGenerateResult{}, errors.Wrap(err, "compute buf input stamp")
	}
	if cached, ok := cache.LookupStamp(BufGenerateStampKey); ok && cached == stamp && len(changedProtos) == 0 {
		return RunBufGenerateResult{Cached: true}, nil
	}
	args := []string{"generate"}
	for _, p := range changedProtos {
		args = append(args, "--path", p)
	}
	c := osexec.CommandContext(ctx, "buf", args...)
	c.Dir = repoRoot
	if out, err := c.CombinedOutput(); err != nil {
		return RunBufGenerateResult{}, errors.Wrapf(err, "buf generate failed: %s", string(out))
	}
	cache.PutStamp(BufGenerateStampKey, stamp)
	return RunBufGenerateResult{Cached: false}, nil
}

// bufInputStamp returns a content-derived stamp covering every input
// `buf generate` consults: the buf.yaml / buf.gen.yaml configs and
// every committed `.proto` file under repoRoot. We hash the file
// contents (not just mtimes) so a `git checkout` of identical bytes
// doesn't trigger an unnecessary regeneration.
func bufInputStamp(ctx context.Context, repoRoot string) (string, error) {
	configs, err := readIfExists(
		filepath.Join(repoRoot, "buf.yaml"),
		filepath.Join(repoRoot, "buf.gen.yaml"),
		filepath.Join(repoRoot, "buf.lock"),
	)
	if err != nil {
		return "", err
	}
	protos, err := findProtoFiles(repoRoot)
	if err != nil {
		return "", errors.Wrap(err, "scan proto files")
	}
	sort.Strings(protos)

	hashes := make([]string, len(protos))
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(8)
	for i, p := range protos {
		eg.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return errors.Wrapf(err, "read %s", p)
			}
			h := sha256.Sum256(b)
			rel, _ := filepath.Rel(repoRoot, p)
			hashes[i] = rel + ":" + hex.EncodeToString(h[:])
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return "", err
	}

	rollup := sha256.New()
	for _, c := range configs {
		rollup.Write(c)
		rollup.Write([]byte{0})
	}
	for _, h := range hashes {
		rollup.Write([]byte(h))
		rollup.Write([]byte{0})
	}
	return hex.EncodeToString(rollup.Sum(nil)), nil
}

func readIfExists(paths ...string) ([][]byte, error) {
	out := make([][]byte, len(paths))
	for i, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errors.Wrapf(err, "read %s", p)
		}
		out[i] = b
	}
	return out, nil
}

// findProtoFiles walks repoRoot for .proto files, skipping common
// vendor / build / .git directories.
func findProtoFiles(repoRoot string) ([]string, error) {
	return walkExtensions(repoRoot, []string{".proto"}, nil)
}

// findPbGoFiles walks repoRoot for the .pb.go and _grpc.pb.go outputs
// `buf generate` emits next to their .proto sources, skipping the same
// vendor / build / .git directories findProtoFiles skips.
func findPbGoFiles(repoRoot string) ([]string, error) {
	return walkExtensions(repoRoot, []string{".go"}, func(name string) bool {
		return strings.HasSuffix(name, ".pb.go") || strings.HasSuffix(name, "_grpc.pb.go")
	})
}

func walkExtensions(repoRoot string, exts []string, namePred func(string) bool) ([]string, error) {
	skipDir := set.New(".git", "node_modules", "dist", "build", "target", "vendor", ".oracle")
	var out []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir.Contains(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		matchExt := false
		for _, e := range exts {
			if e == ext {
				matchExt = true
				break
			}
		}
		if !matchExt {
			return nil
		}
		if namePred != nil && !namePred(filepath.Base(path)) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FormatBufOutputs runs the configured formatter chain over every
// .pb.go and _grpc.pb.go file `buf generate` produced in the repo,
// writing back any whose canonical bytes differ. This restores the
// license header (and re-applies gofmt) to protoc's raw output, which
// otherwise lands on disk header-less.
//
// Returns the count of files actually rewritten. The walk is bounded
// by the same skip-list as findProtoFiles, so vendor / .oracle /
// build directories are not touched.
//
// Idempotent: re-running over already-canonical files is a no-op
// because the license formatter detects an existing header (any year)
// and returns the content unchanged, and gofmt is stable.
func FormatBufOutputs(
	ctx context.Context,
	repoRoot string,
	formatters *format.Registry,
	workers int,
) (int, error) {
	files, err := findPbGoFiles(repoRoot)
	if err != nil {
		return 0, errors.Wrap(err, "scan .pb.go files")
	}
	if len(files) == 0 {
		return 0, nil
	}

	batch := make([]format.File, len(files))
	for i, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			return 0, errors.Wrapf(err, "read %s", path)
		}
		batch[i] = format.File{Path: path, Content: raw}
	}

	formatted, err := formatters.FormatBatch(ctx, batch, workers)
	if err != nil {
		return 0, errors.Wrap(err, "format buf outputs")
	}

	written := 0
	for i, f := range formatted {
		if string(batch[i].Content) == string(f.Content) {
			continue
		}
		if err := os.WriteFile(files[i], f.Content, 0644); err != nil {
			return written, errors.Wrapf(err, "write %s", files[i])
		}
		written++
	}
	return written, nil
}
