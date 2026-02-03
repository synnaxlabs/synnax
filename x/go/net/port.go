// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package net

import "net"

// FindOpenPort finds an open port on the local machine.
func FindOpenPort() (int, error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", a)
	defer func() {
		_ = l.Close()
	}()
	if err != nil {
		return 0, err
	}
	return l.Addr().(*net.TCPAddr).Port, nil
}
