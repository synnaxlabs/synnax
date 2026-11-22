// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkRetrieveByPath measures a retrieve of one channel by its path at depths 0
// and 3, beside a table of 2,001 channels.
func BenchmarkRetrieveByPath(b *testing.B) {
	for _, depth := range []int{0, 3} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			env := newBenchEnv(b, 1_000)
			b.Cleanup(func() { env.close(b) })
			segments := make([]string, 0, depth+1)
			if err := env.db.WithTx(b.Context(), func(tx gorp.Tx) error {
				var parent *group.Key
				for i := range depth {
					g := group.Group{
						Name:  fmt.Sprintf("folder_%d", i),
						Scope: resource.TypeChannel,
					}
					if parent != nil {
						g.Parent = new(group.ResourceID(*parent))
					}
					if err := env.groups.NewWriter(tx).
						Create(b.Context(), &g); err != nil {
						return err
					}
					segments = append(segments, g.Name)
					parent = &g.Key
				}
				ch := channel.Channel{
					Name:     "by_path",
					DataType: telem.Float32T,
					Virtual:  true,
					Group:    parent,
				}
				segments = append(segments, ch.Name)
				return env.svc.NewWriter(tx).Create(b.Context(), &ch)
			}); err != nil {
				b.Fatal(err)
			}
			path := strings.Join(segments, ".")
			var ch channel.Channel
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := env.svc.NewRetrieve().
					Where(channel.MatchPaths(path)).
					Entry(&ch).
					Exec(b.Context(), env.db); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
