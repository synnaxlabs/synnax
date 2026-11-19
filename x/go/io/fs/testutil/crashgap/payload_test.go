// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crashgap_test

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// altered holds every commit in memory and changes what it returns with alter.
type altered struct {
	records  []crash.Record
	alter    func([]crash.Record) []crash.Record
	closeErr error
}

func (a *altered) Commit(
	_ context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	a.records = append(
		a.records,
		crash.Record{ID: r.ID, Payload: slices.Clone(r.Payload)},
	)
	return want, nil
}

func (a *altered) Recovered(context.Context) ([]crash.Record, error) {
	out := slices.Clone(a.records)
	for i := range out {
		out[i].Payload = slices.Clone(out[i].Payload)
	}
	return a.alter(out), nil
}

func (a *altered) Close() error { return a.closeErr }

func live(alter func([]crash.Record) []crash.Record) crash.Config {
	return crash.Config{
		RunKey: 1,
		Open: func(context.Context, xfs.FS) (crash.Engine, error) {
			return &altered{alter: alter}, nil
		},
		Commits: 4,
		Stride:  1,
		Profile: crash.Strict,
	}
}

func rule(err error) crash.Rule {
	GinkgoHelper()
	var v *crash.Violation
	Expect(errors.As(err, &v)).To(BeTrue(), "%v", err)
	return v.Rule
}

var _ = Describe("Payload checks", func() {
	It("Should catch a payload with one flipped bit", func(ctx SpecContext) {
		_, err := crash.Run(ctx, live(func(rs []crash.Record) []crash.Record {
			rs[1].Payload[0] ^= 1
			return rs
		}))
		Expect(rule(err)).To(Equal(crash.R2))
	})

	It("Should catch two commits with their payloads swapped", func(ctx SpecContext) {
		_, err := crash.Run(ctx, live(func(rs []crash.Record) []crash.Record {
			rs[0].Payload, rs[1].Payload = rs[1].Payload, rs[0].Payload
			return rs
		}))
		Expect(rule(err)).To(Equal(crash.R2))
	})
})
