// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift_test

import (
	"context"
	"encoding/json/v2"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db        *gorp.DB
	statusSvc *status.Service
	rackSvc   *rack.Service
	deviceSvc *device.Service
	secretSvc *secret.Service
)

func TestSift(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Sift Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		Ontology: otg,
		DB:       db,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           db,
		Ontology:     otg,
		Group:        groupSvc,
		HostProvider: testutil.NewStaticHostProvider(1),
		Status:       statusSvc,
		DriverRole:   MustSucceed(racktestutil.CreateDriverRole(ctx, otg)),
	}))
	deviceSvc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Status:   statusSvc,
		Rack:     rackSvc,
	}))
	grantSvc := MustOpen(grant.OpenService(ctx, grant.ServiceConfig{DB: db}))
	secretSvc = MustOpen(secret.OpenService(ctx, secret.ServiceConfig{
		DB:            db,
		Ontology:      otg,
		Group:         groupSvc,
		Uses:          sift.SecretUses(deviceSvc),
		Grant:         grantSvc,
		EncryptionKey: secret.NewEncryptionKey(),
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()

// toJSON converts v into the map a stored device or command holds.
func toJSON(v any) msgpack.EncodedJSON {
	GinkgoHelper()
	b := MustSucceed(json.Marshal(v))
	var m msgpack.EncodedJSON
	Expect(json.Unmarshal(b, &m)).To(Succeed())
	return m
}

// createAPIKey creates a stored secret that holds value.
func createAPIKey(ctx context.Context, value string) secret.Key {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{
		Base: secret.Base{Name: "sift-" + uuid.New().String()[:8]},
	}}
	Expect(secretSvc.NewWriter(nil).Create(ctx, &s, []byte(value))).To(Succeed())
	return s.Base().Key
}

// createDevice creates a Sift device with props on the embedded rack.
func createDevice(ctx context.Context, props sift.Properties) device.Device {
	GinkgoHelper()
	dev := device.Device{
		Key:        uuid.New().String(),
		Rack:       rackSvc.EmbeddedKey,
		Location:   "sift",
		Make:       sift.Make,
		Model:      "sift",
		Name:       "sift-" + uuid.New().String()[:8],
		Configured: true,
		Properties: toJSON(props),
	}
	Expect(deviceSvc.NewWriter(nil).Create(ctx, &dev)).To(Succeed())
	return dev
}

// taskStatus retrieves the status of t.
func taskStatus(ctx context.Context, t task.Task) task.Status {
	GinkgoHelper()
	var stat task.Status
	Expect(statusSvc.NewRetrieve[task.StatusDetails]().
		Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
		Entry(&stat).
		Exec(ctx, nil)).To(Succeed())
	return stat
}

// deviceStatus retrieves the status of dev.
func deviceStatus(ctx context.Context, dev device.Device) device.Status {
	GinkgoHelper()
	var stat device.Status
	Expect(statusSvc.NewRetrieve[device.StatusDetails]().
		Where(status.MatchKeys[device.StatusDetails](dev.OntologyID().String())).
		Entry(&stat).
		Exec(ctx, nil)).To(Succeed())
	return stat
}
