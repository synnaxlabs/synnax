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
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSignals(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Signals Suite", SuiteConfig())
}

type testUUIDEntry struct {
	Name string    `json:"name"`
	Key  uuid.UUID `json:"key"`
}

func (t testUUIDEntry) GorpKey() uuid.UUID { return t.Key }

func (t testUUIDEntry) EncodeOrc(w *orc.Writer) error {
	w.String(t.Name)
	w.Write(t.Key[:])
	return nil
}

func (t *testUUIDEntry) DecodeOrc(r *orc.Reader) (err error) {
	if t.Name, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(t.Key[:])
	return err
}

type testNumericEntry struct {
	Value string `json:"value"`
	Key   uint32 `json:"key"`
}

func (t testNumericEntry) GorpKey() uint32 { return t.Key }

func (t testNumericEntry) EncodeOrc(w *orc.Writer) error {
	w.String(t.Value)
	w.Uint32(t.Key)
	return nil
}

func (t *testNumericEntry) DecodeOrc(r *orc.Reader) (err error) {
	if t.Value, err = r.String(); err != nil {
		return err
	}
	t.Key, err = r.Uint32()
	return err
}

type testStringEntry struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

func (t testStringEntry) GorpKey() string { return t.Key }

func (t testStringEntry) EncodeOrc(w *orc.Writer) error {
	w.String(t.Key)
	w.Int64(int64(t.Value))
	return nil
}

func (t *testStringEntry) DecodeOrc(r *orc.Reader) error {
	var err error
	if t.Key, err = r.String(); err != nil {
		return err
	}
	v, err := r.Int64()
	t.Value = int(v)
	return err
}

const uuidTableName = "signals_test_uuid"

// testResource returns the resource of the test entry with the given key.
func testResource[K gorp.Key](k K) resource.ID {
	return resource.ID{Type: "signals_test", Key: gorp.FormatKey(k)}
}

var (
	db          *gorp.DB
	channelSvc  *channel.Service
	framerSvc   *framer.Service
	uuidTable   *gorp.Table[uuid.UUID, testUUIDEntry]
	numTable    *gorp.Table[uint32, testNumericEntry]
	stringTable *gorp.Table[string, testStringEntry]
	nameCounter atomic.Uint64
)

// nextName returns a distinct channel name, since the mock cluster keeps the channels
// of every earlier spec.
func nextName() string {
	return fmt.Sprintf("signals_test_%d", nameCounter.Add(1))
}

// openStreamer opens a streamer on keys and returns once the streamer is subscribed.
// The streamer closes when the spec ends.
func openStreamer(
	ctx context.Context,
	keys ...channel.Key,
) confluence.Outlet[framer.StreamerResponse] {
	GinkgoHelper()
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        keys,
		SendOpenAck: true,
	}))
	requests, responses := confluence.Attach(streamer, 2)
	sCtx, cancel := signal.Isolated()
	closeStreamer := signal.NewHardShutdown(sCtx, cancel)
	DeferCleanup(func() {
		requests.Close()
		confluence.Drain(responses)
		Expect(closeStreamer.Close()).To(Succeed())
	})
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	Eventually(responses.Outlet()).Should(Receive())
	return responses
}

// retrieveChannel returns the channel with the given name.
func retrieveChannel(ctx context.Context, name string) channel.Channel {
	GinkgoHelper()
	var ch channel.Channel
	Expect(channelSvc.NewRetrieve().
		Where(channel.MatchNames(name)).
		Entry(&ch).
		Exec(ctx, db),
	).To(Succeed())
	return ch
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: node.DB,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           node.DB,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	uuidTable = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, testUUIDEntry]{
		DB:   db,
		Name: uuidTableName,
	}))
	numTable = MustOpen(
		gorp.OpenTable(ctx, gorp.TableConfig[uint32, testNumericEntry]{DB: db}),
	)
	stringTable = MustOpen(
		gorp.OpenTable(ctx, gorp.TableConfig[string, testStringEntry]{DB: db}),
	)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
