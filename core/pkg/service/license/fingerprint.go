// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license

import (
	"encoding/hex"
	"net"
	"slices"

	"golang.org/x/crypto/argon2"
)

// fingerprintScheme is the version of the hashing rule readFingerprint implements.
const fingerprintScheme = 1

// fingerprintSalt separates these hashes from any other hash of a hardware address.
const fingerprintSalt = "synnax-license-fingerprint"

// readFingerprint hashes the hardware addresses of the machine's interfaces, skipping
// loopback and point-to-point interfaces and those without an address. The result is
// empty on a machine with no such interface. Argon2id makes recovering an address from
// its hash slow, since the space of addresses is small.
func readFingerprint() (Fingerprint, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	fingerprint := make(Fingerprint, 0, len(ifaces))
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 ||
			iface.Flags&net.FlagPointToPoint != 0 ||
			len(iface.HardwareAddr) == 0 {
			continue
		}
		sum := argon2.IDKey(
			[]byte(iface.HardwareAddr.String()),
			[]byte(fingerprintSalt),
			1,
			4*1024,
			1,
			32,
		)
		fingerprint = append(fingerprint, hex.EncodeToString(sum))
	}
	slices.Sort(fingerprint)
	return slices.Compact(fingerprint), nil
}

// Covers reports whether a license bound to the given hashes applies to this machine:
// an unbound license always does, and a bound one does when any hash belongs to the
// fingerprint. Hashes from a scheme this Core does not implement never match.
func (f Fingerprint) Covers(scheme uint8, hashes []string) bool {
	if len(hashes) == 0 {
		return true
	}
	if scheme != fingerprintScheme {
		return false
	}
	for _, hash := range hashes {
		if slices.Contains(f, hash) {
			return true
		}
	}
	return false
}
