// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/sift/ingest"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// uploadKeyPrefix starts the key of the status of every upload.
const uploadKeyPrefix = "sift_upload:"

// UploadStatus is the status of an upload to Sift.
type UploadStatus = status.Status[UploadDetails]

type uploadArgs struct {
	// Device is the key of the Sift device to send to.
	Device device.Key `json:"device"`
	// Range is the key of the range whose data to send.
	Range ranger.Key `json:"range"`
	// Channels are the channels to send.
	Channels []channel.Key `json:"channels"`
	// Asset is the name of the Sift asset that receives the data.
	Asset string `json:"asset"`
}

func (a uploadArgs) validate() error {
	v := validate.New("sift.upload")
	v.NotEmptyString("device", a.Device)
	v.Ternary("range", a.Range == uuid.Nil(), "required")
	v.NotEmptySlice("channels", a.Channels)
	v.NotEmptyString("asset", a.Asset)
	return v.Error()
}

type uploadResult struct {
	// Status is the key of the status of the upload.
	Status status.Key `json:"status"`
}

// queueUpload decodes and validates the arguments of an upload command, then writes
// the pending status of the upload, which the uploader of the rack picks up.
func queueUpload(
	ctx context.Context,
	cfg FactoryConfig,
	rk rack.Key,
	raw msgpack.EncodedJSON,
) (status.Key, error) {
	var args uploadArgs
	if err := raw.Unmarshal(&args); err != nil {
		return "", errors.Wrap(err, "invalid arguments")
	}
	if err := args.validate(); err != nil {
		return "", err
	}
	var dev device.Device
	if err := cfg.Device.NewRetrieve().
		Where(device.MatchKeys(args.Device)).
		Entry(&dev).
		Exec(ctx, cfg.DB); err != nil {
		return "", err
	}
	if _, err := decodeProperties(dev); err != nil {
		return "", err
	}
	if dev.Rack != rk {
		return "", validate.Errorf("device %s belongs to another rack", dev.Name)
	}
	if err := ingest.Validate(ctx, cfg.DB, cfg.Channel, args.Channels); err != nil {
		return "", err
	}
	var rng ranger.Range
	if err := cfg.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(args.Range)).
		Entry(&rng).
		Exec(ctx, cfg.DB); err != nil {
		return "", err
	}
	// A range that has not ended sends the data written so far.
	tr, now := rng.TimeRange, telem.Now()
	if tr.Start > now {
		return "", validate.Errorf(
			"range %s starts in the future, so it has no data yet",
			rng.Name,
		)
	}
	tr.End = min(tr.End, now)
	if tr.Span() <= 0 {
		return "", validate.Errorf("range %s has no time span to upload", rng.Name)
	}
	s := UploadStatus{
		Key:     uploadKeyPrefix + uuid.NewV7().String(),
		Name:    rng.Name,
		Variant: status.VariantLoading,
		Message: "Waiting to upload to Sift",
		Time:    telem.Now(),
		Details: UploadDetails{
			Device:    dev.Key,
			Range:     rng.Key,
			TimeRange: tr,
			Channels:  args.Channels,
			Asset:     args.Asset,
		},
	}
	if err := cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return cfg.Status.NewWriter(tx).Set(ctx, &s)
	}); err != nil {
		return "", err
	}
	return s.Key, nil
}

// uploader runs the pending uploads to the Sift devices of one rack, one at a time
// per device, oldest first. The pending statuses are its queue: it resumes them after
// a restart, and deleting one cancels its upload.
type uploader struct {
	cfg        FactoryConfig
	rack       rack.Key
	sCtx       signal.Context
	cancel     context.CancelFunc
	disconnect observe.Disconnect
	wake       chan struct{}
	mu         sync.Mutex
	// running holds the upload in progress on each device.
	running map[device.Key]*upload
}

type upload struct {
	key    status.Key
	cancel context.CancelFunc
}

// openUploader starts the uploader of rk. The caller must close it.
func openUploader(cfg FactoryConfig, rk rack.Key) *uploader {
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	u := &uploader{
		cfg:     cfg,
		rack:    rk,
		sCtx:    sCtx,
		cancel:  cancel,
		wake:    make(chan struct{}, 1),
		running: make(map[device.Key]*upload),
	}
	u.disconnect = cfg.Status.Observe().OnChange(u.observe)
	u.signal()
	sCtx.Go(u.run, signal.RecoverWithErrOnPanic())
	return u
}

// Close stops every upload in progress. Each one saves how far Sift confirmed it, and
// the next uploader of the rack resumes it from there.
func (u *uploader) Close() error {
	u.disconnect()
	u.cancel()
	return errors.Skip(u.sCtx.Wait(), context.Canceled)
}

func (u *uploader) signal() {
	select {
	case u.wake <- struct{}{}:
	default:
	}
}

func (u *uploader) observe(
	_ context.Context,
	r gorp.TxReader[status.Key, status.Status[any]],
) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for c := range r {
		if !strings.HasPrefix(c.Key, uploadKeyPrefix) {
			continue
		}
		if c.Kind != gorp.ChangeDelete {
			u.signal()
			continue
		}
		for _, up := range u.running {
			if up.key == c.Key {
				up.cancel()
			}
		}
	}
}

func (u *uploader) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-u.wake:
		}
		if err := u.schedule(ctx); err != nil && ctx.Err() == nil {
			u.cfg.L.Error("failed to start uploads to Sift", zap.Error(err))
		}
	}
}

// schedule starts the oldest pending upload of each idle device on the rack.
func (u *uploader) schedule(ctx context.Context) error {
	// Reading under u.mu keeps an upload that finishes or is deleted meanwhile from
	// starting again.
	u.mu.Lock()
	defer u.mu.Unlock()
	var pending []UploadStatus
	if err := u.cfg.Status.NewRetrieve[UploadDetails]().
		WherePrefix(uploadKeyPrefix).
		Where(status.MatchVariants[UploadDetails](
			status.VariantLoading,
			status.VariantWarning,
		)).
		Entries(&pending).
		Exec(ctx, u.cfg.DB); err != nil {
		return err
	}
	var devs []device.Device
	if err := u.cfg.Device.NewRetrieve().
		Where(device.MatchMakes(Make)).
		Entries(&devs).
		Exec(ctx, u.cfg.DB); err != nil {
		return err
	}
	racks := make(map[device.Key]rack.Key, len(devs))
	for _, dev := range devs {
		racks[dev.Key] = dev.Rack
	}
	// Keys hold version 7 UUIDs, so key order is the order of the queue.
	slices.SortFunc(pending, func(a, b UploadStatus) int {
		return cmp.Compare(a.Key, b.Key)
	})
	for _, s := range pending {
		dev := s.Details.Device
		if _, busy := u.running[dev]; busy {
			continue
		}
		// An upload whose device is gone fails on whichever rack finds it first.
		if rk, ok := racks[dev]; ok && rk != u.rack {
			continue
		}
		u.start(s)
	}
	return nil
}

// start runs the upload of s. u.mu must be held.
func (u *uploader) start(s UploadStatus) {
	ctx, cancel := context.WithCancel(u.sCtx)
	u.running[s.Details.Device] = &upload{key: s.Key, cancel: cancel}
	u.sCtx.Go(func(context.Context) error {
		defer func() {
			cancel()
			u.mu.Lock()
			delete(u.running, s.Details.Device)
			u.mu.Unlock()
			u.signal()
		}()
		u.finish(ctx, &s, u.send(ctx, &s))
		return nil
	}, signal.WithKeyf("upload-%s", s.Key), signal.RecoverWithErrOnPanic())
}

// finish writes the final status of the upload. A stopped upload keeps its pending
// status so that it resumes.
func (u *uploader) finish(ctx context.Context, s *UploadStatus, err error) {
	switch {
	case ctx.Err() != nil:
		return
	case err != nil:
		s.Variant = status.VariantError
		s.Message = "Failed to upload to Sift"
		s.Description = err.Error()
	default:
		s.Variant = status.VariantSuccess
		s.Message = "Uploaded to Sift"
		s.Description = uploadProgress(s.Details)
	}
	u.write(ctx, s)
}

// send sends the data of s to Sift and makes its run, saving progress in s as Sift
// confirms it.
func (u *uploader) send(ctx context.Context, s *UploadStatus) error {
	d := &s.Details
	var dev device.Device
	if err := u.cfg.Device.NewRetrieve().
		Where(device.MatchKeys(d.Device)).
		Entry(&dev).
		Exec(ctx, u.cfg.DB); err != nil {
		return err
	}
	props, err := decodeProperties(dev)
	if err != nil {
		return err
	}
	conn, err := connect(ctx, u.cfg.Secret, u.cfg.DB, props)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			u.cfg.L.Warn("failed to close connection", zap.Error(err))
		}
	}()
	uploading := func() {
		s.Variant = status.VariantLoading
		s.Message = "Uploading to Sift"
		s.Description = uploadProgress(*d)
	}
	uploading()
	u.write(ctx, s)
	retrying := func(ctx context.Context, err error) {
		s.Variant = status.VariantWarning
		s.Message = "Retrying upload to Sift"
		s.Description = err.Error()
		u.write(ctx, s)
	}
	e, err := ingest.New(ctx, ingest.Config{
		DB:                 u.cfg.DB,
		Instrumentation:    u.cfg.Child("upload"),
		Framer:             u.cfg.Framer,
		Channel:            u.cfg.Channel,
		Conn:               conn,
		Asset:              d.Asset,
		Keys:               d.Channels,
		CheckpointInterval: u.cfg.CheckpointInterval,
		OnCheckpoint: func(ctx context.Context, cp ingest.Checkpoint) error {
			d.Positions = toPositions(cp.Positions)
			d.Samples += cp.Samples
			uploading()
			u.write(ctx, s)
			return nil
		},
		OnRetry:    retrying,
		OnRunRetry: retrying,
	})
	if err != nil {
		return err
	}
	if err = e.Upload(ctx, d.TimeRange, fromPositions(d.Positions)); err != nil {
		return err
	}
	var rng ranger.Range
	err = u.cfg.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(d.Range)).
		Entry(&rng).
		Exec(ctx, u.cfg.DB)
	// A range deleted during the upload has no run to make.
	if errors.Is(err, query.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	tags, err := rangeTags(ctx, u.cfg, rng)
	if err != nil {
		return err
	}
	_, err = e.SetRun(ctx, ingest.Run{
		Key:       rng.Key.String(),
		Name:      rng.Name,
		TimeRange: d.TimeRange,
		Tags:      tags,
	})
	return err
}

// write saves s unless its status was deleted, which cancels the upload. It saves even
// after ctx is done, so that a stopped upload keeps its progress.
func (u *uploader) write(ctx context.Context, s *UploadStatus) {
	s.Time = telem.Now()
	ctx = context.WithoutCancel(ctx)
	if err := u.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		// Transactions run one at a time, so no delete lands between the check and the
		// set.
		var stored UploadStatus
		err := u.cfg.Status.NewRetrieve[UploadDetails]().
			Where(status.MatchKeys[UploadDetails](s.Key)).
			Entry(&stored).
			Exec(ctx, tx)
		if errors.Is(err, query.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return u.cfg.Status.NewWriter(tx).Set(ctx, s)
	}); err != nil {
		u.cfg.L.Error(
			"failed to save upload status",
			zap.String("status", s.Key),
			zap.Error(err),
		)
	}
}

func uploadProgress(d UploadDetails) string {
	return fmt.Sprintf("Sent %s to asset %s.", sampleCount(d.Samples), d.Asset)
}

// sampleCount formats n samples, with a comma between each group of three digits.
func sampleCount(n int64) string {
	if n == 1 {
		return "1 sample"
	}
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s + " samples"
}

func toPositions(p ingest.Positions) []Position {
	positions := make([]Position, 0, len(p))
	for index, t := range p {
		positions = append(positions, Position{Index: index, Time: t})
	}
	slices.SortFunc(positions, func(a, b Position) int {
		return cmp.Compare(a.Index, b.Index)
	})
	return positions
}

func fromPositions(positions []Position) ingest.Positions {
	p := make(ingest.Positions, len(positions))
	for _, pos := range positions {
		p[pos.Index] = pos.Time
	}
	return p
}
