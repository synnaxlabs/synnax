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
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// BenchmarkDelete measures the delete route for one channel and for a page of 100, each
// channel held by three schematics, and the same with channels that nothing holds
// ("unheld"). Each run deletes in a transaction that it then discards, so the storage
// delete at commit never runs.
func BenchmarkDelete(b *testing.B) {
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	node := disttestutil.OpenNode(ctx)
	layer, err := svctestutil.OpenLayer(ctx, node)
	if err != nil {
		b.Fatalf("failed to open the layer: %v", err)
	}
	defer func() {
		if err := errors.Join(layer.Close(), node.Close()); err != nil {
			b.Errorf("failed to close: %v", err)
		}
	}()
	svc, err := apichannel.NewService(apichannel.ServiceConfig{
		DB:      node.DB,
		RBAC:    layer.RBAC,
		Channel: layer.Channel,
		Status:  layer.Status,
		Impact:  layer.Impact,
	})
	if err != nil {
		b.Fatalf("failed to open the API service: %v", err)
	}
	keys := createUsedChannels(b, node.DB, layer)
	unheld := createUnheldChannels(b, node.DB, layer)
	run := func(b *testing.B, req apichannel.DeleteRequest) {
		b.Helper()
		b.ReportAllocs()
		for range b.N {
			tx := node.DB.OpenTx()
			_, err := svc.Delete(ctx, tx, req)
			if err = errors.Join(err, tx.Close()); err != nil {
				b.Fatalf("failed to delete: %v", err)
			}
		}
	}
	b.Run("one", func(b *testing.B) {
		run(b, apichannel.DeleteRequest{Keys: keys[:1]})
	})
	b.Run("page", func(b *testing.B) {
		run(b, apichannel.DeleteRequest{Keys: keys})
	})
	b.Run("unheld/one", func(b *testing.B) {
		run(b, apichannel.DeleteRequest{Keys: unheld[:1]})
	})
	b.Run("unheld/page", func(b *testing.B) {
		run(b, apichannel.DeleteRequest{Keys: unheld})
	})
}

// createUnheldChannels creates 100 channels that nothing holds.
func createUnheldChannels(
	b *testing.B,
	db *gorp.DB,
	layer *service.Layer,
) channel.Keys {
	b.Helper()
	channels := make([]channel.Channel, 100)
	for i := range channels {
		channels[i] = channel.Channel{
			Name:     fmt.Sprintf("bench_unheld_%d", i),
			DataType: telem.Float32T,
			Virtual:  true,
		}
	}
	if err := db.WithTx(b.Context(), func(tx gorp.Tx) error {
		return layer.Channel.NewWriter(tx).CreateMany(b.Context(), &channels)
	}); err != nil {
		b.Fatalf("failed to create the channels: %v", err)
	}
	return channel.KeysFromChannels(channels)
}
