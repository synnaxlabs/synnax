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

// benchServices opens a calculation service over a fresh node, with an input channel
// named bench_input.
func benchServices(b *testing.B) (*calculation.Service, *gorp.DB) {
	b.Helper()
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
	return svc, node.DB
}

// createOutputs creates count calculations of bench_input with outputs, and returns
// their keys.
func createOutputs(
	b *testing.B,
	svc *calculation.Service,
	db *gorp.DB,
	count int,
) []calculation.Key {
	b.Helper()
	keys := make([]calculation.Key, count)
	for i := range keys {
		c := calculation.Calculation{Expression: "return bench_input * 2"}
		if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).CreateOutput(
				b.Context(),
				&c,
				fmt.Sprintf("bench_output_%d", i),
				nil,
			)
			return err
		}); err != nil {
			b.Fatalf("failed to create calculation: %v", err)
		}
		keys[i] = c.Key
	}
	return keys
}

// BenchmarkCreateOutput measures a create of one calculation and the output it makes,
// one transaction each.
func BenchmarkCreateOutput(b *testing.B) {
	svc, db := benchServices(b)
	names := make([]string, b.N)
	for i := range names {
		names[i] = fmt.Sprintf("bench_output_%d", i)
	}
	runtime.GC()
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		c := calculation.Calculation{Expression: "return bench_input * 2"}
		if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).CreateOutput(b.Context(), &c, names[i], nil)
			return err
		}); err != nil {
			b.Fatalf("failed to create calculation: %v", err)
		}
	}
}

// BenchmarkDelete measures a delete of one calculation, one transaction each.
func BenchmarkDelete(b *testing.B) {
	svc, db := benchServices(b)
	keys := createOutputs(b, svc, db, b.N)
	runtime.GC()
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(b.Context(), keys[i])
		}); err != nil {
			b.Fatalf("failed to delete calculation: %v", err)
		}
	}
}

// setOf returns an action that sets path to value.
func setOf(path string, value any) calculation.Action {
	return calculation.NewSetAction(calculation.SetPayload{
		Path:  []string{path},
		Value: value,
	})
}

// BenchmarkApply measures an edit of the expression or the operations of count
// calculations in one transaction, and an edit that changes neither. Each iteration
// alternates between two values.
func BenchmarkApply(b *testing.B) {
	avg := calculation.AvgOperation{Duration: telem.Second}
	edits := map[string][2]calculation.Action{
		"expression": {
			setOf("expression", "return bench_input * 3"),
			setOf("expression", "return bench_input * 2"),
		},
		"operations": {
			setOf("operations", []calculation.Operation{{Variant: avg}}),
			setOf("operations", []calculation.Operation{}),
		},
		"unchanged": {
			setOf("expression", "return bench_input * 2"),
			setOf("expression", "return bench_input * 2"),
		},
	}
	for _, name := range []string{"expression", "operations", "unchanged"} {
		for _, count := range []int{1, 100} {
			b.Run(fmt.Sprintf("%s/count=%d", name, count), func(b *testing.B) {
				svc, db := benchServices(b)
				keys := createOutputs(b, svc, db, count)
				runtime.GC()
				b.ReportAllocs()
				b.ResetTimer()
				for i := range b.N {
					edit := edits[name][i%2]
					if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
						w := svc.NewWriter(tx)
						for _, key := range keys {
							if _, err := w.Apply(b.Context(), key, edit); err != nil {
								return err
							}
						}
						return nil
					}); err != nil {
						b.Fatalf("failed to apply: %v", err)
					}
				}
			})
		}
	}
}
