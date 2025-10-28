// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	arctypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/x/spatial"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	arcCreateRequestTranslator    struct{}
	arcCreateResponseTranslator   struct{}
	arcRetrieveRequestTranslator  struct{}
	arcRetrieveResponseTranslator struct{}
	arcDeleteRequestTranslator    struct{}
	arcCreateServer               = fgrpc.UnaryServer[
		api.ArcCreateRequest,
		*gapi.ArcCreateRequest,
		api.ArcCreateResponse,
		*gapi.ArcCreateResponse,
	]
	arcCreateClient = fgrpc.UnaryClient[
		api.ArcCreateRequest,
		*gapi.ArcCreateRequest,
		api.ArcCreateResponse,
		*gapi.ArcCreateResponse,
	]
	arcRetrieveServer = fgrpc.UnaryServer[
		api.ArcRetrieveRequest,
		*gapi.ArcRetrieveRequest,
		api.ArcRetrieveResponse,
		*gapi.ArcRetrieveResponse,
	]
	arcRetrieveClient = fgrpc.UnaryClient[
		api.ArcRetrieveRequest,
		*gapi.ArcRetrieveRequest,
		api.ArcRetrieveResponse,
		*gapi.ArcRetrieveResponse,
	]
	arcDeleteServer = fgrpc.UnaryServer[
		api.ArcDeleteRequest,
		*gapi.ArcDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	arcDeleteClient = fgrpc.UnaryClient[
		api.ArcDeleteRequest,
		*gapi.ArcDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[api.ArcCreateRequest, *gapi.ArcCreateRequest]       = (*arcCreateRequestTranslator)(nil)
	_ fgrpc.Translator[api.ArcCreateResponse, *gapi.ArcCreateResponse]     = (*arcCreateResponseTranslator)(nil)
	_ fgrpc.Translator[api.ArcRetrieveRequest, *gapi.ArcRetrieveRequest]   = (*arcRetrieveRequestTranslator)(nil)
	_ fgrpc.Translator[api.ArcRetrieveResponse, *gapi.ArcRetrieveResponse] = (*arcRetrieveResponseTranslator)(nil)
	_ fgrpc.Translator[api.ArcDeleteRequest, *gapi.ArcDeleteRequest]       = (*arcDeleteRequestTranslator)(nil)
)

func (t arcCreateRequestTranslator) Forward(
	_ context.Context,
	msg api.ArcCreateRequest,
) (*gapi.ArcCreateRequest, error) {
	arcs := make([]*gapi.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcForward(arc, i)
		if err != nil {
			return nil, err
		}
		arcs[i] = translated
	}
	return &gapi.ArcCreateRequest{Arcs: arcs}, nil
}

func (t arcCreateRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcCreateRequest,
) (api.ArcCreateRequest, error) {
	arcs := make([]api.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return api.ArcCreateRequest{}, err
		}
		arcs[i] = translated
	}
	return api.ArcCreateRequest{Arcs: arcs}, nil
}

func (t arcCreateResponseTranslator) Forward(
	_ context.Context,
	msg api.ArcCreateResponse,
) (*gapi.ArcCreateResponse, error) {
	arcs := make([]*gapi.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcForward(arc, i)
		if err != nil {
			return nil, err
		}
		arcs[i] = translated
	}
	return &gapi.ArcCreateResponse{Arcs: arcs}, nil
}

func (t arcCreateResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcCreateResponse,
) (api.ArcCreateResponse, error) {
	arcs := make([]api.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return api.ArcCreateResponse{}, err
		}
		arcs[i] = translated
	}
	return api.ArcCreateResponse{Arcs: arcs}, nil
}

func (t arcRetrieveRequestTranslator) Forward(
	_ context.Context,
	msg api.ArcRetrieveRequest,
) (*gapi.ArcRetrieveRequest, error) {
	keys := lo.Map(msg.Keys, func(k uuid.UUID, _ int) string { return k.String() })
	return &gapi.ArcRetrieveRequest{
		Keys:          keys,
		Names:         msg.Names,
		SearchTerm:    msg.SearchTerm,
		Limit:         int32(msg.Limit),
		Offset:        int32(msg.Offset),
		IncludeStatus: msg.IncludeStatus != nil && *msg.IncludeStatus,
	}, nil
}

func (t arcRetrieveRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcRetrieveRequest,
) (api.ArcRetrieveRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.Keys))
	for _, keyStr := range msg.Keys {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return api.ArcRetrieveRequest{}, err
		}
		keys = append(keys, key)
	}
	var includeStatus *bool
	if msg.IncludeStatus {
		includeStatus = &msg.IncludeStatus
	}
	return api.ArcRetrieveRequest{
		Keys:          keys,
		Names:         msg.Names,
		SearchTerm:    msg.SearchTerm,
		Limit:         int(msg.Limit),
		Offset:        int(msg.Offset),
		IncludeStatus: includeStatus,
	}, nil
}

func (t arcRetrieveResponseTranslator) Forward(
	_ context.Context,
	msg api.ArcRetrieveResponse,
) (*gapi.ArcRetrieveResponse, error) {
	arcs := make([]*gapi.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcForward(arc, i)
		if err != nil {
			return nil, err
		}
		arcs[i] = translated
	}
	return &gapi.ArcRetrieveResponse{Arcs: arcs}, nil
}

func (t arcRetrieveResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcRetrieveResponse,
) (api.ArcRetrieveResponse, error) {
	arcs := make([]api.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return api.ArcRetrieveResponse{}, err
		}
		arcs[i] = translated
	}
	return api.ArcRetrieveResponse{Arcs: arcs}, nil
}

func (t arcDeleteRequestTranslator) Forward(
	_ context.Context,
	msg api.ArcDeleteRequest,
) (*gapi.ArcDeleteRequest, error) {
	keys := lo.Map(msg.Keys, func(k uuid.UUID, _ int) string { return k.String() })
	return &gapi.ArcDeleteRequest{Keys: keys}, nil
}

func (t arcDeleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcDeleteRequest,
) (api.ArcDeleteRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.Keys))
	for _, keyStr := range msg.Keys {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return api.ArcDeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return api.ArcDeleteRequest{Keys: keys}, nil
}

// Protobuf conversion helpers

func translateArcForward(msg api.Arc, _ int) (*gapi.Arc, error) {
	graphPb, err := translateGraphForward(msg.Graph)
	if err != nil {
		return nil, err
	}
	textPb := translateTextForward(msg.Text)
	keyStr := ""
	// Only serialize key if it's not a zero UUID
	if msg.Key != uuid.Nil {
		keyStr = msg.Key.String()
	}
	return &gapi.Arc{
		Key:     keyStr,
		Name:    msg.Name,
		Graph:   graphPb,
		Text:    textPb,
		Deploy:  msg.Deploy,
		Version: msg.Version,
	}, nil
}

func translateArcBackward(msg *gapi.Arc, _ int) (api.Arc, error) {
	// Parse key, treating empty string as zero UUID (to be generated by server)
	var key uuid.UUID
	if msg.Key != "" {
		parsed, err := uuid.Parse(msg.Key)
		if err != nil {
			return api.Arc{}, err
		}
		key = parsed
	}

	graphGo, err := translateGraphBackward(msg.Graph)
	if err != nil {
		return api.Arc{}, err
	}
	textGo := translateTextBackward(msg.Text)

	return api.Arc{
		Arc: arc.Arc{
			Key:     key,
			Name:    msg.Name,
			Graph:   graphGo,
			Text:    textGo,
			Deploy:  msg.Deploy,
			Version: msg.Version,
		},
	}, nil
}

func translateGraphForward(g graph.Graph) (*gapi.Graph, error) {
	viewportPb := &gapi.Viewport{
		Position: &gapi.XY{X: float32(g.Viewport.Position.X), Y: float32(g.Viewport.Position.Y)},
		Zoom:     g.Viewport.Zoom,
	}

	functionsPb := make([]*gapi.Function, len(g.Functions))
	for i, fn := range g.Functions {
		fnPb, err := translateFunctionForward(fn)
		if err != nil {
			return nil, err
		}
		functionsPb[i] = fnPb
	}

	edgesPb := make([]*gapi.Edge, len(g.Edges))
	for i, edge := range g.Edges {
		edgesPb[i] = translateEdgeForward(edge)
	}

	nodesPb := make([]*gapi.GraphNode, len(g.Nodes))
	for i, node := range g.Nodes {
		nodePb, err := translateGraphNodeForward(node)
		if err != nil {
			return nil, err
		}
		nodesPb[i] = nodePb
	}

	return &gapi.Graph{
		Viewport:  viewportPb,
		Functions: functionsPb,
		Edges:     edgesPb,
		Nodes:     nodesPb,
	}, nil
}

func translateGraphBackward(pb *gapi.Graph) (graph.Graph, error) {
	if pb == nil {
		return graph.Graph{}, nil
	}

	viewport := graph.Viewport{}
	if pb.Viewport != nil {
		if pb.Viewport.Position != nil {
			viewport.Position = spatial.XY{X: float64(pb.Viewport.Position.X), Y: float64(pb.Viewport.Position.Y)}
		}
		viewport.Zoom = pb.Viewport.Zoom
	}

	functions := make([]ir.Function, len(pb.Functions))
	for i, fnPb := range pb.Functions {
		fn, err := translateFunctionBackward(fnPb)
		if err != nil {
			return graph.Graph{}, err
		}
		functions[i] = fn
	}

	edges := make([]ir.Edge, len(pb.Edges))
	for i, edgePb := range pb.Edges {
		edges[i] = translateEdgeBackward(edgePb)
	}

	nodes := make([]graph.Node, len(pb.Nodes))
	for i, nodePb := range pb.Nodes {
		node, err := translateGraphNodeBackward(nodePb)
		if err != nil {
			return graph.Graph{}, err
		}
		nodes[i] = node
	}

	return graph.Graph{
		Viewport:  viewport,
		Functions: functions,
		Edges:     edges,
		Nodes:     nodes,
	}, nil
}

func translateTextForward(t text.Text) *gapi.Text {
	return &gapi.Text{Raw: t.Raw}
}

func translateTextBackward(pb *gapi.Text) text.Text {
	if pb == nil {
		return text.Text{}
	}
	return text.Text{Raw: pb.Raw}
}

func translateFunctionForward(fn ir.Function) (*gapi.Function, error) {
	configPb, err := translateParamsForward(fn.Config)
	if err != nil {
		return nil, err
	}
	inputsPb, err := translateParamsForward(fn.Inputs)
	if err != nil {
		return nil, err
	}
	outputsPb, err := translateParamsForward(fn.Outputs)
	if err != nil {
		return nil, err
	}
	channelsPb := translateChannelsForward(fn.Channels)

	return &gapi.Function{
		Key:      fn.Key,
		RawBody:  fn.Body.Raw,
		Config:   configPb,
		Inputs:   inputsPb,
		Outputs:  outputsPb,
		Channels: channelsPb,
	}, nil
}

func translateFunctionBackward(pb *gapi.Function) (ir.Function, error) {
	if pb == nil {
		return ir.Function{}, nil
	}
	config, err := translateParamsBackward(pb.Config)
	if err != nil {
		return ir.Function{}, err
	}
	inputs, err := translateParamsBackward(pb.Inputs)
	if err != nil {
		return ir.Function{}, err
	}
	outputs, err := translateParamsBackward(pb.Outputs)
	if err != nil {
		return ir.Function{}, err
	}
	channels := translateChannelsBackward(pb.Channels)

	return ir.Function{
		Key:      pb.Key,
		Body:     ir.Body{Raw: pb.RawBody},
		Config:   config,
		Inputs:   inputs,
		Outputs:  outputs,
		Channels: channels,
	}, nil
}

func translateParamsForward(p arctypes.Params) (*gapi.Params, error) {
	values := make(map[string]*gapi.Type)
	for key, typ := range p.Iter() {
		typePb, err := translateTypeForward(typ)
		if err != nil {
			return nil, err
		}
		values[key] = typePb
	}
	return &gapi.Params{
		Keys:   p.Keys,
		Values: values,
	}, nil
}

func translateParamsBackward(pb *gapi.Params) (arctypes.Params, error) {
	if pb == nil {
		return arctypes.Params{}, nil
	}
	p := arctypes.Params{
		Keys:   make([]string, len(pb.Keys)),
		Values: make([]arctypes.Type, len(pb.Keys)),
	}
	for i, key := range pb.Keys {
		typePb, ok := pb.Values[key]
		if !ok {
			return p, nil
		}
		typ, err := translateTypeBackward(typePb)
		if err != nil {
			return p, err
		}
		p.Keys[i] = key
		p.Values[i] = typ
	}
	return p, nil
}

func translateTypeForward(t arctypes.Type) (*gapi.Type, error) {
	typePb := &gapi.Type{Kind: translateTypeKindForward(t.Kind)}
	if t.ValueType != nil {
		elemPb, err := translateTypeForward(*t.ValueType)
		if err != nil {
			return nil, err
		}
		typePb.Elem = elemPb
	}
	return typePb, nil
}

func translateTypeBackward(pb *gapi.Type) (arctypes.Type, error) {
	if pb == nil {
		return arctypes.Type{}, nil
	}
	typ := arctypes.Type{Kind: translateTypeKindBackward(pb.Kind)}
	if pb.Elem != nil {
		elem, err := translateTypeBackward(pb.Elem)
		if err != nil {
			return arctypes.Type{}, err
		}
		typ.ValueType = &elem
	}
	return typ, nil
}

func translateTypeKindForward(k arctypes.TypeKind) gapi.TypeKind {
	switch k {
	case arctypes.KindInvalid:
		return gapi.TypeKind_TYPE_INVALID
	case arctypes.KindU8:
		return gapi.TypeKind_TYPE_U8
	case arctypes.KindU16:
		return gapi.TypeKind_TYPE_U16
	case arctypes.KindU32:
		return gapi.TypeKind_TYPE_U32
	case arctypes.KindU64:
		return gapi.TypeKind_TYPE_U64
	case arctypes.KindI8:
		return gapi.TypeKind_TYPE_I8
	case arctypes.KindI16:
		return gapi.TypeKind_TYPE_I16
	case arctypes.KindI32:
		return gapi.TypeKind_TYPE_I32
	case arctypes.KindI64:
		return gapi.TypeKind_TYPE_I64
	case arctypes.KindF32:
		return gapi.TypeKind_TYPE_F32
	case arctypes.KindF64:
		return gapi.TypeKind_TYPE_F64
	case arctypes.KindString:
		return gapi.TypeKind_TYPE_STRING
	case arctypes.KindTimeStamp:
		return gapi.TypeKind_TYPE_TIMESTAMP
	case arctypes.KindTimeSpan:
		return gapi.TypeKind_TYPE_TIMESPAN
	case arctypes.KindChan:
		return gapi.TypeKind_TYPE_CHAN
	case arctypes.KindSeries:
		return gapi.TypeKind_TYPE_SERIES
	default:
		return gapi.TypeKind_TYPE_INVALID
	}
}

func translateTypeKindBackward(k gapi.TypeKind) arctypes.TypeKind {
	switch k {
	case gapi.TypeKind_TYPE_INVALID:
		return arctypes.KindInvalid
	case gapi.TypeKind_TYPE_U8:
		return arctypes.KindU8
	case gapi.TypeKind_TYPE_U16:
		return arctypes.KindU16
	case gapi.TypeKind_TYPE_U32:
		return arctypes.KindU32
	case gapi.TypeKind_TYPE_U64:
		return arctypes.KindU64
	case gapi.TypeKind_TYPE_I8:
		return arctypes.KindI8
	case gapi.TypeKind_TYPE_I16:
		return arctypes.KindI16
	case gapi.TypeKind_TYPE_I32:
		return arctypes.KindI32
	case gapi.TypeKind_TYPE_I64:
		return arctypes.KindI64
	case gapi.TypeKind_TYPE_F32:
		return arctypes.KindF32
	case gapi.TypeKind_TYPE_F64:
		return arctypes.KindF64
	case gapi.TypeKind_TYPE_STRING:
		return arctypes.KindString
	case gapi.TypeKind_TYPE_TIMESTAMP:
		return arctypes.KindTimeStamp
	case gapi.TypeKind_TYPE_TIMESPAN:
		return arctypes.KindTimeSpan
	case gapi.TypeKind_TYPE_CHAN:
		return arctypes.KindChan
	case gapi.TypeKind_TYPE_SERIES:
		return arctypes.KindSeries
	default:
		return arctypes.KindInvalid
	}
}

func translateChannelsForward(c symbol.Channels) *gapi.Channels {
	readMap := make(map[uint32]string)
	for k, v := range c.Read {
		readMap[k] = v
	}
	writeMap := make(map[uint32]string)
	for k, v := range c.Write {
		writeMap[k] = v
	}
	return &gapi.Channels{
		Read:  readMap,
		Write: writeMap,
	}
}

func translateChannelsBackward(pb *gapi.Channels) symbol.Channels {
	if pb == nil {
		return symbol.NewChannels()
	}
	c := symbol.NewChannels()
	for k, v := range pb.Read {
		c.Read[k] = v
	}
	for k, v := range pb.Write {
		c.Write[k] = v
	}
	return c
}

func translateEdgeForward(e ir.Edge) *gapi.Edge {
	return &gapi.Edge{
		Source: &gapi.Handle{Node: e.Source.Node, Param: e.Source.Param},
		Target: &gapi.Handle{Node: e.Target.Node, Param: e.Target.Param},
	}
}

func translateEdgeBackward(pb *gapi.Edge) ir.Edge {
	if pb == nil {
		return ir.Edge{}
	}
	edge := ir.Edge{}
	if pb.Source != nil {
		edge.Source = ir.Handle{Node: pb.Source.Node, Param: pb.Source.Param}
	}
	if pb.Target != nil {
		edge.Target = ir.Handle{Node: pb.Target.Node, Param: pb.Target.Param}
	}
	return edge
}

func translateGraphNodeForward(n graph.Node) (*gapi.GraphNode, error) {
	configMap := make(map[string]*structpb.Value)
	for k, v := range n.Config {
		val, err := structpb.NewValue(v)
		if err != nil {
			return nil, err
		}
		configMap[k] = val
	}
	return &gapi.GraphNode{
		Key:      n.Key,
		Type:     n.Type,
		Config:   configMap,
		Position: &gapi.XY{X: float32(n.Position.X), Y: float32(n.Position.Y)},
	}, nil
}

func translateGraphNodeBackward(pb *gapi.GraphNode) (graph.Node, error) {
	if pb == nil {
		return graph.Node{}, nil
	}
	config := make(map[string]any)
	for k, v := range pb.Config {
		config[k] = v.AsInterface()
	}
	position := spatial.XY{}
	if pb.Position != nil {
		position = spatial.XY{X: float64(pb.Position.X), Y: float64(pb.Position.Y)}
	}
	return graph.Node{
		Key:      pb.Key,
		Type:     pb.Type,
		Config:   config,
		Position: position,
	}, nil
}

func newArc(a *api.Transport) []fgrpc.BindableTransport {
	c := &arcCreateServer{
		RequestTranslator:  arcCreateRequestTranslator{},
		ResponseTranslator: arcCreateResponseTranslator{},
		ServiceDesc:        &gapi.ArcCreateService_ServiceDesc,
	}
	r := &arcRetrieveServer{
		RequestTranslator:  arcRetrieveRequestTranslator{},
		ResponseTranslator: arcRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.ArcRetrieveService_ServiceDesc,
	}
	d := &arcDeleteServer{
		RequestTranslator:  arcDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.ArcDeleteService_ServiceDesc,
	}
	a.ArcCreate = c
	a.ArcRetrieve = r
	a.ArcDelete = d
	return []fgrpc.BindableTransport{c, r, d}
}

func NewArcCreateClient(
	pool *fgrpc.Pool,
) freighter.UnaryClient[api.ArcCreateRequest, api.ArcCreateResponse] {
	return &arcCreateClient{
		RequestTranslator:  arcCreateRequestTranslator{},
		ResponseTranslator: arcCreateResponseTranslator{},
		Pool:               pool,
		ServiceDesc:        &gapi.ArcCreateService_ServiceDesc,
		Exec: func(ctx context.Context, connInterface grpc.ClientConnInterface, request *gapi.ArcCreateRequest) (*gapi.ArcCreateResponse, error) {
			return gapi.NewArcCreateServiceClient(connInterface).Exec(ctx, request)
		},
	}
}

func NewArcRetrieveClient(
	pool *fgrpc.Pool,
) freighter.UnaryClient[api.ArcRetrieveRequest, api.ArcRetrieveResponse] {
	return &arcRetrieveClient{
		RequestTranslator:  arcRetrieveRequestTranslator{},
		ResponseTranslator: arcRetrieveResponseTranslator{},
		Pool:               pool,
		ServiceDesc:        &gapi.ArcRetrieveService_ServiceDesc,
		Exec: func(ctx context.Context, connInterface grpc.ClientConnInterface, request *gapi.ArcRetrieveRequest) (*gapi.ArcRetrieveResponse, error) {
			return gapi.NewArcRetrieveServiceClient(connInterface).Exec(ctx, request)
		},
	}
}

func NewArcDeleteClient(
	pool *fgrpc.Pool,
) freighter.UnaryClient[api.ArcDeleteRequest, types.Nil] {
	return &arcDeleteClient{
		RequestTranslator:  arcDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		Pool:               pool,
		ServiceDesc:        &gapi.ArcDeleteService_ServiceDesc,
		Exec: func(ctx context.Context, connInterface grpc.ClientConnInterface, request *gapi.ArcDeleteRequest) (*emptypb.Empty, error) {
			return gapi.NewArcDeleteServiceClient(connInterface).Exec(ctx, request)
		},
	}
}
