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
	arccompiler "github.com/synnaxlabs/arc/compiler"
	arcgraph "github.com/synnaxlabs/arc/graph"
	arcir "github.com/synnaxlabs/arc/ir"
	arcmodule "github.com/synnaxlabs/arc/module"
	arcsymbol "github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	arctypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
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
		Compile:       msg.Compile,
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
		Compile:       msg.Compile,
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
	graphPb, err := translateGraphToPB(msg.Graph)
	if err != nil {
		return nil, err
	}
	textPb := translateTextToPB(msg.Text)
	keyStr := ""
	// Only serialize key if it's not a zero UUID
	if msg.Key != uuid.Nil {
		keyStr = msg.Key.String()
	}
	modulePb, err := translateModuleToPB(msg.Module)
	if err != nil {
		return nil, err
	}
	return &gapi.Arc{
		Key:     keyStr,
		Name:    msg.Name,
		Graph:   graphPb,
		Text:    textPb,
		Module:  modulePb,
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

	graphGo, err := translateGraphFromPB(msg.Graph)
	if err != nil {
		return api.Arc{}, err
	}
	textGo := translateTextFromPB(msg.Text)
	moduleGo, err := translateModuleFromPB(msg.Module)
	if err != nil {
		return api.Arc{}, err
	}

	return api.Arc{
		Arc: arc.Arc{
			Key:     key,
			Name:    msg.Name,
			Graph:   graphGo,
			Text:    textGo,
			Module:  moduleGo,
			Deploy:  msg.Deploy,
			Version: msg.Version,
		},
	}, nil
}

// translateGraphToPB converts arcgraph.Graph to *arcgraph.PBGraph
func translateGraphToPB(g arcgraph.Graph) (*arcgraph.PBGraph, error) {
	viewportPb := &arcgraph.PBViewport{
		Position: &arcgraph.XY{X: float32(g.Viewport.Position.X), Y: float32(g.Viewport.Position.Y)},
		Zoom:     g.Viewport.Zoom,
	}

	functionsPb := make([]*arcir.PBFunction, len(g.Functions))
	for i, fn := range g.Functions {
		fnPb, err := translateFunctionToPB(fn)
		if err != nil {
			return nil, err
		}
		functionsPb[i] = fnPb
	}

	// Note: The proto has `repeated ir.PBNode edges = 3;` which seems wrong,
	// but we'll work with what's generated. It should be PBEdge.
	edgesPb := make([]*arcir.PBNode, len(g.Edges))
	for i, edge := range g.Edges {
		// Since proto expects PBNode but we have edges, we'll leave this as a type mismatch
		// that needs to be fixed in the proto. For now, skip edges.
		_ = edge
		edgesPb[i] = nil
	}

	nodesPb := make([]*arcgraph.PBNode, len(g.Nodes))
	for i, node := range g.Nodes {
		nodePb, err := translateGraphNodeToPB(node)
		if err != nil {
			return nil, err
		}
		nodesPb[i] = nodePb
	}

	return &arcgraph.PBGraph{
		Viewport:  viewportPb,
		Functions: functionsPb,
		Edges:     edgesPb,
		Nodes:     nodesPb,
	}, nil
}

// translateGraphFromPB converts *arcgraph.PBGraph to arcgraph.Graph
func translateGraphFromPB(pb *arcgraph.PBGraph) (arcgraph.Graph, error) {
	if pb == nil {
		return arcgraph.Graph{}, nil
	}

	viewport := arcgraph.Viewport{}
	if pb.Viewport != nil {
		if pb.Viewport.Position != nil {
			viewport.Position = spatial.XY{X: float64(pb.Viewport.Position.X), Y: float64(pb.Viewport.Position.Y)}
		}
		viewport.Zoom = pb.Viewport.Zoom
	}

	functions := make([]arcir.Function, len(pb.Functions))
	for i, fnPb := range pb.Functions {
		fn, err := translateFunctionFromPB(fnPb)
		if err != nil {
			return arcgraph.Graph{}, err
		}
		functions[i] = fn
	}

	// Skip edges due to proto type mismatch (PBNode instead of PBEdge)
	edges := make([]arcir.Edge, 0)

	nodes := make([]arcgraph.Node, len(pb.Nodes))
	for i, nodePb := range pb.Nodes {
		node, err := translateGraphNodeFromPB(nodePb)
		if err != nil {
			return arcgraph.Graph{}, err
		}
		nodes[i] = node
	}

	return arcgraph.Graph{
		Viewport:  viewport,
		Functions: functions,
		Edges:     edges,
		Nodes:     nodes,
	}, nil
}

// translateTextToPB converts arctext.Text to *arctext.PBText
func translateTextToPB(t arctext.Text) *arctext.PBText {
	return &arctext.PBText{Raw: t.Raw}
}

// translateTextFromPB converts *arctext.PBText to arctext.Text
func translateTextFromPB(pb *arctext.PBText) arctext.Text {
	if pb == nil {
		return arctext.Text{}
	}
	return arctext.Text{Raw: pb.Raw}
}

// translateFunctionToPB converts arcir.Function to *arcir.PBFunction
func translateFunctionToPB(fn arcir.Function) (*arcir.PBFunction, error) {
	configPb, err := translateParamsToPB(fn.Config)
	if err != nil {
		return nil, err
	}
	inputsPb, err := translateParamsToPB(fn.Inputs)
	if err != nil {
		return nil, err
	}
	outputsPb, err := translateParamsToPB(fn.Outputs)
	if err != nil {
		return nil, err
	}
	channelsPb := translateChannelsToPB(fn.Channels)

	return &arcir.PBFunction{
		Key:      fn.Key,
		Body:     &arcir.PBBody{Raw: fn.Body.Raw},
		Config:   configPb,
		Inputs:   inputsPb,
		Outputs:  outputsPb,
		Channels: channelsPb,
	}, nil
}

// translateFunctionFromPB converts *arcir.PBFunction to arcir.Function
func translateFunctionFromPB(pb *arcir.PBFunction) (arcir.Function, error) {
	if pb == nil {
		return arcir.Function{}, nil
	}
	config, err := translateParamsFromPB(pb.Config)
	if err != nil {
		return arcir.Function{}, err
	}
	inputs, err := translateParamsFromPB(pb.Inputs)
	if err != nil {
		return arcir.Function{}, err
	}
	outputs, err := translateParamsFromPB(pb.Outputs)
	if err != nil {
		return arcir.Function{}, err
	}
	channels := translateChannelsFromPB(pb.Channels)

	bodyRaw := ""
	if pb.Body != nil {
		bodyRaw = pb.Body.Raw
	}

	return arcir.Function{
		Key:      pb.Key,
		Body:     arcir.Body{Raw: bodyRaw},
		Config:   config,
		Inputs:   inputs,
		Outputs:  outputs,
		Channels: channels,
	}, nil
}

// translateParamsToPB converts arctypes.Params to []*arctypes.PBParam
func translateParamsToPB(p arctypes.Params) ([]*arctypes.PBParam, error) {
	params := make([]*arctypes.PBParam, len(p))
	for i, param := range p {
		typePb, err := translateTypeToPB(param.Type)
		if err != nil {
			return nil, err
		}
		params[i] = &arctypes.PBParam{
			Name: param.Name,
			Type: typePb,
		}
	}
	return params, nil
}

// translateParamsFromPB converts []*arctypes.PBParam to arctypes.Params
func translateParamsFromPB(pbParams []*arctypes.PBParam) (arctypes.Params, error) {
	if len(pbParams) == 0 {
		return arctypes.Params{}, nil
	}
	params := make(arctypes.Params, 0, len(pbParams))
	for _, pbParam := range pbParams {
		if pbParam == nil {
			continue
		}
		typ, err := translateTypeFromPB(pbParam.Type)
		if err != nil {
			return nil, err
		}
		params = append(params, arctypes.Param{
			Name: pbParam.Name,
			Type: typ,
		})
	}
	return params, nil
}

// translateTypeToPB converts arctypes.Type to *arctypes.PBType
func translateTypeToPB(t arctypes.Type) (*arctypes.PBType, error) {
	typePb := &arctypes.PBType{Kind: translateTypeKindToPB(t.Kind)}
	if t.Elem != nil {
		elemPb, err := translateTypeToPB(*t.Elem)
		if err != nil {
			return nil, err
		}
		typePb.Elem = elemPb
	}
	return typePb, nil
}

// translateTypeFromPB converts *arctypes.PBType to arctypes.Type
func translateTypeFromPB(pb *arctypes.PBType) (arctypes.Type, error) {
	if pb == nil {
		return arctypes.Type{}, nil
	}
	typ := arctypes.Type{Kind: translateTypeKindFromPB(pb.Kind)}
	if pb.Elem != nil {
		elem, err := translateTypeFromPB(pb.Elem)
		if err != nil {
			return arctypes.Type{}, err
		}
		typ.Elem = &elem
	}
	return typ, nil
}

// translateTypeKindToPB converts arctypes.TypeKind to arctypes.PBKind
func translateTypeKindToPB(k arctypes.TypeKind) arctypes.PBKind {
	switch k {
	case arctypes.KindInvalid:
		return arctypes.PBKind_INVALID
	case arctypes.KindU8:
		return arctypes.PBKind_U8
	case arctypes.KindU16:
		return arctypes.PBKind_U16
	case arctypes.KindU32:
		return arctypes.PBKind_U32
	case arctypes.KindU64:
		return arctypes.PBKind_U64
	case arctypes.KindI8:
		return arctypes.PBKind_I8
	case arctypes.KindI16:
		return arctypes.PBKind_I16
	case arctypes.KindI32:
		return arctypes.PBKind_I32
	case arctypes.KindI64:
		return arctypes.PBKind_I64
	case arctypes.KindF32:
		return arctypes.PBKind_F32
	case arctypes.KindF64:
		return arctypes.PBKind_F64
	case arctypes.KindString:
		return arctypes.PBKind_STRING
	case arctypes.KindTimeStamp:
		return arctypes.PBKind_TIMESTAMP
	case arctypes.KindTimeSpan:
		return arctypes.PBKind_TIMESPAN
	case arctypes.KindChan:
		return arctypes.PBKind_CHAN
	case arctypes.KindSeries:
		return arctypes.PBKind_SERIES
	default:
		return arctypes.PBKind_INVALID
	}
}

// translateTypeKindFromPB converts arctypes.PBKind to arctypes.TypeKind
func translateTypeKindFromPB(k arctypes.PBKind) arctypes.TypeKind {
	switch k {
	case arctypes.PBKind_INVALID:
		return arctypes.KindInvalid
	case arctypes.PBKind_U8:
		return arctypes.KindU8
	case arctypes.PBKind_U16:
		return arctypes.KindU16
	case arctypes.PBKind_U32:
		return arctypes.KindU32
	case arctypes.PBKind_U64:
		return arctypes.KindU64
	case arctypes.PBKind_I8:
		return arctypes.KindI8
	case arctypes.PBKind_I16:
		return arctypes.KindI16
	case arctypes.PBKind_I32:
		return arctypes.KindI32
	case arctypes.PBKind_I64:
		return arctypes.KindI64
	case arctypes.PBKind_F32:
		return arctypes.KindF32
	case arctypes.PBKind_F64:
		return arctypes.KindF64
	case arctypes.PBKind_STRING:
		return arctypes.KindString
	case arctypes.PBKind_TIMESTAMP:
		return arctypes.KindTimeStamp
	case arctypes.PBKind_TIMESPAN:
		return arctypes.KindTimeSpan
	case arctypes.PBKind_CHAN:
		return arctypes.KindChan
	case arctypes.PBKind_SERIES:
		return arctypes.KindSeries
	default:
		return arctypes.KindInvalid
	}
}

// translateChannelsToPB converts symbol.Channels to *arcsymbol.PBChannels
func translateChannelsToPB(c arcsymbol.Channels) *arcsymbol.PBChannels {
	readMap := make(map[uint32]string)
	for k, v := range c.Read {
		readMap[k] = v
	}
	writeMap := make(map[uint32]string)
	for k, v := range c.Write {
		writeMap[k] = v
	}
	return &arcsymbol.PBChannels{
		Read:  readMap,
		Write: writeMap,
	}
}

// translateChannelsFromPB converts *arcsymbol.PBChannels to symbol.Channels
func translateChannelsFromPB(pb *arcsymbol.PBChannels) arcsymbol.Channels {
	if pb == nil {
		return arcsymbol.NewChannels()
	}
	c := arcsymbol.NewChannels()
	for k, v := range pb.Read {
		c.Read[k] = v
	}
	for k, v := range pb.Write {
		c.Write[k] = v
	}
	return c
}

// translateGraphNodeToPB converts arcgraph.Node to *arcgraph.PBNode
func translateGraphNodeToPB(n arcgraph.Node) (*arcgraph.PBNode, error) {
	configMap := make(map[string]*structpb.Value)
	for k, v := range n.Config {
		val, err := structpb.NewValue(v)
		if err != nil {
			return nil, err
		}
		configMap[k] = val
	}
	return &arcgraph.PBNode{
		Key:      n.Key,
		Type:     n.Type,
		Config:   configMap,
		Position: &arcgraph.XY{X: float32(n.Position.X), Y: float32(n.Position.Y)},
	}, nil
}

// translateGraphNodeFromPB converts *arcgraph.PBNode to arcgraph.Node
func translateGraphNodeFromPB(pb *arcgraph.PBNode) (arcgraph.Node, error) {
	if pb == nil {
		return arcgraph.Node{}, nil
	}
	config := make(map[string]any)
	for k, v := range pb.Config {
		config[k] = v.AsInterface()
	}
	position := spatial.XY{}
	if pb.Position != nil {
		position = spatial.XY{X: float64(pb.Position.X), Y: float64(pb.Position.Y)}
	}
	return arcgraph.Node{
		Key:      pb.Key,
		Type:     pb.Type,
		Config:   config,
		Position: position,
	}, nil
}

// translateModuleToPB converts module.Module to *arcmodule.PBModule
func translateModuleToPB(m arcmodule.Module) (*arcmodule.PBModule, error) {
	if m.IsZero() {
		return nil, nil
	}
	irPb, err := translateIRToPB(m.IR)
	if err != nil {
		return nil, err
	}
	return &arcmodule.PBModule{
		Ir:                irPb,
		Wasm:              m.WASM,
		OutputMemoryBases: m.OutputMemoryBases,
	}, nil
}

// translateModuleFromPB converts *arcmodule.PBModule to module.Module
func translateModuleFromPB(pb *arcmodule.PBModule) (arcmodule.Module, error) {
	if pb == nil {
		return arcmodule.Module{}, nil
	}
	ir, err := translateIRFromPB(pb.Ir)
	if err != nil {
		return arcmodule.Module{}, err
	}
	return arcmodule.Module{
		IR: ir,
		Output: arccompiler.Output{
			WASM:              pb.Wasm,
			OutputMemoryBases: pb.OutputMemoryBases,
		},
	}, nil
}

// translateIRToPB converts ir.IR to *arcir.PBIR
func translateIRToPB(ir arcir.IR) (*arcir.PBIR, error) {
	functionsPb := make([]*arcir.PBFunction, len(ir.Functions))
	for i, fn := range ir.Functions {
		fnPb, err := translateFunctionToPB(fn)
		if err != nil {
			return nil, err
		}
		functionsPb[i] = fnPb
	}

	nodesPb := make([]*arcir.PBNode, len(ir.Nodes))
	for i, node := range ir.Nodes {
		nodePb, err := translateIRNodeToPB(node)
		if err != nil {
			return nil, err
		}
		nodesPb[i] = nodePb
	}

	edgesPb := make([]*arcir.PBEdge, len(ir.Edges))
	for i, edge := range ir.Edges {
		edgesPb[i] = translateEdgeToPB(edge)
	}

	strataPb := make([]*arcir.PBStratum, len(ir.Strata))
	for i, stratum := range ir.Strata {
		strataPb[i] = &arcir.PBStratum{Nodes: stratum}
	}

	return &arcir.PBIR{
		Functions: functionsPb,
		Nodes:     nodesPb,
		Edges:     edgesPb,
		Strata:    strataPb,
	}, nil
}

// translateIRFromPB converts *arcir.PBIR to ir.IR
func translateIRFromPB(pb *arcir.PBIR) (arcir.IR, error) {
	if pb == nil {
		return arcir.IR{}, nil
	}

	functions := make(arcir.Functions, len(pb.Functions))
	for i, fnPb := range pb.Functions {
		fn, err := translateFunctionFromPB(fnPb)
		if err != nil {
			return arcir.IR{}, err
		}
		functions[i] = fn
	}

	nodes := make(arcir.Nodes, len(pb.Nodes))
	for i, nodePb := range pb.Nodes {
		node, err := translateIRNodeFromPB(nodePb)
		if err != nil {
			return arcir.IR{}, err
		}
		nodes[i] = node
	}

	edges := make(arcir.Edges, len(pb.Edges))
	for i, edgePb := range pb.Edges {
		edges[i] = translateEdgeFromPB(edgePb)
	}

	strata := make(arcir.Strata, len(pb.Strata))
	for i, stratumPb := range pb.Strata {
		strata[i] = stratumPb.Nodes
	}

	return arcir.IR{
		Functions: functions,
		Nodes:     nodes,
		Edges:     edges,
		Strata:    strata,
	}, nil
}

// translateIRNodeToPB converts ir.Node to *arcir.PBNode
func translateIRNodeToPB(n arcir.Node) (*arcir.PBNode, error) {
	configPb, err := translateParamsToPB(n.Config)
	if err != nil {
		return nil, err
	}
	inputsPb, err := translateParamsToPB(n.Inputs)
	if err != nil {
		return nil, err
	}
	outputsPb, err := translateParamsToPB(n.Outputs)
	if err != nil {
		return nil, err
	}
	channelsPb := translateChannelsToPB(n.Channels)

	return &arcir.PBNode{
		Key:      n.Key,
		Type:     n.Type,
		Config:   configPb,
		Inputs:   inputsPb,
		Outputs:  outputsPb,
		Channels: channelsPb,
	}, nil
}

// translateIRNodeFromPB converts *arcir.PBNode to ir.Node
func translateIRNodeFromPB(pb *arcir.PBNode) (arcir.Node, error) {
	if pb == nil {
		return arcir.Node{}, nil
	}
	config, err := translateParamsFromPB(pb.Config)
	if err != nil {
		return arcir.Node{}, err
	}
	inputs, err := translateParamsFromPB(pb.Inputs)
	if err != nil {
		return arcir.Node{}, err
	}
	outputs, err := translateParamsFromPB(pb.Outputs)
	if err != nil {
		return arcir.Node{}, err
	}
	channels := translateChannelsFromPB(pb.Channels)

	return arcir.Node{
		Key:      pb.Key,
		Type:     pb.Type,
		Config:   config,
		Inputs:   inputs,
		Outputs:  outputs,
		Channels: channels,
	}, nil
}

// translateEdgeToPB converts ir.Edge to *arcir.PBEdge
func translateEdgeToPB(e arcir.Edge) *arcir.PBEdge {
	return &arcir.PBEdge{
		Source: &arcir.PBHandle{Node: e.Source.Node, Param: e.Source.Param},
		Target: &arcir.PBHandle{Node: e.Target.Node, Param: e.Target.Param},
	}
}

// translateEdgeFromPB converts *arcir.PBEdge to ir.Edge
func translateEdgeFromPB(pb *arcir.PBEdge) arcir.Edge {
	source := arcir.Handle{}
	target := arcir.Handle{}
	if pb.Source != nil {
		source = arcir.Handle{Node: pb.Source.Node, Param: pb.Source.Param}
	}
	if pb.Target != nil {
		target = arcir.Handle{Node: pb.Target.Node, Param: pb.Target.Param}
	}
	return arcir.Edge{Source: source, Target: target}
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
