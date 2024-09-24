// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	. "github.com/synnaxlabs/x/testutil"
	"time"

	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	securitymock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/user"
)

type Builder struct {
	mock.Builder
}

func (b *Builder) New(ctx context.Context) api.Provider {
	return api.NewProvider(b.NewConfig(ctx))
}

func (b *Builder) NewConfig(ctx context.Context) api.Config {
	dist := b.Builder.New(ctx)
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	return api.Config{
		Channel:       dist.Channel,
		Ontology:      dist.Ontology,
		Storage:       dist.Storage,
		User:          MustSucceed(user.NewService(ctx, user.Config{DB: dist.Storage.Gorpify(), Ontology: dist.Ontology, Group: dist.Group})),
		Token:         &token.Service{KeyProvider: securitymock.KeyProvider{Key: key}, Expiration: 10000 * time.Hour},
		Authenticator: &auth.KV{DB: dist.Storage.Gorpify()},
		RBAC:          MustSucceed(rbac.NewService(rbac.Config{DB: dist.Storage.Gorpify()})),
		Enforcer:      &access.AllowAll{},
		Cluster:       dist.Cluster,
	}

}

func Open(cfg ...distribution.Config) *Builder {
	builder := &Builder{}
	builder.Builder = *mock.NewBuilder(cfg...)
	return builder
}
