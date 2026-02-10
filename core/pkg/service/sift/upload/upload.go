// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package upload provides functionality for uploading data to Sift.
package upload

import (
	"context"
	"encoding/json"
	"maps"
	"sync"

	"github.com/google/uuid"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv1 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v1"
	runsv2 "github.com/sift-stack/sift/go/gen/sift/runs/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	siftdevice "github.com/synnaxlabs/synnax/pkg/service/sift/device"
	"github.com/synnaxlabs/synnax/pkg/service/sift/transform"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TaskType is the task type for Sift uploads.
const TaskType = "sift_upload"

const (
	orchestratorAddr address.Address = "orchestrator"
	ingesterAddr     address.Address = "ingester"
)

// Config contains the configuration for an upload task.
type Config struct {
	// DeviceKey references the Sift device containing connection config.
	DeviceKey string `json:"device_key"`
	// AssetName is the Sift asset name to upload to.
	AssetName string `json:"asset_name"`
	// FlowName is the Sift flow name for this upload.
	FlowName string `json:"flow_name"`
	// RunName is the Sift run name. A run will be created with this name.
	RunName string `json:"run_name"`
	// Channels are the Synnax channel keys to upload.
	Channels []channel.Key `json:"channels"`
	// TimeRange is the time range to upload.
	TimeRange telem.TimeRange `json:"time_range"`
}

func parseConfig(s string) (Config, error) {
	var c Config
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return c, errors.Wrap(err, "failed to parse upload task config")
	}
	return c, nil
}

// channelGroup represents a set of channels that share the same index. Each group has
// its own ingestion config on Sift but shares the same run.
type channelGroup struct {
	index           channel.Key
	DataChannelKeys []channel.Key
	configID        string
}

// GroupChannelsByIndex groups channels by their index channel.
// Returns a map from index key to the group of data channels that use that index.
func GroupChannelsByIndex(channels []channel.Channel) map[channel.Key]*channelGroup {
	groups := make(map[channel.Key]*channelGroup)
	for _, ch := range channels {
		IndexKey := ch.Index()
		if _, ok := groups[IndexKey]; !ok {
			groups[IndexKey] = &channelGroup{index: IndexKey}
		}
		groups[IndexKey].DataChannelKeys = append(groups[IndexKey].DataChannelKeys, ch.Key())
	}
	return groups
}

// Dependencies contains the dependencies needed for configuring upload tasks.
type Dependencies struct {
	Device  *device.Service
	Framer  *framer.Service
	Channel *channel.Service
	Status  *status.Service
	Task    *task.Service
	Pool    *client.Pool
	L       *alamos.Logger
}

// Task handles uploading data to Sift as a driver.Task.
// It manages multiple channel groups, each with its own ingestion config but
// sharing the same run ID.
type Task struct {
	task     task.Task
	cfg      Config
	deps     Dependencies
	client   client.Client
	ingester client.Ingester
	iter     framer.StreamIterator
	groups   []*channelGroup
	runID    string

	mu      sync.Mutex
	wg      sync.WaitGroup
	cancel  context.CancelFunc
	running bool
}

var _ driver.Task = (*Task)(nil)

func (u *Task) run(ctx context.Context) {
	u.mu.Lock()
	if u.running {
		u.cancel()
	}
	u.running = true
	u.wg.Add(1)
	ctx, u.cancel = context.WithCancel(ctx)
	u.mu.Unlock()

	defer func() {
		u.ingester.Close()
		u.mu.Lock()
		u.running = false
		u.cancel = nil
		u.mu.Unlock()
		u.wg.Done()
	}()

	u.setStatus(xstatus.VariantInfo, "Uploading", true)

	err := u.streamData(ctx)
	if err != nil {
		if ctx.Err() != nil {
			u.setStatus(xstatus.VariantWarning, "Upload cancelled", false)
		} else {
			u.setStatus(xstatus.VariantError, err.Error(), false)
		}
		return
	}

	u.setStatus(xstatus.VariantSuccess, "Upload completed", false)
	u.deleteTask()
}

func (u *Task) streamData(ctx context.Context) error {
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()

	orchestrator := &uploadOrchestrator{
		iter:     u.iter,
		groups:   u.groups,
		runID:    u.runID,
		flowName: u.cfg.FlowName,
	}

	pipe := plumber.New()
	plumber.SetSource(pipe, orchestratorAddr, orchestrator)
	plumber.SetSink(pipe, ingesterAddr, u.ingester)
	plumber.MustConnect[*ingestv1.IngestWithConfigDataStreamRequest](
		pipe, orchestratorAddr, ingesterAddr, 1,
	)

	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return sCtx.Wait()
}

// uploadOrchestrator drives the iterator and outputs ingest requests for
// multiple channel groups. Each group has its own ingestion config ID but
// shares the same run ID.
type uploadOrchestrator struct {
	confluence.AbstractUnarySource[*ingestv1.IngestWithConfigDataStreamRequest]
	iter     framer.StreamIterator
	groups   []*channelGroup
	runID    string
	flowName string
}

func (o *uploadOrchestrator) Flow(sCtx signal.Context, _ ...confluence.Option) {
	// Set up iterator flow
	iterRequests := confluence.NewStream[framer.IteratorRequest](2)
	iterResponses := confluence.NewStream[framer.IteratorResponse](50)
	o.iter.InFrom(iterRequests)
	o.iter.OutTo(iterResponses)
	o.iter.Flow(sCtx, confluence.CloseOutputInletsOnExit())

	sCtx.Go(func(ctx context.Context) error {
		return o.run(ctx, iterRequests, iterResponses)
	}, signal.WithKey("upload-orchestrator"))
}

func (o *uploadOrchestrator) run(
	ctx context.Context,
	iterRequests *confluence.Stream[framer.IteratorRequest],
	iterResponses *confluence.Stream[framer.IteratorResponse],
) error {
	defer func() {
		iterRequests.Close()
		o.Out.Close()
	}()

	// Seek to first position
	iterRequests.Inlet() <- framer.IteratorRequest{Command: iterator.CommandSeekFirst}
	ack := <-iterResponses.Outlet()
	if !ack.Ack {
		return nil
	}

	// Iterate through data
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		iterRequests.Inlet() <- framer.IteratorRequest{
			Command: iterator.CommandNext,
			Span:    iterator.AutoSpan,
		}

		var frames []framer.Frame
		for res := range iterResponses.Outlet() {
			if res.Variant == iterator.ResponseVariantAck {
				if res.Error != nil {
					return errors.Wrap(res.Error, "iterator error")
				}
				if !res.Ack {
					return nil
				}
				break
			}
			frames = append(frames, res.Frame)
		}

		for _, fr := range frames {
			if fr.Empty() {
				continue
			}
			if err := o.sendFrame(ctx, fr); err != nil {
				return err
			}
		}
	}
}

func (o *uploadOrchestrator) sendFrame(ctx context.Context, frame framer.Frame) error {
	entries := maps.Collect(frame.Entries())

	// Process each channel group independently
	for _, group := range o.groups {
		indexSeries := entries[group.index]

		channelValues := make([][]*ingestv1.IngestWithConfigDataChannelValue, len(group.DataChannelKeys))
		var numSamples int
		var timestamps []telem.TimeStamp

		for i, key := range group.DataChannelKeys {
			series, ok := entries[key]
			if !ok {
				continue
			}
			values, err := transform.SeriesToProtoValues(series)
			if err != nil {
				continue
			}
			channelValues[i] = values
			if numSamples == 0 {
				numSamples = len(values)
				timestamps = getTimestamps(series, indexSeries)
			}
		}

		if numSamples == 0 {
			continue
		}

		for i := 0; i < numSamples; i++ {
			row := make([]*ingestv1.IngestWithConfigDataChannelValue, len(group.DataChannelKeys))
			for j, values := range channelValues {
				if values != nil && i < len(values) {
					row[j] = values[i]
				}
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case o.Out.Inlet() <- &ingestv1.IngestWithConfigDataStreamRequest{
				IngestionConfigId: group.configID,
				Flow:              o.flowName,
				Timestamp:         timestamppb.New(timestamps[i].Time()),
				RunId:             o.runID,
				ChannelValues:     row,
			}:
			}
		}
	}

	return nil
}

func getTimestamps(series, indexSeries telem.Series) []telem.TimeStamp {
	n := series.Len()

	if indexSeries.Len() == n && indexSeries.DataType == telem.TimeStampT {
		return telem.UnmarshalSlice[telem.TimeStamp](indexSeries.Data, indexSeries.DataType)
	}

	timestamps := make([]telem.TimeStamp, n)
	if n == 1 {
		timestamps[0] = series.TimeRange.Start
	} else {
		span := series.TimeRange.Span()
		step := telem.TimeSpan(int64(span) / int64(n-1))
		for i := range timestamps {
			timestamps[i] = series.TimeRange.Start.Add(telem.TimeSpan(i) * step)
		}
	}
	return timestamps
}

func (*Task) Exec(context.Context, task.Command) error {
	return driver.ErrUnsupportedCommand
}

func (u *Task) Stop() error {
	u.mu.Lock()
	if u.cancel != nil {
		u.cancel()
	}
	u.mu.Unlock()
	u.wg.Wait()
	return nil
}

func (u *Task) setStatus(variant xstatus.Variant, message string, running bool) {
	stat := task.Status{
		Key:     task.OntologyID(u.task.Key).String(),
		Name:    u.task.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: u.task.Key, Running: running},
	}
	if err := status.NewWriter[task.StatusDetails](
		u.deps.Status, nil,
	).Set(context.Background(), &stat); err != nil {
		u.deps.L.Error("failed to set status", zap.Error(err))
	}
}

func (u *Task) deleteTask() {
	if err := u.deps.Task.NewWriter(nil).Delete(
		context.Background(), u.task.Key, false,
	); err != nil {
		u.deps.L.Error("failed to delete task",
			zap.Uint64("task", uint64(u.task.Key)),
			zap.Error(err))
	}
}

// Configure creates and configures a new upload task.
func Configure(
	ctx driver.Context,
	t task.Task,
	deps Dependencies,
	setStatus func(driver.Context, task.Task, xstatus.Variant, string, bool),
) (driver.Task, error) {
	cfg, err := parseConfig(t.Config)
	cfg.FlowName = uuid.New().String()
	if err != nil {
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	var dev device.Device
	if err := deps.Device.NewRetrieve().
		WhereKeys(cfg.DeviceKey).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}
	if dev.Make != siftdevice.Make {
		err := errors.Newf("device make %s is not supported", dev.Make)
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}
	if dev.Model != siftdevice.Model {
		err := errors.Newf("device model %s is not supported", dev.Model)
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}
	props, err := siftdevice.ParseProperties(dev.Properties)
	if err != nil {
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	siftClient, err := deps.Pool.Get(ctx, props.URI, props.APIKey)
	if err != nil {
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	var channels []channel.Channel
	if err := deps.Channel.NewRetrieve().
		WhereKeys(cfg.Channels...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, errors.Wrap(err, "failed to retrieve channels")
	}

	// Build channel name/type map for creating flow configs
	channelMap := make(map[channel.Key]channel.Channel)
	for _, ch := range channels {
		channelMap[ch.Key()] = ch
	}

	// Group channels by their index
	groups := GroupChannelsByIndex(channels)
	if len(groups) == 0 {
		err := errors.New("no valid channels to upload")
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	// Create ingestion config for each group
	groupSlice := make([]*channelGroup, 0, len(groups))
	groupNum := 0
	for _, group := range groups {
		var flowChannels []*ingestionconfigsv1.ChannelConfig
		for _, key := range group.DataChannelKeys {
			ch := channelMap[key]
			dt, err := transform.DataType(ch.DataType)
			if err != nil {
				err = errors.Wrapf(err, "channel %s has unsupported data type %s", ch.Name, ch.DataType)
				setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
				return nil, err
			}
			flowChannels = append(flowChannels, &ingestionconfigsv1.ChannelConfig{
				Name:     ch.Name,
				DataType: dt,
			})
		}
		if len(flowChannels) == 0 {
			continue
		}

		ingestionCfgRes, err := siftClient.CreateIngestionConfig(
			ctx,
			&ingestionconfigsv1.CreateIngestionConfigRequest{
				AssetName: cfg.AssetName,
				Flows:     []*ingestionconfigsv1.FlowConfig{{Name: cfg.FlowName, Channels: flowChannels}},
			},
		)
		if err != nil {
			err := errors.Wrap(err, "failed to create ingestion config")
			setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
			return nil, err
		}

		group.configID = ingestionCfgRes.IngestionConfig.IngestionConfigId
		groupSlice = append(groupSlice, group)
		groupNum++
	}

	if len(groupSlice) == 0 {
		err := errors.New("no valid channels to upload")
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	// Create a single run shared across all groups
	runRes, err := siftClient.CreateRun(ctx, &runsv2.CreateRunRequest{
		Name:      cfg.RunName,
		StartTime: timestamppb.New(cfg.TimeRange.Start.Time()),
		StopTime:  timestamppb.New(cfg.TimeRange.End.Time()),
	})
	if err != nil {
		err := errors.Wrap(err, "failed to create run")
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	ingester, err := siftClient.OpenIngester(ctx)
	if err != nil {
		err := errors.Wrap(err, "failed to open ingester")
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	iter, err := deps.Framer.NewStreamIterator(ctx, framer.IteratorConfig{
		Keys:   cfg.Channels,
		Bounds: cfg.TimeRange,
	})
	if err != nil {
		err := errors.Wrap(err, "failed to create stream iterator")
		err = errors.Combine(err, ingester.Close())
		setStatus(ctx, t, xstatus.VariantError, err.Error(), false)
		return nil, err
	}

	uploadTask := &Task{
		task:     t,
		cfg:      cfg,
		deps:     deps,
		client:   siftClient,
		ingester: ingester,
		iter:     iter,
		groups:   groupSlice,
		runID:    runRes.Run.RunId,
	}

	setStatus(ctx, t, xstatus.VariantSuccess, "Uploading to Sift", true)
	go uploadTask.run(ctx)
	return uploadTask, nil
}
