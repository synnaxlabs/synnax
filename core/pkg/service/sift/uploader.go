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
	"context"
	"sync/atomic"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// uploadTask handles a single historical data upload to Sift. It is an ephemeral task
// that deletes itself upon completion.
type uploadTask struct {
	task       task.Task
	cfg        TaskConfig
	props      DeviceProperties
	factoryCfg FactoryConfig
	pool       *ClientPool
	running    atomic.Bool
}

func newUploadTask(
	t task.Task,
	cfg TaskConfig,
	props DeviceProperties,
	factoryCfg FactoryConfig,
	pool *ClientPool,
) *uploadTask {
	return &uploadTask{
		task:       t,
		cfg:        cfg,
		props:      props,
		factoryCfg: factoryCfg,
		pool:       pool,
	}
}

// Exec handles task commands.
func (u *uploadTask) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return u.start(ctx)
	default:
		return errors.Newf("unknown command: %s", cmd.Type)
	}
}

func (u *uploadTask) start(ctx context.Context) error {
	if u.running.Load() {
		return errors.New("upload already in progress")
	}
	u.running.Store(true)
	go u.run(ctx)
	return nil
}

func (u *uploadTask) run(ctx context.Context) {
	defer u.running.Store(false)
	defer u.deleteTask()

	u.setStatus(xstatus.VariantInfo, "Starting upload", true)

	// Retrieve the range
	var rng ranger.Range
	if err := u.factoryCfg.Ranger.NewRetrieve().
		WhereKeys(u.cfg.RangeKey).
		Entry(&rng).
		Exec(ctx, nil); err != nil {
		u.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to retrieve range").Error(), false)
		return
	}

	// Determine time bounds
	bounds := rng.TimeRange
	if u.cfg.TimeRange != nil {
		bounds = *u.cfg.TimeRange
	}

	// Retrieve channels
	var channels []channel.Channel
	if err := u.factoryCfg.Channel.NewRetrieve().
		WhereKeys(u.cfg.Channels...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		u.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to retrieve channels").Error(), false)
		return
	}

	// Build channel lookup map
	channelMap := make(map[channel.Key]channel.Channel)
	for _, ch := range channels {
		channelMap[ch.Key()] = ch
	}

	// Get Sift client
	client, err := u.pool.Get(ctx, u.props)
	if err != nil {
		u.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to connect to Sift").Error(), false)
		return
	}

	// Build flow config
	flow := u.buildFlowConfig(channels)

	// Get or create ingestion config
	ingestionCfg, err := client.GetOrCreateIngestionConfig(ctx, []FlowConfig{flow})
	if err != nil {
		u.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to create ingestion config").Error(), false)
		return
	}

	// Create run if requested
	var runID string
	if u.cfg.RunName != "" {
		run, err := client.CreateRun(ctx, u.cfg.RunName)
		if err != nil {
			u.factoryCfg.L.Warn("failed to create run", zap.Error(err))
		} else {
			runID = run.RunId
		}
	}

	// Open ingest stream
	stream, err := client.OpenIngestStream(ctx, ingestionCfg.IngestionConfigId)
	if err != nil {
		u.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to open ingest stream").Error(), false)
		return
	}
	defer stream.Close()

	// Open data iterator
	iter, err := u.factoryCfg.Framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   u.cfg.Channels,
		Bounds: bounds,
	})
	if err != nil {
		u.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to open iterator").Error(), false)
		return
	}
	defer iter.Close()

	// Stream data to Sift
	var frameCount int64
	for iter.SeekFirst(); iter.Valid(); iter.Next(telem.Second) {
		select {
		case <-ctx.Done():
			u.setStatus(xstatus.VariantWarning, "Upload cancelled", false)
			return
		default:
		}

		frame := iter.Value()
		if frame.Empty() {
			continue
		}
		frameCount++

		if err := u.sendFrame(stream, frame, channelMap, runID); err != nil {
			u.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to send data").Error(), false)
			return
		}

		if frameCount%100 == 0 {
			u.setStatus(xstatus.VariantInfo, "Uploading", true)
		}
	}

	if err := iter.Error(); err != nil {
		u.setStatus(xstatus.VariantError, errors.Wrap(err, "iterator error").Error(), false)
		return
	}

	u.setStatus(xstatus.VariantSuccess, "Upload completed", false)
}

func (u *uploadTask) buildFlowConfig(channels []channel.Channel) FlowConfig {
	cfgs := make([]ChannelConfig, 0, len(channels))
	for _, ch := range channels {
		if ch.IsIndex {
			continue
		}
		dt, err := MapDataType(ch.DataType)
		if err != nil {
			u.factoryCfg.L.Warn("skipping channel with unsupported type",
				zap.String("channel", ch.Name),
				zap.Error(err))
			continue
		}
		cfgs = append(cfgs, ChannelConfig{Name: ch.Name, DataType: dt})
	}
	return FlowConfig{Name: u.cfg.FlowName, Channels: cfgs}
}

func (u *uploadTask) sendFrame(
	stream *IngestStream,
	frame framer.Frame,
	channelMap map[channel.Key]channel.Channel,
	runID string,
) error {
	for key, series := range frame.Entries() {
		ch, ok := channelMap[key]
		if !ok || ch.IsIndex {
			continue
		}

		values, err := ConvertSeriesToValues(series)
		if err != nil {
			continue
		}

		// Calculate sample timestamps
		seriesLen := series.Len()
		var sampleSpan telem.TimeSpan
		if seriesLen > 1 {
			sampleSpan = telem.TimeSpan(int64(series.TimeRange.Span()) / int64(seriesLen-1))
		}

		for i, val := range values {
			ts := series.TimeRange.Start.Add(telem.TimeSpan(i) * sampleSpan)
			req := &ingestv1.IngestWithConfigDataStreamRequest{
				Flow:          u.cfg.FlowName,
				Timestamp:     timestamppb.New(ts.Time()),
				ChannelValues: []*ingestv1.IngestWithConfigDataChannelValue{u.toChannelValue(val, series.DataType)},
				RunId:         runID,
			}
			if err := stream.Send(req); err != nil {
				return err
			}
		}
	}
	return nil
}

func (u *uploadTask) toChannelValue(v any, dt telem.DataType) *ingestv1.IngestWithConfigDataChannelValue {
	switch dt {
	case telem.Float64T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Double{Double: v.(float64)},
		}
	case telem.Float32T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Float{Float: v.(float32)},
		}
	case telem.Int64T, telem.TimeStampT:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Int64{Int64: v.(int64)},
		}
	case telem.Int32T, telem.Int16T, telem.Int8T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Int32{Int32: v.(int32)},
		}
	case telem.Uint64T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Uint64{Uint64: v.(uint64)},
		}
	case telem.Uint32T, telem.Uint16T, telem.Uint8T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Uint32{Uint32: v.(uint32)},
		}
	default:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Double{Double: 0},
		}
	}
}

func (u *uploadTask) setStatus(variant xstatus.Variant, message string, running bool) {
	stat := task.Status{
		Key:     task.OntologyID(u.task.Key).String(),
		Name:    u.task.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: u.task.Key, Running: running},
	}
	if err := status.NewWriter[task.StatusDetails](
		u.factoryCfg.Status, nil,
	).Set(context.Background(), &stat); err != nil {
		u.factoryCfg.L.Error("failed to set status", zap.Error(err))
	}
}

func (u *uploadTask) deleteTask() {
	if err := u.factoryCfg.Task.NewWriter(nil).Delete(
		context.Background(), u.task.Key, false,
	); err != nil {
		u.factoryCfg.L.Error("failed to delete task", zap.Uint64("task", uint64(u.task.Key)), zap.Error(err))
	}
}

// Stop cancels the upload. Since the upload runs in a goroutine with context, this is
// a no-op - the context cancellation handles stopping.
func (u *uploadTask) Stop() error { return nil }

var _ driver.Task = (*uploadTask)(nil)
