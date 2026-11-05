// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rtsp_test

import (
	"context"
	"encoding/json/v2"
	"net"
	"time"
	"uuid"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rtsp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	sps = []byte{
		0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02, 0x27, 0xe5, 0x84, 0x00, 0x00,
		0x03, 0x00, 0x04, 0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
	}
	pps = []byte{0x68, 0xce, 0x3c, 0x80}
	idr = []byte{0x65, 0x88, 0x84, 0x00}
	// The slice headers below hold first_mb_in_slice = 0, then slice_type.
	pSlice = []byte{0x41, 0xc0, 0x00}
	bSlice = []byte{0x41, 0xa0, 0x00}
)

func annexB(au ...[]byte) []byte { return MustSucceed(h264.AnnexB(au).Marshal()) }

func keyframe(au ...[]byte) telem.VideoFrame {
	return telem.VideoFrame{
		Keyframe: true,
		Codec:    telem.VideoCodecH264,
		Data:     annexB(au...),
	}
}

func deltaFrame(au ...[]byte) telem.VideoFrame {
	return telem.VideoFrame{Codec: telem.VideoCodecH264, Data: annexB(au...)}
}

// camera is an RTSP server in the test process that serves one stream.
type camera struct {
	addr      string
	server    *gortsplib.Server
	stream    *gortsplib.ServerStream
	media     *description.Media
	encoder   *rtph264.Encoder
	timestamp uint32
}

var (
	_ gortsplib.ServerHandlerOnDescribe = (*camera)(nil)
	_ gortsplib.ServerHandlerOnSetup    = (*camera)(nil)
	_ gortsplib.ServerHandlerOnPlay     = (*camera)(nil)
)

func (c *camera) OnDescribe(
	*gortsplib.ServerHandlerOnDescribeCtx,
) (*base.Response, *gortsplib.ServerStream, error) {
	return &base.Response{StatusCode: base.StatusOK}, c.stream, nil
}

func (c *camera) OnSetup(
	*gortsplib.ServerHandlerOnSetupCtx,
) (*base.Response, *gortsplib.ServerStream, error) {
	return &base.Response{StatusCode: base.StatusOK}, c.stream, nil
}

func (c *camera) OnPlay(*gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	return &base.Response{StatusCode: base.StatusOK}, nil
}

// freeAddr returns a local TCP address that no listener holds.
func freeAddr() string {
	GinkgoHelper()
	l := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
	defer func() { Expect(l.Close()).To(Succeed()) }()
	return l.Addr().String()
}

// openCamera starts a camera at addr that serves forma over TCP. The camera closes at
// the end of the spec.
func openCamera(addr string, forma format.Format) *camera {
	GinkgoHelper()
	c := &camera{addr: addr}
	c.server = &gortsplib.Server{Handler: c, RTSPAddress: addr}
	Expect(c.server.Start()).To(Succeed())
	c.media = &description.Media{
		Type:    description.MediaTypeVideo,
		Formats: []format.Format{forma},
	}
	c.stream = &gortsplib.ServerStream{
		Server: c.server,
		Desc:   &description.Session{Medias: []*description.Media{c.media}},
	}
	Expect(c.stream.Initialize()).To(Succeed())
	if h, ok := forma.(*format.H264); ok {
		c.encoder = MustSucceed(h.CreateEncoder())
	}
	DeferCleanup(c.close)
	return c
}

func openH264Camera(addr string, parameterSets ...[]byte) *camera {
	GinkgoHelper()
	forma := &format.H264{PayloadTyp: 96, PacketizationMode: 1}
	if len(parameterSets) == 2 {
		forma.SPS, forma.PPS = parameterSets[0], parameterSets[1]
	}
	return openCamera(addr, forma)
}

func (c *camera) url() string { return "rtsp://" + c.addr + "/stream" }

// write sends one access unit to every client that plays the stream.
func (c *camera) write(au ...[]byte) {
	GinkgoHelper()
	for _, pkt := range MustSucceed(c.encoder.Encode(au)) {
		pkt.Timestamp = c.timestamp
		Expect(c.stream.WritePacketRTP(c.media, pkt)).To(Succeed())
	}
	c.timestamp += 3000
}

// close stops the camera, which ends the session of every client.
func (c *camera) close() {
	if c.server == nil {
		return
	}
	c.stream.Close()
	c.server.Close()
	c.server = nil
}

// create creates ch and deletes it at the end of the spec. The GOP cache streams every
// video channel, so a channel that outlives its spec holds a relay tap open.
func create(ctx context.Context, ch *channel.Channel) {
	GinkgoHelper()
	ch.Name = UniqueChannelName()
	w := channelSvc.NewWriter(nil)
	Expect(w.Create(ctx, ch)).To(Succeed())
	key := ch.Key()
	DeferCleanup(func(ctx SpecContext) {
		Expect(errors.Skip(w.Delete(ctx, key, false), query.ErrNotFound)).
			To(Succeed())
	})
}

func createVirtual(ctx context.Context, dt telem.DataType) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{DataType: dt, Virtual: true}
	create(ctx, &ch)
	return ch
}

// createPersisted creates a video channel and its index, and returns both.
func createPersisted(ctx context.Context) (channel.Channel, channel.Channel) {
	GinkgoHelper()
	idx := channel.Channel{DataType: telem.TimestampT, IsIndex: true}
	create(ctx, &idx)
	ch := channel.Channel{DataType: telem.VideoT, LocalIndex: idx.LocalKey}
	create(ctx, &ch)
	return idx, ch
}

// readPersisted returns the times and frames stored in a video channel and its index.
func readPersisted(
	ctx context.Context,
	idx, ch channel.Channel,
) ([]telem.TimeStamp, []telem.VideoFrame) {
	GinkgoHelper()
	iter := MustSucceed(framerSvc.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   channel.Keys{idx.Key(), ch.Key()},
		Bounds: telem.TimeRangeMax,
	}))
	defer func() { Expect(iter.Close()).To(Succeed()) }()
	Expect(iter.SeekFirst()).To(BeTrue())
	Expect(iter.Next(framer.IteratorAutoSpan)).To(BeTrue())
	var (
		stamps []telem.TimeStamp
		frames []telem.VideoFrame
	)
	for _, s := range iter.Value().Get(idx.Key()).Series {
		stamps = append(stamps, s.Unmarshal[telem.TimeStamp]()...)
	}
	for _, s := range iter.Value().Get(ch.Key()).Series {
		frames = append(frames, MustSucceed(s.DecodeVideo())...)
	}
	return stamps, frames
}

func newTask(cfg rtsp.ReadConfig) task.Task {
	GinkgoHelper()
	var encoded msgpack.EncodedJSON
	Expect(json.Unmarshal(MustSucceed(json.Marshal(cfg)), &encoded)).To(Succeed())
	return task.Task{
		Key:    uuid.New(),
		Name:   "rtsp-task",
		Type:   rtsp.ReadTaskType,
		Config: encoded,
	}
}

func readConfig(url string, ch channel.Channel) rtsp.ReadConfig {
	return rtsp.ReadConfig{
		AutoStart: true,
		URL:       url,
		Channel:   ch.Key(),
		Protocol:  rtsp.ProtocolTCP,
	}
}

// start configures and starts a task, and stops it at the end of the spec.
func start(ctx context.Context, f *rtsp.Factory, t task.Task) {
	GinkgoHelper()
	configured := MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
	DeferCleanup(func() { Expect(configured.Stop(false)).To(Succeed()) })
}

// taskStatus returns the status of t, or the zero status when it has none.
func taskStatus(ctx context.Context, t task.Task) task.Status {
	GinkgoHelper()
	var stat task.Status
	Expect(errors.Skip(
		statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).
			Exec(ctx, nil),
		query.ErrNotFound,
	)).To(Succeed())
	return stat
}

// expectStatus waits for the status of t to have variant, message, and description.
func expectStatus(
	ctx context.Context,
	t task.Task,
	variant status.Variant,
	message, description string,
) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		stat := taskStatus(ctx, t)
		g.Expect(stat.Variant).To(Equal(variant))
		g.Expect(stat.Message).To(Equal(message))
		g.Expect(stat.Description).To(Equal(description))
	}).WithTimeout(10 * time.Second).Should(Succeed())
}

// viewer receives the frames written to one video channel.
type viewer struct {
	key       channel.Key
	responses <-chan framer.StreamerResponse
	pending   []telem.VideoFrame
}

// watch opens a viewer of ch that closes at the end of the spec.
func watch(ctx context.Context, ch channel.Channel) *viewer {
	GinkgoHelper()
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys: channel.Keys{ch.Key()},
	}))
	var (
		requests  = confluence.NewStream[framer.StreamerRequest]()
		responses = confluence.NewStream[framer.StreamerResponse](1024)
	)
	streamer.InFrom(requests)
	streamer.OutTo(responses)
	sCtx, cancel := signal.Isolated()
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	DeferCleanup(func() {
		requests.Close()
		confluence.Drain(responses)
		cancel()
		Expect(sCtx.Wait()).To(Succeed())
	})
	return &viewer{key: ch.Key(), responses: responses.Outlet()}
}

// take returns the next frame that the viewer received, and false if it has none.
func (v *viewer) take() (telem.VideoFrame, bool) {
	GinkgoHelper()
	for len(v.pending) == 0 {
		select {
		case res := <-v.responses:
			for _, s := range res.Frame.Get(v.key).Series {
				v.pending = append(v.pending, MustSucceed(s.DecodeVideo())...)
			}
		default:
			return telem.VideoFrame{}, false
		}
	}
	f := v.pending[0]
	v.pending = v.pending[1:]
	return f, true
}

// clear drops the frames that the viewer received and has not taken.
func (v *viewer) clear() {
	for {
		if _, ok := v.take(); !ok {
			return
		}
	}
}

// connect writes au until the viewer receives a frame, and returns that frame.
func (v *viewer) connect(cam *camera, au ...[]byte) telem.VideoFrame {
	GinkgoHelper()
	var f telem.VideoFrame
	Eventually(func(g Gomega) {
		cam.write(au...)
		var ok bool
		f, ok = v.take()
		g.Expect(ok).To(BeTrue())
	}).WithTimeout(10 * time.Second).Should(Succeed())
	return f
}

// expect skips frames until the viewer receives f.
func (v *viewer) expect(f telem.VideoFrame) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		next, ok := v.take()
		g.Expect(ok).To(BeTrue())
		g.Expect(next).To(Equal(f))
	}).Should(Succeed())
}
