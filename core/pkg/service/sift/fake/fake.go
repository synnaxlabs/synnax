// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package fake runs an in-process Sift server for tests. It keeps what Sift keeps:
// ingestion configs, assets, and the last value at each timestamp of each channel.
package fake

import (
	"context"
	"io"
	"net"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	assetsv1 "github.com/sift-stack/sift/go/gen/sift/assets/v1"
	typev1 "github.com/sift-stack/sift/go/gen/sift/common/type/v1"
	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	ingestionconfigsv2 "github.com/sift-stack/sift/go/gen/sift/ingestion_configs/v2"
	pingv1 "github.com/sift-stack/sift/go/gen/sift/ping/v1"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Config configures a Server.
type Config struct {
	// APIKey is the bearer token every call must carry. An empty APIKey accepts every
	// call.
	APIKey string
	// Assets are the names of assets that exist before any ingestion config creates
	// one.
	Assets []string
}

// Point is one value Sift stores for a channel.
type Point struct {
	// Time is the timestamp of the value.
	Time time.Time
	// Value is the Go value of the sample: float64, float32, bool, int32, uint32,
	// int64, uint64, string, or []byte.
	Value any
}

type ingestionConfig struct {
	*ingestionconfigsv2.IngestionConfig
	asset string
	flows map[string]*ingestionconfigsv2.FlowConfig
}

type channelKey struct{ asset, channel string }

// Server is an in-process Sift server. Its methods are safe for concurrent use.
type Server struct {
	ingestv1.UnimplementedIngestServiceServer
	ingestionconfigsv2.UnimplementedIngestionConfigServiceServer
	pingv1.UnimplementedPingServiceServer
	assetsv1.UnimplementedAssetServiceServer
	cfg      Config
	grpc     *grpc.Server
	listener net.Listener
	served   chan error
	// mu guards every field below it.
	mu        sync.Mutex
	configs   []ingestionConfig
	points    map[channelKey]map[time.Time]any
	invalid   []*ingestv1.IngestWithConfigDataStreamRequest
	confirmed int
	received  int
	streams   int
	failures  []error
	paused    chan struct{}
}

var (
	_ ingestv1.IngestServiceServer                    = (*Server)(nil)
	_ ingestionconfigsv2.IngestionConfigServiceServer = (*Server)(nil)
	_ pingv1.PingServiceServer                        = (*Server)(nil)
	_ assetsv1.AssetServiceServer                     = (*Server)(nil)
)

// Open starts a Server on a free local port. The caller must Close it.
func Open(cfg Config) (*Server, error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{
		cfg:      cfg,
		listener: lis,
		points:   make(map[channelKey]map[time.Time]any),
	}
	s.grpc = grpc.NewServer(
		grpc.ChainUnaryInterceptor(s.authorizeUnary),
		grpc.ChainStreamInterceptor(s.authorizeStream),
	)
	ingestv1.RegisterIngestServiceServer(s.grpc, s)
	ingestionconfigsv2.RegisterIngestionConfigServiceServer(s.grpc, s)
	pingv1.RegisterPingServiceServer(s.grpc, s)
	assetsv1.RegisterAssetServiceServer(s.grpc, s)
	s.served = make(chan error, 1)
	go func() { s.served <- s.grpc.Serve(lis) }()
	return s, nil
}

// Address returns the host and port the Server listens on.
func (s *Server) Address() string { return s.listener.Addr().String() }

// Close stops the Server and ends every open call.
func (s *Server) Close() error {
	s.Resume()
	s.grpc.Stop()
	// Serve returns ErrServerStopped when Stop runs before it starts.
	if err := <-s.served; !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

// Points returns the values Sift stores for the channel of the asset, in time order.
func (s *Server) Points(asset, channel string) []Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.points[channelKey{asset: asset, channel: channel}]
	points := make([]Point, 0, len(stored))
	for t, v := range stored {
		points = append(points, Point{Time: t, Value: v})
	}
	slices.SortFunc(points, func(a, b Point) int { return a.Time.Compare(b.Time) })
	return points
}

// IngestionConfigs returns every ingestion config created on the Server.
func (s *Server) IngestionConfigs() []*ingestionconfigsv2.IngestionConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	configs := make([]*ingestionconfigsv2.IngestionConfig, len(s.configs))
	for i, c := range s.configs {
		configs[i] = c.IngestionConfig
	}
	return configs
}

// Invalid returns the requests that did not match their ingestion config. Sift sends
// these to its dead letter queue instead of storing them.
func (s *Server) Invalid() []*ingestv1.IngestWithConfigDataStreamRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.invalid)
}

// Confirmed returns how many ingest streams the Server has confirmed.
func (s *Server) Confirmed() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.confirmed
}

// Received returns how many ingest requests the Server has read, confirmed or not.
func (s *Server) Received() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.received
}

// Streams returns how many ingest streams clients have opened.
func (s *Server) Streams() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streams
}

// Fail makes each of the next ingest streams end with one of errs, in order. A failed
// stream stores none of its values.
func (s *Server) Fail(errs ...error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = append(s.failures, errs...)
}

// Pause stops ingest streams from reading requests until Resume, so clients back up.
func (s *Server) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused == nil {
		s.paused = make(chan struct{})
	}
}

// Resume lets ingest streams read requests again.
func (s *Server) Resume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.paused != nil {
		close(s.paused)
		s.paused = nil
	}
}

// CreateIngestionConfig implements ingestionconfigsv2.IngestionConfigServiceServer.
func (s *Server) CreateIngestionConfig(
	_ context.Context,
	req *ingestionconfigsv2.CreateIngestionConfigRequest,
) (*ingestionconfigsv2.CreateIngestionConfigResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.AssetName == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_name is required")
	}
	if req.ClientKey != "" {
		for _, c := range s.configs {
			if c.ClientKey == req.ClientKey {
				return nil, status.Errorf(
					codes.AlreadyExists, "client key %q already exists", req.ClientKey,
				)
			}
		}
	}
	cfg := ingestionConfig{
		IngestionConfig: &ingestionconfigsv2.IngestionConfig{
			IngestionConfigId: "ingestion-config-" + strconv.Itoa(len(s.configs)+1),
			AssetId:           "asset-" + req.AssetName,
			ClientKey:         req.ClientKey,
		},
		asset: req.AssetName,
		flows: make(map[string]*ingestionconfigsv2.FlowConfig, len(req.Flows)),
	}
	for _, f := range req.Flows {
		if _, ok := cfg.flows[f.Name]; ok {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"duplicate flow %q",
				f.Name,
			)
		}
		cfg.flows[f.Name] = f
	}
	s.configs = append(s.configs, cfg)
	return &ingestionconfigsv2.CreateIngestionConfigResponse{
		IngestionConfig: cfg.IngestionConfig,
	}, nil
}

var clientKeyFilter = regexp.MustCompile(`^client_key\s*==\s*"([^"]*)"$`)

// ListIngestionConfigs implements ingestionconfigsv2.IngestionConfigServiceServer. It
// understands an empty filter and the filter `client_key == "<key>"`.
func (s *Server) ListIngestionConfigs(
	_ context.Context,
	req *ingestionconfigsv2.ListIngestionConfigsRequest,
) (*ingestionconfigsv2.ListIngestionConfigsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var key *string
	if req.Filter != "" {
		m := clientKeyFilter.FindStringSubmatch(req.Filter)
		if m == nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"bad filter %q",
				req.Filter,
			)
		}
		key = &m[1]
	}
	res := &ingestionconfigsv2.ListIngestionConfigsResponse{}
	for _, c := range s.configs {
		if key == nil || c.ClientKey == *key {
			res.IngestionConfigs = append(res.IngestionConfigs, c.IngestionConfig)
		}
	}
	return res, nil
}

// Ping implements pingv1.PingServiceServer.
func (s *Server) Ping(
	context.Context,
	*pingv1.PingRequest,
) (*pingv1.PingResponse, error) {
	return &pingv1.PingResponse{Response: "pong"}, nil
}

// ListAssets implements assetsv1.AssetServiceServer. It lists the assets in
// Config.Assets and the assets of each ingestion config by name, in pages of
// PageSize.
func (s *Server) ListAssets(
	_ context.Context,
	req *assetsv1.ListAssetsRequest,
) (*assetsv1.ListAssetsResponse, error) {
	s.mu.Lock()
	names := slices.Clone(s.cfg.Assets)
	for _, c := range s.configs {
		names = append(names, c.asset)
	}
	s.mu.Unlock()
	slices.Sort(names)
	names = slices.Compact(names)
	start := 0
	if req.PageToken != "" {
		var err error
		start, err = strconv.Atoi(req.PageToken)
		if err != nil || start < 0 || start > len(names) {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"bad page token %q",
				req.PageToken,
			)
		}
	}
	size := int(req.PageSize)
	if size == 0 {
		size = 50
	}
	end := min(start+size, len(names))
	res := &assetsv1.ListAssetsResponse{}
	for _, name := range names[start:end] {
		res.Assets = append(
			res.Assets,
			&assetsv1.Asset{AssetId: "asset-" + name, Name: name},
		)
	}
	if end < len(names) {
		res.NextPageToken = strconv.Itoa(end)
	}
	return res, nil
}

// IngestWithConfigDataStream implements ingestv1.IngestServiceServer. It stores the
// values of a stream only when the client half-closes it, and then confirms it with
// an empty response.
func (s *Server) IngestWithConfigDataStream(
	stream ingestv1.IngestService_IngestWithConfigDataStreamServer,
) error {
	s.mu.Lock()
	s.streams++
	s.mu.Unlock()
	var received []*ingestv1.IngestWithConfigDataStreamRequest
	for {
		if err := s.waitResumed(stream.Context()); err != nil {
			return err
		}
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.received++
		s.mu.Unlock()
		received = append(received, req)
	}
	s.mu.Lock()
	if len(s.failures) > 0 {
		err := s.failures[0]
		s.failures = s.failures[1:]
		s.mu.Unlock()
		return err
	}
	for _, req := range received {
		s.store(req)
	}
	s.confirmed++
	s.mu.Unlock()
	return stream.SendAndClose(&ingestv1.IngestWithConfigDataStreamResponse{})
}

func (s *Server) waitResumed(ctx context.Context) error {
	s.mu.Lock()
	paused := s.paused
	s.mu.Unlock()
	if paused == nil {
		return nil
	}
	select {
	case <-paused:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// store must be called with mu held.
func (s *Server) store(req *ingestv1.IngestWithConfigDataStreamRequest) {
	idx := slices.IndexFunc(s.configs, func(c ingestionConfig) bool {
		return c.IngestionConfigId == req.IngestionConfigId
	})
	if idx < 0 {
		s.invalid = append(s.invalid, req)
		return
	}
	cfg := s.configs[idx]
	flow, ok := cfg.flows[req.Flow]
	if !ok || len(flow.Channels) != len(req.ChannelValues) || req.Timestamp == nil {
		s.invalid = append(s.invalid, req)
		return
	}
	values := make([]any, len(req.ChannelValues))
	for i, v := range req.ChannelValues {
		value, valid := decode(flow.Channels[i].DataType, v)
		if !valid {
			s.invalid = append(s.invalid, req)
			return
		}
		values[i] = value
	}
	t := req.Timestamp.AsTime()
	for i, v := range values {
		if v == nil {
			continue
		}
		key := channelKey{asset: cfg.asset, channel: flow.Channels[i].Name}
		if s.points[key] == nil {
			s.points[key] = make(map[time.Time]any)
		}
		s.points[key][t] = v
	}
}

// decode returns the Go value of v, nil for an empty value, and whether v matches dt.
func decode(
	dt typev1.ChannelDataType,
	v *ingestv1.IngestWithConfigDataChannelValue,
) (any, bool) {
	switch t := v.Type.(type) {
	case *ingestv1.IngestWithConfigDataChannelValue_Empty:
		return nil, true
	case *ingestv1.IngestWithConfigDataChannelValue_Double:
		return t.Double, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_DOUBLE
	case *ingestv1.IngestWithConfigDataChannelValue_Float:
		return t.Float, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_FLOAT
	case *ingestv1.IngestWithConfigDataChannelValue_Bool:
		return t.Bool, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_BOOL
	case *ingestv1.IngestWithConfigDataChannelValue_Int32:
		return t.Int32, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_32
	case *ingestv1.IngestWithConfigDataChannelValue_Uint32:
		return t.Uint32, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_32
	case *ingestv1.IngestWithConfigDataChannelValue_Int64:
		return t.Int64, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_INT_64
	case *ingestv1.IngestWithConfigDataChannelValue_Uint64:
		return t.Uint64, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_UINT_64
	case *ingestv1.IngestWithConfigDataChannelValue_String_:
		return t.String_, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_STRING
	case *ingestv1.IngestWithConfigDataChannelValue_Bytes:
		return t.Bytes, dt == typev1.ChannelDataType_CHANNEL_DATA_TYPE_BYTES
	default:
		return nil, false
	}
}

func (s *Server) authorize(ctx context.Context) error {
	if s.cfg.APIKey == "" {
		return nil
	}
	md, _ := metadata.FromIncomingContext(ctx)
	if slices.Contains(md.Get("authorization"), "Bearer "+s.cfg.APIKey) {
		return nil
	}
	return status.Error(codes.Unauthenticated, "invalid API key")
}

func (s *Server) authorizeUnary(
	ctx context.Context,
	req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func (s *Server) authorizeStream(
	srv any,
	ss grpc.ServerStream,
	_ *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	if err := s.authorize(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}
