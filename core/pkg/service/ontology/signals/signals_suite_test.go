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
	dist       mock.Node
	channelSvc *channel.Service
	svc        *changeService
)

var _ = BeforeSuite(func(ctx SpecContext) {
	dist = mock.MustOpenNode(ctx)
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostResolver: dist.Cluster,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		Search:       dist.Search,
	}))
	svc = &changeService{Observer: observe.New[iter.Seq[ontology.Change]]()}
	dist.Ontology.RegisterService(svc)
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Search:   dist.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Label:    labelSvc,
		Search:   dist.Search,
	}))
	framerSvc := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:       dist.Framer,
		Channel:      channelSvc,
		DB:           dist.DB,
		Status:       statusSvc,
		HostResolver: dist.Cluster,
	}))
	sigs := MustSucceed(svcsignals.New(svcsignals.Config{
		Channel: channelSvc,
		Framer:  framerSvc,
	}))
	MustOpen(signals.Publish(ctx, sigs, dist.Ontology))
})
