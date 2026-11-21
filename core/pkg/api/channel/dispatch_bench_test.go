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
	"runtime"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
)

// benchWriter holds the old API create and dispatch, bound as the Core binds them, and
// a subject that can use them.
type benchWriter struct {
	create   freighter.UnaryHandler[apichannel.CreateRequest, apichannel.CreateResponse]
	dispatch freighter.UnaryHandler[apichannel.DispatchRequest, apichannel.DispatchResponse]
	subject  freighter.Context
}

func openBenchWriter(b *testing.B, db *gorp.DB, layer *service.Layer) benchWriter {
	b.Helper()
	svc, err := apichannel.NewService(apichannel.ServiceConfig{
		DB:      db,
		RBAC:    layer.RBAC,
		Channel: layer.Channel,
		Status:  layer.Status,
	})
	if err != nil {
		b.Fatalf("failed to open the API service: %v", err)
	}
	binder, err := bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: layer.RBAC,
		Held: apiauth.HeldTables(),
	})
	if err != nil {
		b.Fatalf("failed to open the binder: %v", err)
	}
	u, err := rbactestutil.CreateUser(
		b.Context(), db, layer.RBAC, layer.User, user.User{Name: "bench_writer"},
	)
	if err != nil {
		b.Fatalf("failed to create the user: %v", err)
	}
	if _, err = rbactestutil.Grant(
		b.Context(),
		db,
		layer.RBAC,
		layer.User,
		u.Key,
		access.AllActions,
		resource.ID{Type: resource.TypeChannel},
		resource.ID{Type: resource.TypeFramer},
		resource.ID{Type: resource.TypeGroup},
	); err != nil {
		b.Fatalf("failed to grant: %v", err)
	}
	return benchWriter{
		create:   bind.Write(binder, svc.Create),
		dispatch: bind.Write(binder, svc.Dispatch),
		subject:  AuthedCtx(b.Context(), u),
	}
}

// BenchmarkDispatchCalculated measures an old API edit of the expression or the
// operations of a calculated channel. Each iteration alternates between two values.
func BenchmarkDispatchCalculated(b *testing.B) {
	gomega.RegisterTestingT(b)
	node := disttestutil.OpenNode(b.Context())
	layer, err := svctestutil.OpenLayer(b.Context(), node)
	if err != nil {
		b.Fatalf("failed to open the layer: %v", err)
	}
	defer func() {
		if err := layer.Close(); err != nil {
			b.Errorf("failed to close the layer: %v", err)
		}
		if err := node.Close(); err != nil {
			b.Errorf("failed to close the node: %v", err)
		}
	}()
	w := openBenchWriter(b, node.DB, layer)
	created, err := w.create(w.subject, apichannel.CreateRequest{
		Channels: []apichannel.Channel{
			{Name: "bench_input", DataType: telem.Float32T, Virtual: true},
		},
	})
	if err != nil {
		b.Fatalf("failed to create the input: %v", err)
	}
	avg := []channel.Operation{{Type: channel.OperationTypeAvg, Duration: telem.Second}}
	edits := map[string][2]channel.Action{
		"expression": {
			benchSet("expression", "return bench_input * 3"),
			benchSet("expression", "return bench_input * 2"),
		},
		"operations": {
			benchSet("operations", avg),
			benchSet("operations", []channel.Operation{}),
		},
	}
	for i, name := range []string{"expression", "operations"} {
		calc, err := w.create(w.subject, apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:       fmt.Sprintf("bench_calculated_%d", i),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return bench_input * 2",
			}},
		})
		if err != nil {
			b.Fatalf("failed to create the calculated channel: %v", err)
		}
		if len(created.Channels) == 0 || len(calc.Channels) == 0 {
			b.Fatalf("expected the created channels")
		}
		key := calc.Channels[0].Key
		b.Run(name, func(b *testing.B) {
			runtime.GC()
			b.ReportAllocs()
			b.ResetTimer()
			for j := range b.N {
				if _, err := w.dispatch(w.subject, apichannel.DispatchRequest{
					Key:         key,
					DispatchKey: "bench",
					Actions:     []channel.Action{edits[name][j%2]},
				}); err != nil {
					b.Fatalf("failed to dispatch: %v", err)
				}
			}
		})
	}
}

// benchSet returns an action that sets path to value.
func benchSet(path string, value any) channel.Action {
	return channel.NewSetAction(channel.SetPayload{Path: []string{path}, Value: value})
}
