// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package udp

import (
	"net/netip"

	"github.com/synnaxlabs/x/validate"
)

// CheckLocal refuses a local address that is not an IP address, and a local port of 0
// on a device with no remote, as no sender could reach the port that the operating
// system picks.
func (p Properties) CheckLocal() error {
	v := validate.New("Properties")
	_, err := netip.ParseAddr(p.LocalAddress)
	v.Ternary(
		"local_address",
		p.LocalAddress != "" && err != nil,
		"must be an IP address",
	)
	v.Ternary(
		"local_port",
		p.LocalPort == 0 && p.Remote == nil,
		"required when the device has no remote",
	)
	return v.Error()
}
