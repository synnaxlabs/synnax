// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// RemovedFileName names the marker file of a chain whose resource is removed whole,
// stored data included. It is the only entry of its chain directory.
const RemovedFileName = "removed"

// Discover scans repoRoot's schema tree for version chains, keyed by live import path
// ("schemas/x/telem"). A chain directory may contain only vN.oracle files, or only the
// removed marker, which Discover skips.
func Discover(repoRoot string) (map[string]Chain, error) {
	dirs, err := readChainDirs(repoRoot)
	if err != nil {
		return nil, err
	}
	chains := make(map[string]Chain, len(dirs))
	for _, d := range dirs {
		if d.removed {
			continue
		}
		chain, err := readChain(d)
		if err != nil {
			return nil, err
		}
		chains[chain.LivePath()] = chain
	}
	return chains, nil
}

// Removed returns the live import paths of the chains that repoRoot marks as removed.
// It returns an error if a removed chain directory holds other files or if the live
// file of a removed chain still exists.
func Removed(repoRoot string) (set.Set[string], error) {
	dirs, err := readChainDirs(repoRoot)
	if err != nil {
		return nil, err
	}
	removed := make(set.Set[string])
	for _, d := range dirs {
		if d.removed {
			removed.Add(d.chain.LivePath())
		}
	}
	return removed, nil
}

// chainDir is one chain directory of a schema tree.
type chainDir struct {
	// chain is the chain the directory holds, without its version numbers.
	chain   Chain
	path    string
	entries []os.DirEntry
	removed bool
}

// readChainDirs returns every chain directory under repoRoot.
func readChainDirs(repoRoot string) ([]chainDir, error) {
	schemasDir := filepath.Join(repoRoot, "schemas")
	domains, err := os.ReadDir(schemasDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", schemasDir)
	}
	var dirs []chainDir
	for _, domain := range domains {
		if !domain.IsDir() {
			continue
		}
		versionsDir := filepath.Join(
			schemasDir, domain.Name(), paths.VersionsDirName,
		)
		resources, err := os.ReadDir(versionsDir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read %s", versionsDir)
		}
		for _, resource := range resources {
			if !resource.IsDir() {
				return nil, errors.Newf(
					"%s must contain only per-resource chain directories, found %s",
					versionsDir, resource.Name(),
				)
			}
			d := chainDir{
				chain: Chain{Domain: domain.Name(), Resource: resource.Name()},
				path:  filepath.Join(versionsDir, resource.Name()),
			}
			if d.entries, err = os.ReadDir(d.path); err != nil {
				return nil, errors.Wrapf(err, "failed to read %s", d.path)
			}
			if d.removed, err = isRemoved(repoRoot, d); err != nil {
				return nil, err
			}
			dirs = append(dirs, d)
		}
	}
	return dirs, nil
}

// isRemoved reports whether d holds the removed marker. It returns an error if the
// marker has company or if the live file of the chain still exists.
func isRemoved(repoRoot string, d chainDir) (bool, error) {
	if !slices.ContainsFunc(d.entries, func(e os.DirEntry) bool {
		return e.Name() == RemovedFileName
	}) {
		return false, nil
	}
	if len(d.entries) > 1 {
		return false, errors.Newf(
			"%s must contain only its %s marker", d.path, RemovedFileName,
		)
	}
	livePath := d.chain.LivePath()
	_, err := os.Stat(filepath.Join(repoRoot, livePath+".oracle"))
	if err == nil {
		return false, errors.Newf(
			"%s is marked removed, but %s.oracle exists", livePath, livePath,
		)
	}
	if !os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

// readChain reads the chain in d, validating that every entry is a version file and
// that no version is declared twice.
func readChain(d chainDir) (Chain, error) {
	chain, dir := d.chain, d.path
	for _, entry := range d.entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".oracle") {
			return Chain{}, errors.Newf(
				"%s must contain only vN.oracle files, found %s", dir, entry.Name(),
			)
		}
		_, n, ok := paths.VersionFile(filepath.Join(dir, entry.Name()))
		if !ok {
			return Chain{}, errors.Newf(
				"%s must contain only vN.oracle files, found %s", dir, entry.Name(),
			)
		}
		chain.Numbers = append(chain.Numbers, n)
	}
	if len(chain.Numbers) == 0 {
		return Chain{}, errors.Newf("%s declares no versions", dir)
	}
	sort.Ints(chain.Numbers)
	for i := 1; i < len(chain.Numbers); i++ {
		if chain.Numbers[i] == chain.Numbers[i-1] {
			return Chain{}, errors.Newf(
				"%s declares v%d twice", dir, chain.Numbers[i],
			)
		}
	}
	return chain, nil
}
