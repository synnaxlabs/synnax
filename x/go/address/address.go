// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package address provides utilities for working with address types of the format
// host:port.
package address

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/uuid"
)

// Address represents an addressable item in a network.
type Address string

// Newf creates a new address with the given format.
func Newf(format string, args ...any) Address {
	return Address(fmt.Sprintf(format, args...))
}

// String implements fmt.Stringer.
func (a Address) String() string { return string(a) }

// PortString returns a string in the format ":port" where port is the port portion
// of the address.
// For example, Address("localhost:9090").PortString() would return ":9090".
func (a Address) PortString() string {
	parts := strings.Split(string(a), ":")
	if len(parts) != 2 {
		return ""
	}
	return ":" + parts[1]
}

// Port returns the port portion of the address.
// For example, Address("localhost:9090").Port() would return 9090.
func (a Address) Port() int {
	parts := strings.Split(string(a), ":")
	if len(parts) != 2 {
		return 0
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return port
}

// Host returns the host portion of the address.
// For example, Address("localhost:9090").Host() would return "localhost".
func (a Address) Host() string {
	parts := strings.Split(string(a), ":")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// Rand returns a random address.
func Rand() Address {
	return Address(uuid.New().String())
}
