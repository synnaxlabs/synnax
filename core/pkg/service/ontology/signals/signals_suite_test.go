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
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology/signals"
	svcsignals "github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSignals(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Ontology Signals Suite")
}

var (
	node       mock.Node
	svc        *changeService
	framerSvc  *framer.Service
	channelSvc *channel.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	node = mock.NewNode(ctx)
	svc = &changeService{Observer: observe.New[iter.Seq[ontology.Change]]()}
	node.Ontology.RegisterService(svc)
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: node.Ontology,
		Group:    node.Group,
		Search:   node.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Ontology: node.Ontology,
		Group:    node.Group,
		Label:    labelSvc,
		Search:   node.Search,
	}))
	channelSvc = MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
		DB:           node.DB,
		Distribution: node.Channel,
		Status:       statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:  node.Framer,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	sigs := MustSucceed(svcsignals.New(svcsignals.Config{
		Channel: channelSvc,
		Framer:  framerSvc,
	}))
	MustOpen(signals.Publish(ctx, sigs, node.Ontology))
})
