// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// CacheVersion is the on-disk cache schema version. Bump this whenever
// the cache file format changes incompatibly so stale caches are dropped
// instead of misinterpreted.
const CacheVersion = 1

// Cache records the SHA-256 of the raw (pre-format) content oracle emitted
// for each generated file in the previous sync. When a subsequent sync
// produces the same raw bytes for the same path, the on-disk file is
// presumed to already be the canonical formatted form and the format +
// write steps can be skipped.
//
// The cache is keyed by repo-relative path. It lives at
// `<repoRoot>/.oracle/sync-cache.json` and is gitignored.
type Cache struct {
	path string
	mu   sync.Mutex
	data cacheFile
}

type cacheFile struct {
	Version int               `json:"version"`
	Hashes  map[string]string `json:"hashes"`
	// Stamps holds opaque per-key stamps for non-file inputs (e.g. the
	// hashed proto-tree input for buf generate). Not used for sync skip
	// decisions on individual files.
	Stamps map[string]string `json:"stamps"`
}

// LoadCache reads the cache from disk. A missing or unreadable cache file
// produces an empty cache without an error so first-run syncs work; a
// version mismatch likewise yields an empty cache so an incompatible
// upgrade discards stale entries instead of misapplying them.
func LoadCache(repoRoot string) *Cache {
	c := &Cache{
		path: filepath.Join(repoRoot, ".oracle", "sync-cache.json"),
		data: cacheFile{
			Version: CacheVersion,
			Hashes:  make(map[string]string),
			Stamps:  make(map[string]string),
		},
	}
	raw, err := os.ReadFile(c.path)
	if err != nil {
		return c
	}
	var f cacheFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return c
	}
	if f.Version != CacheVersion {
		return c
	}
	if f.Hashes == nil {
		f.Hashes = make(map[string]string)
	}
	if f.Stamps == nil {
		f.Stamps = make(map[string]string)
	}
	c.data = f
	return c
}

// Hash returns the canonical SHA-256 hex digest of content used as the
// cache value for a generated file.
func Hash(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// LookupRaw returns the cached raw-content hash for a repo-relative path
// and whether one was present.
func (c *Cache) LookupRaw(repoRelPath string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data.Hashes[repoRelPath]
	return v, ok
}

// PutRaw stores the raw-content hash for a repo-relative path.
func (c *Cache) PutRaw(repoRelPath, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.Hashes[repoRelPath] = hash
}

// LookupStamp returns an opaque stamp for a non-file input key.
func (c *Cache) LookupStamp(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data.Stamps[key]
	return v, ok
}

// PutStamp stores an opaque stamp for a non-file input key.
func (c *Cache) PutStamp(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.Stamps[key] = value
}

// PruneRawTo drops every raw-content entry whose key is not in keep.
// This removes stale entries for files that are no longer generated.
func (c *Cache) PruneRawTo(keep set.Set[string]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.data.Hashes {
		if !keep.Contains(k) {
			delete(c.data.Hashes, k)
		}
	}
}

// Save persists the cache to disk, creating the parent directory if
// needed.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return errors.Wrap(err, "create cache dir")
	}
	raw, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal cache")
	}
	if err := os.WriteFile(c.path, raw, 0644); err != nil {
		return errors.Wrap(err, "write cache")
	}
	return nil
}
