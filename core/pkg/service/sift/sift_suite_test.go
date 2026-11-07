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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db         *gorp.DB
	otg        *ontology.Ontology
	statusSvc  *status.Service
	rackSvc    *rack.Service
	deviceSvc  *device.Service
	secretSvc  *secret.Service
	channelSvc *channel.Service
	framerSvc  *framer.Service
	rangerSvc  *ranger.Service
	labelSvc   *label.Service
)

func TestSift(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Sift Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
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
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      db,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           db,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	rangerSvc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
})

// factoryConfig returns the configuration of a factory on the services of the suite.
func factoryConfig() sift.FactoryConfig {
	return sift.FactoryConfig{
		DB:                 db,
		Device:             deviceSvc,
		Status:             statusSvc,
		Secret:             secretSvc,
		Ranger:             rangerSvc,
		Channel:            channelSvc,
		Framer:             framerSvc,
		Label:              labelSvc,
		Timeout:            2 * telem.Second,
		CheckpointInterval: 20 * telem.Millisecond,
	}
}

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

// createIndex creates an index channel with a unique name.
func createIndex(ctx context.Context) channel.Channel {
	GinkgoHelper()
	idx := channel.Channel{
		Name:     "time_" + uuid.New().String()[:8],
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &idx)).To(Succeed())
	return idx
}

// createData creates a float64 channel on idx with a unique name.
func createData(ctx context.Context, idx channel.Channel) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:       "data_" + uuid.New().String()[:8],
		DataType:   telem.Float64T,
		LocalIndex: idx.LocalKey,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

// write writes the samples of ch at stamps to storage.
func write(
	ctx context.Context,
	idx, ch channel.Channel,
	stamps []telem.TimeStamp,
	values telem.Series,
) {
	GinkgoHelper()
	keys := channel.Keys{idx.Key(), ch.Key()}
	w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:  keys,
		Start: stamps[0],
	}))
	Expect(MustSucceed(w.Write(frame.NewMulti(
		keys,
		[]telem.Series{telem.NewSeries(stamps), values},
	)))).To(BeTrue())
	Expect(w.Close()).To(Succeed())
}

// createRange creates a range over tr.
func createRange(ctx context.Context, tr telem.TimeRange) ranger.Range {
	GinkgoHelper()
	r := ranger.Range{Name: "test-" + uuid.New().String()[:8], TimeRange: tr}
	Expect(rangerSvc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
	return r
}

// execCommand runs a command on the task dt of t and returns the status that answers
// it.
func execCommand(
	ctx context.Context,
	dt driver.Task,
	t task.Task,
	typ string,
	args any,
) task.Status {
	GinkgoHelper()
	key := uuid.New().String()
	Expect(dt.Exec(ctx, task.Command{
		Task: t.Key,
		Type: typ,
		Key:  key,
		Args: toJSON(args),
	})).To(Succeed())
	var stat task.Status
	Eventually(func(g Gomega) {
		stat = taskStatus(ctx, t)
		g.Expect(stat.Details.Cmd).To(Equal(key))
	}).WithTimeout(10 * time.Second).Should(Succeed())
	return stat
}

// uploadStatus retrieves the status of the upload with key, or returns
// query.ErrNotFound.
func uploadStatus(ctx context.Context, key status.Key) (sift.UploadStatus, error) {
	var s sift.UploadStatus
	err := statusSvc.NewRetrieve[sift.UploadDetails]().
		Where(status.MatchKeys[sift.UploadDetails](key)).
		Entry(&s).
		Exec(ctx, nil)
	return s, err
}
