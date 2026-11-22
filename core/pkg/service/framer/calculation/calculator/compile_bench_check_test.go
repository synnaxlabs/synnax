// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculator_test

import (
	"fmt"
	"testing"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

func BenchmarkCheckCompile(b *testing.B) {
	env := newBenchEnv(b)
	defer env.close(b)
	for i, form := range []string{"return in_%d * 2", "return 'in_%d' * 2"} {
		base := []channel.Channel{{
			Name:     fmt.Sprintf("in_%d", i),
			DataType: telem.Int64T,
			Virtual:  true,
		}}
		if err := env.node.DB.WithTx(env.ctx, func(tx gorp.Tx) error {
			return env.channelSvc.NewWriter(tx).CreateMany(env.ctx, &base)
		}); err != nil {
			b.Fatal(err)
		}
		out, err := env.creator.Create(env.ctx, calctestutil.Calculation{
			Name:       fmt.Sprintf("out_%d", i),
			DataType:   telem.Int64T,
			Expression: fmt.Sprintf(form, i),
		})
		if err != nil {
			b.Fatal(err)
		}
		b.Run([]string{"bare", "quoted"}[i], func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if _, err := compiler.Compile(env.ctx, env.node.DB, compiler.Config{
					ChannelService: env.channelSvc,
					Channel:        out,
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
