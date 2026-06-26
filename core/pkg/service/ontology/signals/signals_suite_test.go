// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"iter"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ontology/signals"
	svcsignals "github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSignals(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Ontology Signals Suite")
}

var (
	node mock.Node
	svc  *changeService
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = mock.NewNode(ctx)
	svc = &changeService{Observer: observe.New[iter.Seq[ontology.Change]]()}
	node.Ontology.RegisterService(svc)
	sigs := MustSucceed(svcsignals.New(svcsignals.Config{
		Channel: channel.Wrap(node.Channel),
		Framer:  framer.Wrap(node.Framer),
	}))
	MustOpen(signals.Publish(ctx, sigs, node.Ontology))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
