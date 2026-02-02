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
	"sync"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Uploader handles uploading historical data to Sift.
type Uploader struct {
	Client     client.Client
	Framer     *framer.Service
	ChannelSvc *channel.Service

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

// UploadParams contains the parameters for an upload operation.
type UploadParams struct {
	ClientKey string
	AssetName string
	FlowName  string
	RunName   string
	Channels  []channel.Key
	TimeRange telem.TimeRange
}

// Upload starts an upload operation. It runs synchronously and returns when complete.
func (u *Uploader) Upload(ctx context.Context, params UploadParams) error {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return errors.New("upload already in progress")
	}
	u.running = true
	ctx, u.cancel = context.WithCancel(ctx)
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		u.running = false
		u.cancel = nil
		u.mu.Unlock()
	}()

	return u.doUpload(ctx, params)
}

// Stop cancels an in-progress upload.
func (u *Uploader) Stop() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.cancel != nil {
		u.cancel()
	}
}

func (u *Uploader) doUpload(ctx context.Context, params UploadParams) error {
	// Retrieve channel metadata
	var channels []channel.Channel
	if err := u.ChannelSvc.NewRetrieve().
		WhereKeys(params.Channels...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve channels")
	}

	// Build channel lookup map, flow config, and ordered data channel keys
	channelMap := make(map[channel.Key]channel.Channel)
	var flowChannels []*client.ChannelConfig
	var dataChannelKeys []channel.Key
	var indexChannelKey channel.Key
	for _, ch := range channels {
		channelMap[ch.Key()] = ch
		if ch.IsIndex {
			indexChannelKey = ch.Key()
			continue
		}
		dt, err := MapDataType(ch.DataType)
		if err != nil {
			continue
		}
		flowChannels = append(flowChannels, &client.ChannelConfig{
			Name:     ch.Name,
			DataType: dt,
		})
		dataChannelKeys = append(dataChannelKeys, ch.Key())
	}

	if len(flowChannels) == 0 {
		return errors.New("no valid channels to upload")
	}

	// Create ingestion config
	ingestionCfgRes, err := u.Client.CreateIngestionConfig(ctx, &client.CreateIngestionConfigRequest{
		ClientKey: params.ClientKey,
		AssetName: params.AssetName,
		Flows:     []*client.FlowConfig{{Name: params.FlowName, Channels: flowChannels}},
	})
	if err != nil {
		return err
	}
	configID := ingestionCfgRes.IngestionConfig.IngestionConfigId

	// Create run
	runRes, err := u.Client.CreateRun(ctx, &client.CreateRunRequest{
		Name:      params.RunName,
		StartTime: timestamppb.New(params.TimeRange.Start.Time()),
		StopTime:  timestamppb.New(params.TimeRange.End.Time()),
	})
	if err != nil {
		return err
	}
	runID := runRes.Run.RunId

	// Open ingest stream
	stream, err := u.Client.OpenIngester(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Open data iterator
	iter, err := u.Framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   params.Channels,
		Bounds: params.TimeRange,
	})
	if err != nil {
		return errors.Wrap(err, "failed to open iterator")
	}
	defer iter.Close()

	// Set up confluence flow
	sCtx, cancel := signal.Isolated()
	defer cancel()
	requests := confluence.NewStream[*client.DataStreamRequest](1)
	stream.InFrom(requests)
	stream.Flow(sCtx)

	// Stream data to Sift
	if !iter.SeekFirst() {
		return nil // No data to upload
	}
	for iter.Next(telem.Second) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		frame := iter.Value()
		if frame.Empty() {
			continue
		}

		if err := u.sendFrame(ctx, requests, frame, indexChannelKey, dataChannelKeys, configID, params.FlowName, runID); err != nil {
			return err
		}
	}

	if err := iter.Error(); err != nil {
		return errors.Wrap(err, "iterator error")
	}

	// Close the requests inlet to signal no more data and wait for the flow to finish
	requests.Close()
	return errors.Skip(sCtx.Wait(), context.Canceled)
}

func (u *Uploader) sendFrame(
	ctx context.Context,
	requests confluence.Inlet[*client.DataStreamRequest],
	frame framer.Frame,
	indexChannelKey channel.Key,
	dataChannelKeys []channel.Key,
	configID string,
	flowName string,
	runID string,
) error {
	// Collect frame entries into a map for indexed access
	entries := make(map[channel.Key]telem.Series)
	for key, series := range frame.Entries() {
		entries[key] = series
	}

	// Get timestamps from index channel
	indexSeries := entries[indexChannelKey]
	var timestamps []telem.TimeStamp

	// Convert each data channel's series to proto channel values
	channelValues := make([][]*ingestv1.IngestWithConfigDataChannelValue, len(dataChannelKeys))
	var numSamples int
	for i, key := range dataChannelKeys {
		series, ok := entries[key]
		if !ok {
			continue
		}
		values, err := ConvertSeriesToProtoValues(series)
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
		return nil
	}

	// Send one request per timestamp row with all channel values
	for i := 0; i < numSamples; i++ {
		row := make([]*ingestv1.IngestWithConfigDataChannelValue, len(dataChannelKeys))
		for j, values := range channelValues {
			if values != nil && i < len(values) {
				row[j] = values[i]
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case requests.Inlet() <- &client.DataStreamRequest{
			IngestionConfigId: configID,
			Flow:              flowName,
			Timestamp:         timestamppb.New(timestamps[i].Time()),
			RunId:             runID,
			ChannelValues:     row,
		}:
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
