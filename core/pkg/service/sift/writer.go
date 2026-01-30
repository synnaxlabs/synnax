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
	"sync/atomic"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// writerTask handles real-time streaming of channel data to Sift.
type writerTask struct {
	task       task.Task
	cfg        WriterTaskConfig
	props      DeviceProperties
	factoryCfg FactoryConfig
	pool       *ConnectionPool

	mu         sync.Mutex
	running    atomic.Bool
	cancelFn   context.CancelFunc
	wg         signal.WaitGroup
	channelMap map[channel.Key]channel.Channel
}

func newWriterTask(
	ctx context.Context,
	t task.Task,
	cfg WriterTaskConfig,
	props DeviceProperties,
	factoryCfg FactoryConfig,
	pool *ConnectionPool,
) (*writerTask, error) {
	// Retrieve channel metadata for the configured channels
	var channels []channel.Channel
	if err := factoryCfg.Channel.NewRetrieve().
		WhereKeys(cfg.Channels...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve channels")
	}

	// Build channel map for lookup
	channelMap := make(map[channel.Key]channel.Channel)
	for _, ch := range channels {
		channelMap[ch.Key()] = ch
	}

	return &writerTask{
		task:       t,
		cfg:        cfg,
		props:      props,
		factoryCfg: factoryCfg,
		pool:       pool,
		channelMap: channelMap,
	}, nil
}

// Exec handles commands for the writer task.
func (w *writerTask) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return w.start(ctx)
	case "stop":
		return w.stop()
	default:
		return errors.Newf("unknown command type: %s", cmd.Type)
	}
}

func (w *writerTask) start(ctx context.Context) error {
	if w.running.Load() {
		return errors.New("writer already running")
	}

	w.mu.Lock()
	streamCtx, cancel := context.WithCancel(ctx)
	w.cancelFn = cancel
	w.mu.Unlock()

	w.running.Store(true)
	go w.runStream(streamCtx)

	return nil
}

func (w *writerTask) stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cancelFn != nil {
		w.cancelFn()
		w.cancelFn = nil
	}
	return nil
}

func (w *writerTask) runStream(ctx context.Context) {
	defer w.running.Store(false)

	w.setStatus(xstatus.VariantInfo, "Connecting to Sift", true)

	// Get or create Sift client
	client, err := w.pool.Get(ctx, w.props)
	if err != nil {
		w.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to connect to Sift").Error(), false)
		return
	}

	// Build flow config from channels
	var channels []channel.Channel
	for _, ch := range w.channelMap {
		channels = append(channels, ch)
	}
	flows, err := w.buildFlowConfig(w.cfg.FlowName, channels)
	if err != nil {
		w.setStatus(xstatus.VariantError, err.Error(), false)
		return
	}

	// Get or create ingestion config
	ingestionConfig, err := client.GetOrCreateIngestionConfig(ctx, flows)
	if err != nil {
		w.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to create ingestion config").Error(), false)
		return
	}

	// Create run if name provided
	var runID string
	if w.cfg.RunName != "" {
		run, err := client.CreateRun(ctx, w.cfg.RunName)
		if err != nil {
			w.factoryCfg.L.Warn("failed to create run, continuing without run",
				zap.String("run_name", w.cfg.RunName),
				zap.Error(err),
			)
		} else {
			runID = run.RunId
		}
	}

	// Open ingest stream
	stream, err := client.OpenIngestStream(ctx, ingestionConfig.IngestionConfigId)
	if err != nil {
		w.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to open ingest stream").Error(), false)
		return
	}
	defer stream.Close()

	// Open Synnax streamer
	streamer, err := w.factoryCfg.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys: w.cfg.Channels,
	})
	if err != nil {
		w.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to open streamer").Error(), false)
		return
	}

	// Create channels for the streamer
	requests := confluence.NewStream[framer.StreamerRequest](1)
	responses := confluence.NewStream[framer.StreamerResponse](100)

	// Connect streamer
	streamer.InFrom(requests)
	streamer.OutTo(responses)

	// Create a signal context for the streamer
	sCtx, cancel := signal.Isolated()
	defer cancel()

	// Start the streamer
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())

	// Send the initial request to start streaming
	requests.Inlet() <- framer.StreamerConfig{Keys: w.cfg.Channels}

	w.setStatus(xstatus.VariantSuccess, "Streaming to Sift", true)

	// Process incoming frames
	var frameCount int64
	for {
		select {
		case <-ctx.Done():
			w.setStatus(xstatus.VariantWarning, "Streaming stopped", false)
			return
		case res, ok := <-responses.Outlet():
			if !ok {
				w.setStatus(xstatus.VariantWarning, "Streamer closed", false)
				return
			}

			frame := res.Frame
			if frame.Empty() {
				continue
			}

			frameCount++

			// Send each sample in the frame to Sift
			for key, series := range frame.Entries() {
				ch, ok := w.channelMap[key]
				if !ok {
					continue
				}

				values, err := w.convertSeriesToChannelValues(series)
				if err != nil {
					w.factoryCfg.L.Warn("failed to convert series",
						zap.String("channel", ch.Name),
						zap.Error(err),
					)
					continue
				}

				// Calculate sample spacing
				seriesLen := series.Len()
				timeSpan := series.TimeRange.Span()
				var sampleSpan telem.TimeSpan
				if seriesLen > 1 {
					sampleSpan = telem.TimeSpan(int64(timeSpan) / (seriesLen - 1))
				}

				for i, val := range values {
					ts := series.TimeRange.Start.Add(telem.TimeSpan(i) * sampleSpan)

					req := &ingestv1.IngestWithConfigDataStreamRequest{
						Flow:          w.cfg.FlowName,
						Timestamp:     timestamppb.New(ts.Time()),
						ChannelValues: []*ingestv1.IngestWithConfigDataChannelValue{val},
						RunId:         runID,
					}

					if err := stream.Send(req); err != nil {
						w.setStatus(xstatus.VariantError, errors.Wrap(err, "failed to send data").Error(), false)
						return
					}
				}
			}

			// Update status periodically
			if frameCount%1000 == 0 {
				w.setStatus(xstatus.VariantSuccess, "Streaming to Sift", true)
			}
		}
	}
}

func (w *writerTask) buildFlowConfig(flowName string, channels []channel.Channel) ([]FlowConfig, error) {
	channelConfigs := make([]ChannelConfig, 0, len(channels))

	for _, ch := range channels {
		if ch.IsIndex {
			continue // Skip index channels
		}

		siftType, err := MapDataType(ch.DataType)
		if err != nil {
			return nil, errors.Wrapf(err, "unsupported data type for channel %s", ch.Name)
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

func (w *writerTask) convertSeriesToChannelValues(series telem.Series) ([]*ingestv1.IngestWithConfigDataChannelValue, error) {
	values, err := ConvertSeriesToValues(series)
	if err != nil {
		return nil, err
	}

	result := make([]*ingestv1.IngestWithConfigDataChannelValue, len(values))
	for i, v := range values {
		result[i] = w.toChannelValue(v, series.DataType)
	}
	return result, nil
}

func (w *writerTask) toChannelValue(v any, dt telem.DataType) *ingestv1.IngestWithConfigDataChannelValue {
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
		// Fall back to double for unknown types
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Double{Double: 0},
		}
	}
}

func (w *writerTask) setStatus(variant xstatus.Variant, message string, running bool) {
	stat := task.Status{
		Key:     task.OntologyID(w.task.Key).String(),
		Name:    w.task.Name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{
			Task:    w.task.Key,
			Running: running,
		},
	}
	// Use a background context for status updates since the task context may be cancelled
	if err := status.NewWriter[task.StatusDetails](w.factoryCfg.Status, nil).Set(context.Background(), &stat); err != nil {
		w.factoryCfg.L.Error("failed to set status",
			zap.Uint64("task", uint64(w.task.Key)),
			zap.Error(err),
		)
	}
}

// Stop stops the writer task.
func (w *writerTask) Stop() error { return w.stop() }

// Key returns the task key.
func (w *writerTask) Key() task.Key { return w.task.Key }

var _ driver.Task = (*writerTask)(nil)
