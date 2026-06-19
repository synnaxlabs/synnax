// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	apiconfig "github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	mockCluster *mock.Cluster
	dist        *distribution.Layer
	channelSvc  *channel.Service
	resolver    apiChannelResolver
)

// apiChannelResolver adapts the API-layer channel service into a codec.DataTypeResolver.
// It injects the root user as the request subject so the API layer's access checks pass;
// in production this subject is supplied by the auth middleware that runs ahead of the
// codec.
type apiChannelResolver struct {
	svc     *apichannel.Service
	subject ontology.ID
}

func (r apiChannelResolver) RetrieveDataTypes(
	ctx context.Context,
	keys channel.Keys,
) (map[channel.Key]telem.DataType, error) {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", r.subject)
	return r.svc.RetrieveDataTypes(fctx, keys)
}

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport gRPC Framer Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	mockCluster = mock.NewCluster(ctx, 1)
	node := mockCluster.Nodes[1]
	dist = node.Layer
	insecure := true
	sec := MustSucceed(security.NewProvider(security.ProviderConfig{
		Insecure: &insecure,
		KeySize:  secmock.SmallKeySize,
	}))
	svc := MustOpen(service.OpenLayer(ctx, service.LayerConfig{
		Distribution:    dist,
		Security:        sec,
		Storage:         node.Storage,
		RootCredentials: auth.Credentials{Username: "synnax", Password: "seldon"},
	}))
	channelSvc = svc.Channel
	apiChSvc := MustSucceed(apichannel.NewService(apiconfig.LayerConfig{
		Service:      svc,
		Distribution: dist,
	}))
	var root user.User
	Expect(svc.User.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entry(&root).
		Exec(ctx, nil)).To(Succeed())
	resolver = apiChannelResolver{svc: apiChSvc, subject: user.OntologyID(root.Key)}
})
