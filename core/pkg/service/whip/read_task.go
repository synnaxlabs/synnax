// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package whip

import (
	"context"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/pion/rtcp"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/video/ingest"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/video"
)

// maxBatch is the most frames that one read returns.
const maxBatch = 64

// source connects a ReadTask to the encoders that publish to it.
type source struct {
	factory *Factory
	task    uuid.UUID
	channel channel.Key
	// keys are the channels from ingest.Channels.
	keys    channel.Keys
	offset  telem.TimeSpan
	capture bool
}

var _ driver.Source = (*source)(nil)

// Open implements driver.Source. It accepts publishes until the reader closes.
func (s *source) Open(context.Context) (driver.Reader, error) {
	in := &inbox{
		channel:  s.channel,
		keys:     s.keys,
		offset:   s.offset,
		capture:  s.capture,
		limit:    s.factory.cfg.KeyframeLimit,
		arrivals: make(chan arrival, maxBatch),
		failed:   make(chan struct{}),
		closed:   make(chan struct{}),
	}
	s.factory.register(s.task, in)
	return &reader{source: s, inbox: in}, nil
}

// reader reads the frames of one run.
type reader struct {
	*source
	inbox *inbox
}

var _ driver.Reader = (*reader)(nil)

// Read implements driver.Reader.
func (r *reader) Read(ctx context.Context) (driver.Batch, error) {
	var first arrival
	select {
	case <-ctx.Done():
		return driver.Batch{}, ctx.Err()
	case <-r.inbox.failed:
		return driver.Batch{}, r.inbox.fatal
	case first = <-r.inbox.arrivals:
	}
	var (
		frames []telem.VideoFrame
		stamps []telem.TimeStamp
		err    error
	)
	take := func(a arrival) {
		err = errors.Combine(err, a.err)
		if a.usable {
			frames = append(frames, a.frame)
			stamps = append(stamps, a.stamp)
		}
	}
	take(first)
drain:
	for len(frames) < maxBatch {
		select {
		case a := <-r.inbox.arrivals:
			take(a)
		default:
			break drain
		}
	}
	if err != nil {
		err = errors.Mark(err, driver.ErrDegraded)
	}
	if len(frames) == 0 {
		return driver.Batch{}, err
	}
	return driver.Batch{Frame: ingest.Frame(r.keys, frames, stamps)}, err
}

// Close implements driver.Reader.
func (r *reader) Close() error {
	r.factory.unregister(r.task, r.inbox)
	r.inbox.close()
	return nil
}

// arrival is one frame from a session, with its time and the result of the keyframe
// watch.
type arrival struct {
	frame  telem.VideoFrame
	stamp  telem.TimeStamp
	usable bool
	err    error
}

// inbox takes the frames of the sessions of one run. A run has at most one session.
type inbox struct {
	channel  channel.Key
	keys     channel.Keys
	offset   telem.TimeSpan
	capture  bool
	limit    time.Duration
	arrivals chan arrival
	// failed closes after fatal is set.
	failed   chan struct{}
	fatal    error
	failOnce sync.Once
	// closed closes when the run ends.
	closed chan struct{}
	mu     struct {
		sync.Mutex
		session *session
		closed  bool
	}
}

func (in *inbox) publish(
	ctx context.Context,
	api *webrtc.API,
	offer string,
) (string, uuid.UUID, error) {
	if err := checkOffer(offer); err != nil {
		return "", uuid.Nil(), err
	}
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return "", uuid.Nil(), err
	}
	s := &session{key: uuid.New(), pc: pc, inbox: in, done: make(chan struct{})}
	answer, err := s.negotiate(ctx, offer)
	if err != nil {
		s.close()
		return "", uuid.Nil(), err
	}
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.mu.closed {
		s.close()
		return "", uuid.Nil(), ErrNotRunning
	}
	if in.mu.session != nil {
		in.mu.session.close()
	}
	in.mu.session = s
	return answer, s.key, nil
}

func (in *inbox) unpublish(key uuid.UUID) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.mu.session == nil || in.mu.session.key != key {
		return errors.Wrapf(query.ErrNotFound, "session %s does not exist", key)
	}
	in.mu.session.close()
	in.mu.session = nil
	return nil
}

// end closes s when the peer connection fails or closes.
func (in *inbox) end(s *session) {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.mu.session == s {
		in.mu.session = nil
	}
	s.close()
}

func (in *inbox) close() {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.mu.closed {
		return
	}
	in.mu.closed = true
	close(in.closed)
	if in.mu.session != nil {
		in.mu.session.close()
		in.mu.session = nil
	}
}

func (in *inbox) fail(err error) {
	in.failOnce.Do(func() {
		in.fatal = err
		close(in.failed)
	})
}

// deliver sends a to the reader. It returns false when the session or the run ends
// first.
func (in *inbox) deliver(a arrival, done <-chan struct{}) bool {
	select {
	case in.arrivals <- a:
		return true
	case <-done:
		return false
	case <-in.closed:
		return false
	}
}

// checkOffer returns an error that matches validate.ErrValidation for an offer that
// cannot be parsed or that offers no H.264 video.
func checkOffer(offer string) error {
	var desc sdp.SessionDescription
	if err := desc.UnmarshalString(offer); err != nil {
		return errors.Wrapf(validate.ErrValidation, "offer: %s", err)
	}
	for _, media := range desc.MediaDescriptions {
		if media.MediaName.Media != "video" {
			continue
		}
		for _, attr := range media.Attributes {
			if attr.Key == "rtpmap" && strings.Contains(
				strings.ToUpper(attr.Value), " H264/",
			) {
				return nil
			}
		}
	}
	return errors.Wrap(
		validate.ErrValidation,
		"offer: the encoder offers no H.264 video. Set its video encoder to H.264",
	)
}

// session is the WebRTC connection of one publish.
type session struct {
	key       uuid.UUID
	pc        *webrtc.PeerConnection
	inbox     *inbox
	done      chan struct{}
	closeOnce sync.Once
}

// negotiate answers offer once the connection gathers all of its ICE candidates, so
// that the encoder needs no trickle ICE.
func (s *session) negotiate(ctx context.Context, offer string) (string, error) {
	if _, err := s.pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		return "", err
	}
	s.pc.OnTrack(s.read)
	s.pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed {
			s.inbox.end(s)
		}
	})
	if err := s.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offer,
	}); err != nil {
		return "", errors.Wrapf(validate.ErrValidation, "offer: %s", err)
	}
	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	gathered := webrtc.GatheringCompletePromise(s.pc)
	if err = s.pc.SetLocalDescription(answer); err != nil {
		return "", err
	}
	select {
	case <-gathered:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return s.pc.LocalDescription().SDP, nil
}

// read runs on its own goroutine for each track that the encoder sends.
func (s *session) read(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	if !strings.EqualFold(track.Codec().MimeType, webrtc.MimeTypeH264) {
		return
	}
	// A virtual channel has no index, so its frames carry no time.
	var clock *video.Clock
	if len(s.inbox.keys) > 1 {
		clock = &video.Clock{
			Rate:    track.Codec().ClockRate,
			Offset:  s.inbox.offset,
			Capture: s.inbox.capture,
		}
	}
	go readReports(receiver, clock)
	dec := &rtph264.Decoder{PacketizationMode: 1}
	if err := dec.Init(); err != nil {
		s.inbox.fail(err)
		return
	}
	var (
		h264  video.H264
		watch = video.KeyframeWatch{Limit: s.inbox.limit}
	)
	// A keyframe request lets viewers start without waiting for the encoder's
	// keyframe interval.
	lastRequest := s.requestKeyframe(track)
	for {
		pkt, _, err := track.ReadRTP()
		if err != nil {
			return
		}
		au, err := dec.Decode(pkt)
		if err != nil {
			// The decoder waits for more packets, or drops an access unit with lost
			// packets.
			continue
		}
		f, err := h264.Frame(au)
		if err != nil {
			s.inbox.fail(err)
			return
		}
		now := time.Now()
		a := arrival{frame: f}
		a.usable, a.err = watch.Observe(f, now)
		if a.err != nil && now.Sub(lastRequest) > s.inbox.limit {
			lastRequest = s.requestKeyframe(track)
		}
		if clock != nil {
			var ok bool
			if a.stamp, ok = clock.Stamp(telem.Now(), pkt.Timestamp); !ok {
				continue
			}
		}
		if !s.inbox.deliver(a, s.done) {
			return
		}
	}
}

// readReports passes the sender reports of receiver to clock, which may be nil, until
// the connection closes. Pion's interceptors also need RTCP to be read.
func readReports(receiver *webrtc.RTPReceiver, clock *video.Clock) {
	for {
		pkts, _, err := receiver.ReadRTCP()
		if err != nil {
			return
		}
		for _, pkt := range pkts {
			if sr, ok := pkt.(*rtcp.SenderReport); ok && clock != nil {
				clock.Report(sr.NTPTime, sr.RTPTime)
			}
		}
	}
}

// requestKeyframe asks the encoder for a keyframe and returns the time it asked.
func (s *session) requestKeyframe(track *webrtc.TrackRemote) time.Time {
	_ = s.pc.WriteRTCP([]rtcp.Packet{
		&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())},
	})
	return time.Now()
}

func (s *session) close() {
	s.closeOnce.Do(func() {
		close(s.done)
		_ = s.pc.Close()
	})
}
