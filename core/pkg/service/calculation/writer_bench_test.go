// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkCreateOutput measures a create of one calculation and the output it makes,
// one transaction each.
func BenchmarkCreateOutput(b *testing.B) {
	gomega.RegisterTestingT(b)
	node := testutil.OpenNode(b.Context())
	statuses, err := status.OpenService(b.Context(), status.ServiceConfig{DB: node.DB})
	if err != nil {
		b.Fatalf("failed to open status service: %v", err)
	}
	channels, err := channel.OpenService(b.Context(), channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statuses,
	})
	if err != nil {
		b.Fatalf("failed to open channel service: %v", err)
	}
	svc, err := calculation.OpenService(b.Context(), calculation.ServiceConfig{
		DB:      node.DB,
		Channel: channels,
	})
	if err != nil {
		b.Fatalf("failed to open calculation service: %v", err)
	}
	b.Cleanup(func() {
		if err := (io.MultiCloser{node, svc, channels, statuses}).Close(); err != nil {
			b.Errorf("failed to close: %v", err)
		}
	})
	input := channel.Channel{
		Name:     "bench_input",
		DataType: telem.Float32T,
		Virtual:  true,
	}
	if err := node.DB.WithTx(b.Context(), func(tx gorp.Tx) error {
		return channels.NewWriter(tx).Create(b.Context(), &input)
	}); err != nil {
		b.Fatalf("failed to create input: %v", err)
	}
	names := make([]string, b.N)
	for i := range names {
		names[i] = fmt.Sprintf("bench_output_%d", i)
	}
	runtime.GC()
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		c := calculation.Calculation{Expression: "return bench_input * 2"}
		if err := node.DB.WithTx(b.Context(), func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).CreateOutput(b.Context(), &c, names[i], nil)
			return err
		}); err != nil {
			b.Fatalf("failed to create calculation: %v", err)
		}
	}
}
