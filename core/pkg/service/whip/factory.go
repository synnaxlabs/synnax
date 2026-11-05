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
	"net"
	"sync"
	"time"
	"uuid"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/video/ingest"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// FactoryConfig is the configuration for the WHIP task factory.
type FactoryConfig struct {
	// DB opens the transactions that status writes run in.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channel retrieves the video channels of tasks.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer opens the writers of tasks.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status writes task statuses.
	//
	// [REQUIRED]
	Status *status.Service
	alamos.Instrumentation
	// KeyframeLimit is the longest time that a stream may go without a keyframe
	// before its task warns that no viewer can start to show it.
	//
	// [OPTIONAL] - Defaults to 10s.
	KeyframeLimit time.Duration
	// UDPPort is the UDP port that receives the media of every WHIP session.
	//
	// [OPTIONAL] - Defaults to 0, which gives each session its own ports.
	UDPPort int
}

var (
	_ config.Config[FactoryConfig] = FactoryConfig{}
	// DefaultFactoryConfig is the default configuration for the WHIP task factory.
	DefaultFactoryConfig = FactoryConfig{KeyframeLimit: 10 * time.Second}
)

// Override implements config.Config.
func (c FactoryConfig) Override(other FactoryConfig) FactoryConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Status = override.Nil(c.Status, other.Status)
	c.KeyframeLimit = override.Numeric(c.KeyframeLimit, other.KeyframeLimit)
	c.UDPPort = override.Numeric(c.UDPPort, other.UDPPort)
	return c
}

// Validate implements config.Config.
func (c FactoryConfig) Validate() error {
	v := validate.New("whip.factory")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NotNil("status", c.Status)
	v.Positive("keyframe_limit", c.KeyframeLimit)
	v.Ternary("udp_port", c.UDPPort < 0 || c.UDPPort > 65535, "must be a port number")
	return v.Error()
}

// ErrNotRunning is returned for a publish to a task that does not run on this Core.
var ErrNotRunning = errors.Wrap(
	query.ErrNotFound,
	"the task is not running on this Core",
)

// Factory configures WHIP read tasks and connects the encoders that publish to them.
// It must be closed after the tasks it configured stop. Safe for concurrent use.
type Factory struct {
	cfg      FactoryConfig
	reporter driver.ConfigReporter
	api      *webrtc.API
	closer   xio.MultiCloser
	mu       struct {
		sync.Mutex
		inboxes map[uuid.UUID]*inbox
	}
}

var _ driver.Factory = (*Factory)(nil)

// OpenFactory validates the configuration and opens a Factory. It listens on UDPPort
// when the port is not zero.
func OpenFactory(cfgs ...FactoryConfig) (*Factory, error) {
	cfg, err := config.New(DefaultFactoryConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	f := &Factory{
		cfg: cfg,
		reporter: driver.ConfigReporter{
			DB:              cfg.DB,
			Status:          cfg.Status,
			Instrumentation: cfg.Instrumentation,
		},
	}
	f.mu.inboxes = make(map[uuid.UUID]*inbox)
	if f.api, err = f.newAPI(); err != nil {
		return nil, errors.Combine(err, f.closer.Close())
	}
	return f, nil
}

func (f *Factory) newAPI() (*webrtc.API, error) {
	media := &webrtc.MediaEngine{}
	feedback := []webrtc.RTCPFeedback{
		{Type: "goog-remb"},
		{Type: "ccm", Parameter: "fir"},
		{Type: "nack"},
		{Type: "nack", Parameter: "pli"},
	}
	for i, profile := range []string{"42001f", "42e01f", "4d001f", "64001f"} {
		if err := media.RegisterCodec(webrtc.RTPCodecParameters{
			MimeType:  webrtc.MimeTypeH264,
			ClockRate: 90000,
			SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;" +
				"profile-level-id=" + profile,
			RTCPFeedback: feedback,
			PayloadType:  webrtc.PayloadType(102 + 2*i),
		}, webrtc.RTPCodecTypeVideo); err != nil {
			return nil, err
		}
	}
	interceptors := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(media, interceptors); err != nil {
		return nil, err
	}
	settings := webrtc.SettingEngine{}
	if f.cfg.UDPPort != 0 {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: f.cfg.UDPPort})
		if err != nil {
			return nil, err
		}
		mux := webrtc.NewICEUDPMux(nil, conn)
		f.closer = append(f.closer, mux)
		settings.SetICEUDPMux(mux)
	}
	return webrtc.NewAPI(
		webrtc.WithMediaEngine(media),
		webrtc.WithInterceptorRegistry(interceptors),
		webrtc.WithSettingEngine(settings),
	), nil
}

// Close releases the UDP port of the factory.
func (f *Factory) Close() error { return f.closer.Close() }

// ConfigureTask implements driver.Factory.
func (f *Factory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	if t.Type != ReadTaskType {
		return nil, driver.ErrTaskNotHandled
	}
	var cfg ReadConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		f.reporter.Report(ctx, t, cmdKey, false, err)
		return nil, err
	}
	rt, err := f.configureRead(ctx, t, cfg)
	if err != nil {
		f.reporter.Report(ctx, t, cmdKey, cfg.AutoStart, err)
		return nil, err
	}
	// A successful configure writes no status: the start that follows it answers the
	// command, and a "configured" status would answer it first with running false.
	if cfg.AutoStart {
		if err = rt.Start(ctx, driver.NoCommand); err != nil {
			return nil, err
		}
	}
	return rt, nil
}

// InitialTasks implements driver.Factory.
func (f *Factory) InitialTasks() []task.Task { return nil }

// Name implements driver.Factory.
func (f *Factory) Name() string { return "whip" }

func (f *Factory) configureRead(
	ctx context.Context,
	t task.Task,
	cfg ReadConfig,
) (*driver.ReadTask, error) {
	keys, err := ingest.Channels(ctx, f.cfg.Channel, cfg.Channel)
	if err != nil {
		return nil, err
	}
	src := &source{
		factory: f,
		task:    t.Key,
		channel: cfg.Channel,
		keys:    keys,
		offset:  cfg.Offset,
		capture: cfg.TimeSource == TimeSourceCapture,
	}
	mode := framer.WriterModePersistStream
	if cfg.DataSavingDisabled {
		mode = framer.WriterModeStreamOnly
	}
	return driver.NewReadTask(driver.ReadTaskConfig{
		Instrumentation: f.cfg.Child(t.Key.String()),
		Source:          src,
		DB:              f.cfg.DB,
		Status:          f.cfg.Status,
		Framer:          f.cfg.Framer,
		Channels:        keys,
		Task:            t,
		Mode:            mode,
	})
}

// Channel returns the video channel of a task that runs on this Core. It returns
// ErrNotRunning for any other task.
func (f *Factory) Channel(task uuid.UUID) (channel.Key, error) {
	in, err := f.inbox(task)
	if err != nil {
		return 0, err
	}
	return in.channel, nil
}

// Publish connects an encoder to a running task. It takes the SDP offer of the
// encoder and returns the SDP answer and the key of the new session. A publish
// replaces the session that the task had. It returns ErrNotRunning for a task that
// does not run on this Core, and an error that matches validate.ErrValidation for an
// invalid offer.
func (f *Factory) Publish(
	ctx context.Context,
	task uuid.UUID,
	offer string,
) (answer string, key uuid.UUID, err error) {
	in, err := f.inbox(task)
	if err != nil {
		return "", uuid.Nil(), err
	}
	return in.publish(ctx, f.api, offer)
}

// Unpublish ends a session of a task. It returns an error that matches
// query.ErrNotFound when the task has no session with that key.
func (f *Factory) Unpublish(task, key uuid.UUID) error {
	in, err := f.inbox(task)
	if err != nil {
		return err
	}
	return in.unpublish(key)
}

func (f *Factory) inbox(task uuid.UUID) (*inbox, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	in, ok := f.mu.inboxes[task]
	if !ok {
		return nil, ErrNotRunning
	}
	return in, nil
}

func (f *Factory) register(task uuid.UUID, in *inbox) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mu.inboxes[task] = in
}

func (f *Factory) unregister(task uuid.UUID, in *inbox) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.mu.inboxes[task] == in {
		delete(f.mu.inboxes, task)
	}
}
