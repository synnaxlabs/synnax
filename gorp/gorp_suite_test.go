// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"math"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	kvDB kv.DB
	db   *gorp.DB
)

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	kvDB = memkv.New()
	db = DeferClose(gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec)))
})

func TestGorp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gorp Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// leastAllocs returns the fewest allocations per run of f across several samples.
// Other goroutines only add to a sample, so the least sample is exact.
func leastAllocs(f func()) float64 {
	n := math.Inf(1)
	for range 5 {
		n = min(n, testing.AllocsPerRun(100, f))
	}
	return n
}
