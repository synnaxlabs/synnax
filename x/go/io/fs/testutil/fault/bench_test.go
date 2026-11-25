// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fault_test

import (
	"os"
	"testing"

	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

// BenchmarkSync measures a Sync after a write of one block to a file of 64 MiB.
func BenchmarkSync(b *testing.B) {
	const size = 64 << 20
	fs, err := fault.New(fault.Config{})
	if err != nil {
		b.Fatal(err)
	}
	f, err := fs.Open("f.bin", os.O_CREATE|os.O_WRONLY)
	if err != nil {
		b.Fatal(err)
	}
	if err = f.Truncate(size); err != nil {
		b.Fatal(err)
	}
	if err = f.Sync(); err != nil {
		b.Fatal(err)
	}
	block := make([]byte, 4096)
	for i := 0; b.Loop(); i++ {
		if _, err = f.WriteAt(block, int64(i%(size/4096))*4096); err != nil {
			b.Fatal(err)
		}
		if err = f.Sync(); err != nil {
			b.Fatal(err)
		}
	}
	if err = f.Close(); err != nil {
		b.Fatal(err)
	}
}
