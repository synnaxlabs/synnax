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
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	securitymock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/user"
	"go.uber.org/zap"
)

type Builder struct {
	mock.Builder
}

func (b *Builder) New() api.Provider { return api.NewProvider(b.NewConfig()) }

func (b *Builder) NewConfig() api.Config {
	dist := b.Builder.New()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	return api.Config{
		Logger:        zap.NewNop(),
		Channel:       dist.Channel,
		Framer:        dist.Framer,
		Ontology:      dist.Ontology,
		Storage:       dist.Storage,
		User:          &user.Service{DB: dist.Storage.Gorpify(), Ontology: dist.Ontology},
		Token:         &token.Service{KeyProvider: securitymock.KeyProvider{Key: key}, Expiration: 10000 * time.Hour},
		Authenticator: &auth.KV{DB: dist.Storage.Gorpify()},
		Enforcer:      access.AllowAll{},
		Cluster:       dist.Cluster,
	}

}

func New(cfg ...distribution.Config) *Builder {
	builder := &Builder{}
	builder.Builder = *mock.NewBuilder(cfg...)
	return builder
}
