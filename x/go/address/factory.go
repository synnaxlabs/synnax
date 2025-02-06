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
	"strconv"
	"sync"
)

type Factory struct {
	Host      string
	PortStart int
	mu        sync.Mutex
	addresses []Address
}

func (f *Factory) Next() Address {
	f.mu.Lock()
	defer f.mu.Unlock()
	addr := Address(f.Host + ":" + strconv.Itoa(f.PortStart+len(f.addresses)))
	f.addresses = append(f.addresses, addr)
	return addr
}

func (f *Factory) NextN(n int) (addresses []Address) {
	for i := 0; i < n; i++ {
		addresses = append(addresses, f.Next())
	}
	return addresses
}

func NewLocalFactory(portStart int) *Factory {
	return &Factory{Host: "localhost", PortStart: portStart}
}
