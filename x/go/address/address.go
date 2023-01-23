// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package address

import (
	"fmt"
	"strings"
)

type Address string

func Newf(format string, args ...any) Address {
	return Address(fmt.Sprintf(format, args...))
}

func (a Address) String() string { return string(a) }

func (a Address) PortString() string {
	str := strings.Split(string(a), ":")
	return ":" + str[1]
}

func (a Address) HostString() string {
	str := strings.Split(string(a), ":")
	return str[0]
}

type Addressable interface {
	Address() Address
}
