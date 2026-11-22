// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var db *gorp.DB

// secretsKey encrypts the secrets of every spec, which share db.
var secretsKey = secret.NewEncryptionKey()

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(db.Run)
})

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service OPC UA Server Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// env holds a server service and the services that its writes reference.
type env struct {
	status *status.Service
	racks  *rack.Service
	secret *secret.Service
	certs  *certificate.Service
	svc    *server.Service
}

// openEnv opens a server service on db with the services it depends on.
func openEnv(ctx SpecContext, db *gorp.DB, sigs *signals.Provider) env {
	GinkgoHelper()
	var e env
	MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
	e.status = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
	roles := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
	e.racks = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		HostKey:             1,
		Status:              e.status,
		DriverRole:          MustSucceed(racktestutil.CreateDriverRole(ctx, db, roles)),
		HealthCheckInterval: telem.Hour,
	}))
	e.secret = secrettestutil.OpenService(
		ctx,
		db,
		secret.ServiceConfig{EncryptionKey: secretsKey},
	)
	e.certs = MustOpen(certificate.OpenService(ctx, certificate.ServiceConfig{
		DB:      db,
		Secret:  e.secret,
		Status:  e.status,
		Rack:    e.racks.EmbeddedKey,
		HostKey: 1,
	}))
	e.svc = MustOpen(server.OpenService(ctx, server.ServiceConfig{
		DB:          db,
		Status:      e.status,
		Secret:      e.secret,
		Certificate: e.certs,
		Signals:     sigs,
	}))
	return e
}

// newConnection returns a connection to endpoint with the schema defaults.
func newConnection(endpoint string) server.Connection {
	var c server.Connection
	c.SetDefaults()
	c.Endpoint = endpoint
	return c
}

// signed returns a signed connection under the Basic256Sha256 policy that trusts the
// server as trust says.
func signed(trust server.ServerTrust) server.Security {
	var c server.SecuredChannel
	c.SetDefaults()
	c.Policy = server.SecurityPolicyBasic256Sha256
	c.ServerTrust = trust
	return server.Security{Variant: server.SignSecurity{SecuredChannel: c}}
}

// pinned returns the trust in the one server certificate stored under key.
func pinned(key certificate.Key) server.ServerTrust {
	return server.ServerTrust{Variant: server.PinnedServerTrust{Certificate: key}}
}
