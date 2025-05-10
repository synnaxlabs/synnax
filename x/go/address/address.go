// Copyright 2025 Synnax Labs, Inc.
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
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Address string

func Newf(format string, args ...any) Address {
	return Address(fmt.Sprintf(format, args...))
}

func (a Address) String() string { return string(a) }

func (a Address) PortString() string {
	parts := strings.Split(string(a), ":")
	return ":" + parts[1]
}

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

func (a Address) Host() string {
	split := strings.Split(string(a), ":")
	if len(split) == 0 {
		return ""
	}
	return split[0]
}

func Rand() Address {
	return Address(uuid.New().String())
}
