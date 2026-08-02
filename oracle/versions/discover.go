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
	"sort"
	"strings"

	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/x/errors"
)

// Discover scans repoRoot's schema tree for version chains, keyed by live
// import path ("schemas/x/telem"). A chain directory may contain only
// vN.oracle files, and its versions must be dense from v0.
func Discover(repoRoot string) (map[string]Chain, error) {
	chains := make(map[string]Chain)
	schemasDir := filepath.Join(repoRoot, "schemas")
	domains, err := os.ReadDir(schemasDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", schemasDir)
	}
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
			chain, err := readChain(
				domain.Name(), resource.Name(),
				filepath.Join(versionsDir, resource.Name()),
			)
			if err != nil {
				return nil, err
			}
			chains[chain.LivePath()] = chain
		}
	}
	return chains, nil
}

// readChain reads one chain directory, validating that every entry is a
// version file and that the versions are dense from v0.
func readChain(domain, resource, dir string) (Chain, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Chain{}, errors.Wrapf(err, "failed to read %s", dir)
	}
	chain := Chain{Domain: domain, Resource: resource}
	for _, entry := range entries {
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
	for i, n := range chain.Numbers {
		if n != i {
			return Chain{}, errors.Newf(
				"%s versions must be dense from v0; missing v%d", dir, i,
			)
		}
	}
	return chain, nil
}
