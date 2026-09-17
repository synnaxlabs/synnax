// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"slices"
)

// Host identifies the machine the Core runs on: the sorted SHA-256 hex digests of
// every physical network interface's hardware address.
type Host []string

// hostScheme is the version of the hashing rule readHost implements.
const hostScheme = 1

// readHost hashes the hardware addresses of the machine's interfaces, skipping
// loopback and point-to-point interfaces and those without an address. The result is
// empty on a machine with no such interface.
func readHost() (Host, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	host := make(Host, 0, len(ifaces))
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 ||
			iface.Flags&net.FlagPointToPoint != 0 ||
			len(iface.HardwareAddr) == 0 {
			continue
		}
		sum := sha256.Sum256([]byte(iface.HardwareAddr.String()))
		host = append(host, hex.EncodeToString(sum[:]))
	}
	slices.Sort(host)
	return slices.Compact(host), nil
}

// Covers reports whether a grant bound to the given hashes applies to the host: an
// unbound grant always does, and a bound one does when any hash belongs to the host.
// Hashes from a scheme this Core does not implement never match.
func (h Host) Covers(scheme uint8, hashes []string) bool {
	if len(hashes) == 0 {
		return true
	}
	if scheme != hostScheme {
		return false
	}
	for _, hash := range hashes {
		if slices.Contains(h, hash) {
			return true
		}
	}
	return false
}
