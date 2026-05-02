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
// instead of misinterpreted. v1 keyed only the raw-content hash. v2
// stores a per-file Entry that tracks BOTH the raw plugin-output hash
// AND the canonical post-format hash, so sync can detect when its
// cached "this file is up to date" decision is invalidated by a
// formatter version bump or a hand edit.
const CacheVersion = 2

// Cache records, for each generated file, the SHA-256 of the raw plugin
// output AND the SHA-256 of the canonical post-format bytes that were
// last written. Sync's "skip this file" decision must verify BOTH:
// the plugin emitted the same raw bytes (so re-formatting would
// produce the same canonical bytes) AND the on-disk file currently
// hashes to that canonical value (so the user has not hand-edited the
// file and the formatter has not been version-bumped since the cache
// was written).
//
// Verifying only the raw hash is unsafe: any change in the formatter
// chain leaves stale on-disk bytes that sync would silently keep in
// place, while `oracle check` (which always re-runs the chain) would
// flag them as drift.
//
// The cache is keyed by repo-relative path. It lives at
// `<repoRoot>/.oracle/sync-cache.json` and is gitignored.
type Cache struct {
	path string
	mu   sync.Mutex
	data cacheFile
}

// Entry is the per file cache value: hashes of both the raw plugin
// output and the canonical formatted output.
type Entry struct {
	Raw       string `json:"raw"`
	Canonical string `json:"canonical"`
}

type cacheFile struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
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
			Entries: make(map[string]Entry),
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
	if f.Entries == nil {
		f.Entries = make(map[string]Entry)
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

// Lookup returns the cached Entry for a repo-relative path and whether
// one was present.
func (c *Cache) Lookup(repoRelPath string) (Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data.Entries[repoRelPath]
	return v, ok
}

// Put stores the Entry for a repo-relative path.
func (c *Cache) Put(repoRelPath string, entry Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data.Entries[repoRelPath] = entry
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

// EntryKeys returns the repo-relative paths of every cached entry. Used
// by check.OrphanGate to compare cached entries against the current
// run's produced set; not used during normal sync.
func (c *Cache) EntryKeys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.data.Entries))
	for k := range c.data.Entries {
		keys = append(keys, k)
	}
	return keys
}

// PruneTo drops every entry whose key is not in keep. This removes
// stale entries for files that are no longer generated.
func (c *Cache) PruneTo(keep set.Set[string]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.data.Entries {
		if !keep.Contains(k) {
			delete(c.data.Entries, k)
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
