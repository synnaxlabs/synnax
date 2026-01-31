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
	"encoding/json"
	"sync"
	"sync/atomic"

	ingest "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
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

// uploaderTask handles historical data uploads to Sift.
type uploaderTask struct {
	task       task.Task
	props      DeviceProperties
	factoryCfg FactoryConfig
	pool       *ConnectionPool
	mu         sync.Mutex
	uploading  atomic.Bool
	cancelFn   context.CancelFunc
}

func newUploaderTask(
	t task.Task,
	props DeviceProperties,
	factoryCfg FactoryConfig,
	pool *ConnectionPool,
) *uploaderTask {
	return &uploaderTask{
		task:       t,
		props:      props,
		factoryCfg: factoryCfg,
		pool:       pool,
	}
}

// Exec handles commands for the uploader task.
func (u *uploaderTask) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "upload":
		return u.startUpload(ctx, cmd)
	case "cancel":
		return u.cancel()
	default:
		return errors.Newf("unknown command type: %s", cmd.Type)
	}
}

func (u *uploaderTask) startUpload(ctx context.Context, cmd task.Command) error {
	if u.uploading.Load() {
		return errors.New("upload already in progress")
	}

	var uploadCmd UploadCommand
	if err := json.Unmarshal(cmd.Args, &uploadCmd); err != nil {
		return errors.Wrap(err, "failed to parse upload command")
	}

	u.mu.Lock()
	uploadCtx, cancel := context.WithCancel(ctx)
	u.cancelFn = cancel
	u.mu.Unlock()

	u.uploading.Store(true)
	go u.doUpload(uploadCtx, uploadCmd)

	return nil
}

func (u *uploaderTask) cancel() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.cancelFn != nil {
		u.cancelFn()
		u.cancelFn = nil
	}
	return nil
}

func (u *uploaderTask) doUpload(ctx context.Context, cmd UploadCommand) {
	defer u.uploading.Store(false)

	// Set status to running
	u.setStatus(xstatus.VariantInfo, "Upload starting", true)

	// Retrieve the range
	var rng ranger.Range
	if err := u.factoryCfg.Ranger.NewRetrieve().
		WhereKeys(cmd.RangeKey).
		Entry(&rng).
		Exec(ctx, nil); err != nil {
		u.setStatus(
			xstatus.VariantError,
			errors.Wrap(err, "failed to retrieve range").Error(),
			false,
		)
		return
	}

	// Use range time bounds or override
	bounds := rng.TimeRange
	if cmd.TimeRange != nil {
		bounds = *cmd.TimeRange
	}

	// Retrieve channel metadata
	var channels []channel.Channel
	if err := u.factoryCfg.Channel.NewRetrieve().
		WhereKeys(cmd.Channels...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		u.setStatus(
			xstatus.VariantError,
			errors.Wrap(err, "failed to retrieve channels").Error(),
			false,
		)
		return
	}

	// Build channel map for lookup
	channelMap := make(map[channel.Key]channel.Channel)
	for _, ch := range channels {
		channelMap[ch.Key()] = ch
	}

	// Get or create Sift client
	client, err := u.pool.Get(ctx, u.props)
	if err != nil {
		u.setStatus(
			xstatus.VariantError,
			errors.Wrap(err, "failed to connect to Sift").Error(),
			false,
		)
		return
	}

	// Build flow config from channels
	flows, err := u.buildFlowConfig(cmd.FlowName, channels)
	if err != nil {
		u.setStatus(xstatus.VariantError, err.Error(), false)
		return
	}

	// Get or create ingestion config
	ingestionConfig, err := client.GetOrCreateIngestionConfig(ctx, flows)
	if err != nil {
		u.setStatus(
			xstatus.VariantError,
			errors.Wrap(err, "failed to create ingestion config").Error(),
			false,
		)
		return
	}

	// Create run if name provided
	var runID string
	if cmd.RunName != "" {
		run, err := client.CreateRun(ctx, cmd.RunName)
		if err != nil {
			u.factoryCfg.L.Warn("failed to create run, continuing without run",
				zap.String("run_name", cmd.RunName),
				zap.Error(err),
			)
		} else {
			runID = run.RunId
		}
	}

	// Open ingest stream
	stream, err := client.OpenIngestStream(ctx, ingestionConfig.IngestionConfigId)
	if err != nil {
		u.setStatus(
			xstatus.VariantError,
			errors.Wrap(err, "failed to open ingest stream").Error(),
			false,
		)
		return
	}
	defer stream.Close()

	// Open iterator for the range
	iter, err := u.factoryCfg.Framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   cmd.Channels,
		Bounds: bounds,
	})
	if err != nil {
		u.setStatus(
			xstatus.VariantError,
			errors.Wrap(err, "failed to open iterator").Error(),
			false,
		)
		return
	}
	defer iter.Close()

	// Iterate and send data
	var (
		sampleCount int64
		frameCount  int64
	)

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

		// Send each sample in the frame
		for key, series := range frame.Entries() {
			ch, ok := channelMap[key]
			if !ok {
				continue
			}

			values, err := u.convertSeriesToChannelValues(series)
			if err != nil {
				u.factoryCfg.L.Warn("failed to convert series",
					zap.String("channel", ch.Name),
					zap.Error(err),
				)
				continue
			}

			// Calculate sample spacing based on time range and sample count
			seriesLen := series.Len()
			timeSpan := series.TimeRange.Span()
			var sampleSpan telem.TimeSpan
			if seriesLen > 1 {
				sampleSpan = telem.TimeSpan(int64(timeSpan) / (seriesLen - 1))
			}

			for i, val := range values {
				ts := series.TimeRange.Start.Add(telem.TimeSpan(i) * sampleSpan)

				req := &ingest.IngestWithConfigDataStreamRequest{
					Flow:          cmd.FlowName,
					Timestamp:     timestamppb.New(ts.Time()),
					ChannelValues: []*ingest.IngestWithConfigDataChannelValue{val},
					RunId:         runID,
				}

				if err := stream.Send(req); err != nil {
					u.setStatus(
						xstatus.VariantError,
						errors.Wrap(err, "failed to send data").Error(),
						false,
					)
					return
				}
				sampleCount++
			}
		}

		// Update progress periodically
		if frameCount%100 == 0 {
			u.setStatus(xstatus.VariantInfo, "Upload in progress", true)
		}
	}

	if err := iter.Error(); err != nil {
		u.setStatus(
			xstatus.VariantError,
			errors.Wrap(err, "iterator error").Error(),
			false,
		)
		return
	}

	u.setStatus(xstatus.VariantSuccess, "Upload completed", false)
}

func (u *uploaderTask) buildFlowConfig(
	flowName string,
	channels []channel.Channel,
) ([]FlowConfig, error) {
	channelConfigs := make([]ChannelConfig, 0, len(channels))

	for _, ch := range channels {
		if ch.IsIndex {
			continue // Skip index channels
		}

		siftType, err := MapDataType(ch.DataType)
		if err != nil {
			return nil,
				errors.Wrapf(err, "unsupported data type for channel %s", ch.Name)
		}

		channelConfigs = append(channelConfigs, ChannelConfig{
			Name:        ch.Name,
			Component:   "",
			Unit:        "",
			Description: "",
			DataType:    siftType,
		})
	}

	return []FlowConfig{{
		Name:     flowName,
		Channels: channelConfigs,
	}}, nil
}

func (u *uploaderTask) convertSeriesToChannelValues(
	series telem.Series,
) ([]*ingest.IngestWithConfigDataChannelValue, error) {
	values, err := ConvertSeriesToValues(series)
	if err != nil {
		return nil, err
	}

	result := make([]*ingest.IngestWithConfigDataChannelValue, len(values))
	for i, v := range values {
		result[i] = u.toChannelValue(v, series.DataType)
	}
	return result, nil
}

func (u *uploaderTask) toChannelValue(
	v any,
	dt telem.DataType,
) *ingest.IngestWithConfigDataChannelValue {
	switch dt {
	case telem.Float64T:
		return &ingest.IngestWithConfigDataChannelValue{
			Type: &ingest.IngestWithConfigDataChannelValue_Double{
				Double: v.(float64),
			},
		}
	case telem.Float32T:
		return &ingest.IngestWithConfigDataChannelValue{
			Type: &ingest.IngestWithConfigDataChannelValue_Float{Float: v.(float32)},
		}
	case telem.Int64T, telem.TimeStampT:
		return &ingest.IngestWithConfigDataChannelValue{
			Type: &ingest.IngestWithConfigDataChannelValue_Int64{Int64: v.(int64)},
		}
	case telem.Int32T, telem.Int16T, telem.Int8T:
		return &ingest.IngestWithConfigDataChannelValue{
			Type: &ingest.IngestWithConfigDataChannelValue_Int32{Int32: v.(int32)},
		}
	case telem.Uint64T:
		return &ingest.IngestWithConfigDataChannelValue{
			Type: &ingest.IngestWithConfigDataChannelValue_Uint64{Uint64: v.(uint64)},
		}
	case telem.Uint32T, telem.Uint16T, telem.Uint8T:
		return &ingest.IngestWithConfigDataChannelValue{
			Type: &ingest.IngestWithConfigDataChannelValue_Uint32{Uint32: v.(uint32)},
		}
	default:
		// Fall back to double for unknown types
		return &ingest.IngestWithConfigDataChannelValue{
			Type: &ingest.IngestWithConfigDataChannelValue_Double{Double: 0},
		}
	}
}

func (u *uploaderTask) setStatus(
	variant xstatus.Variant,
	message string,
	running bool,
) {
	stat := task.Status{
		Key:     task.OntologyID(u.task.Key).String(),
		Name:    u.task.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{
			Task:    u.task.Key,
			Running: running,
		},
	}
	// Use a background context for status updates since the task context may be
	// cancelled
	if err := status.NewWriter[task.StatusDetails](
		u.factoryCfg.Status, nil,
	).Set(context.Background(), &stat); err != nil {
		u.factoryCfg.L.Error("failed to set status",
			zap.Uint64("task", uint64(u.task.Key)),
			zap.Error(err),
		)
	}
}

func (u *uploaderTask) Stop() error { return u.cancel() }

// Ensure uploaderTask implements driver.Task
var _ driver.Task = (*uploaderTask)(nil)
