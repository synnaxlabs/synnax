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

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/x/errors"
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
func RunBufGenerate(
	ctx context.Context,
	repoRoot string,
	changedProtos []string,
	cache *format.Cache,
) error {
	stamp, err := bufInputStamp(ctx, repoRoot)
	if err != nil {
		return errors.Wrap(err, "compute buf input stamp")
	}
	if cached, ok := cache.LookupStamp(BufGenerateStampKey); ok && cached == stamp && len(changedProtos) == 0 {
		return nil
	}
	args := []string{"generate"}
	for _, p := range changedProtos {
		args = append(args, "--path", p)
	}
	c := osexec.CommandContext(ctx, "buf", args...)
	c.Dir = repoRoot
	if out, err := c.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "buf generate failed: %s", string(out))
	}
	cache.PutStamp(BufGenerateStampKey, stamp)
	return nil
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
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(8)
	for i, p := range protos {
		i, p := i, p
		eg.Go(func() error {
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
	skipDir := map[string]struct{}{
		".git":         {},
		"node_modules": {},
		"dist":         {},
		"build":        {},
		"target":       {},
		"vendor":       {},
		".oracle":      {},
	}
	var out []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, skip := skipDir[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".proto" {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
