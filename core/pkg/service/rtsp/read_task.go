// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rtsp

import (
	"context"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/video/ingest"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/video"
	"go.uber.org/zap"
)

// ErrNoH264 is returned for a stream with no H.264 track.
var ErrNoH264 = errors.New("the stream has no H.264 video track")

// maxBatch is the most frames that one read returns.
const maxBatch = 64

// source connects a ReadTask to one RTSP stream.
type source struct {
	url      *base.URL
	protocol *gortsplib.Protocol
	// keys are the channels from ingest.Channels.
	keys    channel.Keys
	offset  telem.TimeSpan
	capture bool
	limit   time.Duration
	ins     alamos.Instrumentation
}

var _ driver.Source = (*source)(nil)

// Open implements driver.Source. The reader connects on its first read, so that an
// offline camera makes the task warn and retry instead of failing to start.
func (s *source) Open(context.Context) (driver.Reader, error) {
	return &reader{source: s}, nil
}

// reader reads the frames of one run, and connects again after it loses the stream.
type reader struct {
	*source
	sess *session
}

var _ driver.Reader = (*reader)(nil)

// Read implements driver.Reader.
func (r *reader) Read(ctx context.Context) (driver.Batch, error) {
	if r.sess == nil {
		sess, err := r.connect(ctx)
		if err != nil {
			return driver.Batch{}, err
		}
		r.sess = sess
	}
	var first timed
	select {
	case <-ctx.Done():
		return driver.Batch{}, ctx.Err()
	case <-r.sess.failed:
		return driver.Batch{}, r.sess.fatal
	case <-r.sess.done:
		err := r.sess.err
		r.sess.close()
		r.sess = nil
		return driver.Batch{}, errors.Mark(
			errors.Wrap(err, "lost the stream"), driver.ErrTemporary,
		)
	case first = <-r.sess.frames:
	}
	var (
		frames = []telem.VideoFrame{first.frame}
		stamps = []telem.TimeStamp{first.stamp}
	)
drain:
	for len(frames) < maxBatch {
		select {
		case t := <-r.sess.frames:
			frames = append(frames, t.frame)
			stamps = append(stamps, t.stamp)
		default:
			break drain
		}
	}
	var (
		err  error
		kept int
		now  = time.Now()
	)
	for i, f := range frames {
		usable, wErr := r.sess.watch.Observe(f, now)
		if wErr != nil {
			err = errors.Mark(wErr, driver.ErrDegraded)
		}
		if usable {
			frames[kept], stamps[kept] = f, stamps[i]
			kept++
		}
	}
	if kept == 0 {
		return driver.Batch{}, err
	}
	return driver.Batch{Frame: ingest.Frame(r.keys, frames[:kept], stamps[:kept])}, err
}

// Close implements driver.Reader.
func (r *reader) Close() error {
	if r.sess != nil {
		r.sess.close()
	}
	return nil
}

// connect opens a session that plays the stream. A failure to reach the stream
// matches driver.ErrTemporary.
func (r *reader) connect(ctx context.Context) (*session, error) {
	c := &gortsplib.Client{
		Scheme:   r.url.Scheme,
		Host:     r.url.Host,
		Protocol: r.protocol,
		OnTransportSwitch: func(err error) {
			r.ins.L.Info("switched transport", zap.Error(err))
		},
		OnPacketsLost: func(lost uint64) {
			r.ins.L.Debug("lost RTP packets", zap.Uint64("count", lost))
		},
		OnDecodeError: func(err error) {
			r.ins.L.Debug("failed to decode RTP packet", zap.Error(err))
		},
	}
	if err := c.Start(); err != nil {
		return nil, errors.Mark(err, driver.ErrTemporary)
	}
	// A Close from another goroutine is the only way to cancel a request.
	stop := context.AfterFunc(ctx, c.Close)
	defer stop()
	s, err := r.play(c)
	if err != nil {
		c.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	return s, nil
}

func (r *reader) play(c *gortsplib.Client) (*session, error) {
	desc, _, err := c.Describe(r.url)
	if err != nil {
		return nil, errors.Mark(err, driver.ErrTemporary)
	}
	var h264 *format.H264
	media := desc.FindFormat(&h264)
	if media == nil {
		return nil, ErrNoH264
	}
	dec, err := h264.CreateDecoder()
	if err != nil {
		return nil, err
	}
	if _, err = c.Setup(desc.BaseURL, media, 0, 0); err != nil {
		return nil, errors.Mark(err, driver.ErrTemporary)
	}
	s := &session{
		client: c,
		frames: make(chan timed, maxBatch),
		failed: make(chan struct{}),
		closed: make(chan struct{}),
		done:   make(chan struct{}),
		h264:   video.H264{SPS: h264.SPS, PPS: h264.PPS},
		watch:  video.KeyframeWatch{Limit: r.limit},
	}
	// A virtual channel has no index, so its frames carry no time.
	if len(r.keys) > 1 {
		s.clock = &video.Clock{
			Rate:    uint32(h264.ClockRate()),
			Offset:  r.offset,
			Capture: r.capture,
		}
		c.OnPacketRTCP(media, func(pkt rtcp.Packet) {
			if sr, ok := pkt.(*rtcp.SenderReport); ok {
				s.clock.Report(sr.NTPTime, sr.RTPTime)
			}
		})
	}
	c.OnPacketRTP(media, h264, func(pkt *rtp.Packet) { s.handle(dec, pkt) })
	if _, err = c.Play(nil); err != nil {
		return nil, errors.Mark(err, driver.ErrTemporary)
	}
	go func() {
		s.err = c.Wait()
		close(s.done)
	}()
	return s, nil
}

// timed is a frame and its time.
type timed struct {
	frame telem.VideoFrame
	stamp telem.TimeStamp
}

// session is one connection to the stream.
type session struct {
	client *gortsplib.Client
	// clock times the frames of a persisted channel. It is nil for a virtual one.
	clock *video.Clock
	// frames holds the assembled frames that the reader has not taken yet.
	frames chan timed
	// failed closes after fatal is set.
	failed chan struct{}
	fatal  error
	// closed stops a callback that waits to send a frame.
	closed chan struct{}
	// done closes after err is set, when the connection ends.
	done chan struct{}
	err  error
	// h264 is used only by the RTP callback.
	h264  video.H264
	watch video.KeyframeWatch
}

// handle runs on the gortsplib read goroutine for each RTP packet.
func (s *session) handle(dec *rtph264.Decoder, pkt *rtp.Packet) {
	if s.fatal != nil {
		return
	}
	au, err := dec.Decode(pkt)
	if err != nil {
		// The decoder waits for more packets, or drops an access unit with lost
		// packets.
		return
	}
	f, err := s.h264.Frame(au)
	if err != nil {
		s.fatal = err
		close(s.failed)
		return
	}
	t := timed{frame: f}
	if s.clock != nil {
		var ok bool
		if t.stamp, ok = s.clock.Stamp(telem.Now(), pkt.Timestamp); !ok {
			return
		}
	}
	select {
	case s.frames <- t:
	case <-s.closed:
	}
}

func (s *session) close() {
	close(s.closed)
	s.client.Close()
	<-s.done
}
