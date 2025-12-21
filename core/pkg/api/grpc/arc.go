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
	"maps"

	"github.com/google/uuid"
	"github.com/samber/lo"
	arccompiler "github.com/synnaxlabs/arc/compiler"
	arcgraph "github.com/synnaxlabs/arc/graph"
	arcir "github.com/synnaxlabs/arc/ir"
	arcmodule "github.com/synnaxlabs/arc/module"
	arcsymbol "github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	arctypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/spatial"
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
	arcRetrieveServer = fgrpc.UnaryServer[
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
)

var (
	_ fgrpc.Translator[api.ArcCreateRequest, *gapi.ArcCreateRequest]       = (*arcCreateRequestTranslator)(nil)
	_ fgrpc.Translator[api.ArcCreateResponse, *gapi.ArcCreateResponse]     = (*arcCreateResponseTranslator)(nil)
	_ fgrpc.Translator[api.ArcRetrieveRequest, *gapi.ArcRetrieveRequest]   = (*arcRetrieveRequestTranslator)(nil)
	_ fgrpc.Translator[api.ArcRetrieveResponse, *gapi.ArcRetrieveResponse] = (*arcRetrieveResponseTranslator)(nil)
	_ fgrpc.Translator[api.ArcDeleteRequest, *gapi.ArcDeleteRequest]       = (*arcDeleteRequestTranslator)(nil)
)

func (arcCreateRequestTranslator) Forward(
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

func (arcCreateRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcCreateRequest,
) (api.ArcCreateRequest, error) {
	arcs := make([]api.Arc, len(msg.GetArcs()))
	for i, arc := range msg.GetArcs() {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return api.ArcCreateRequest{}, err
		}
		arcs[i] = translated
	}
	return api.ArcCreateRequest{Arcs: arcs}, nil
}

func (arcCreateResponseTranslator) Forward(
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

func (arcCreateResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcCreateResponse,
) (api.ArcCreateResponse, error) {
	arcs := make([]api.Arc, len(msg.GetArcs()))
	for i, arc := range msg.GetArcs() {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return api.ArcCreateResponse{}, err
		}
		arcs[i] = translated
	}
	return api.ArcCreateResponse{Arcs: arcs}, nil
}

func (arcRetrieveRequestTranslator) Forward(
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
		IncludeStatus: msg.IncludeStatus,
		Compile:       msg.Compile,
	}, nil
}

func (arcRetrieveRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcRetrieveRequest,
) (api.ArcRetrieveRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.GetKeys()))
	for _, keyStr := range msg.GetKeys() {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return api.ArcRetrieveRequest{}, err
		}
		keys = append(keys, key)
	}
	return api.ArcRetrieveRequest{
		Keys:          keys,
		Names:         msg.GetNames(),
		SearchTerm:    msg.GetSearchTerm(),
		Limit:         int(msg.GetLimit()),
		Offset:        int(msg.GetOffset()),
		IncludeStatus: msg.GetIncludeStatus(),
		Compile:       msg.GetCompile(),
	}, nil
}

func (arcRetrieveResponseTranslator) Forward(
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

func (arcRetrieveResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcRetrieveResponse,
) (api.ArcRetrieveResponse, error) {
	arcs := make([]api.Arc, len(msg.GetArcs()))
	for i, arc := range msg.GetArcs() {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return api.ArcRetrieveResponse{}, err
		}
		arcs[i] = translated
	}
	return api.ArcRetrieveResponse{Arcs: arcs}, nil
}

func (arcDeleteRequestTranslator) Forward(
	_ context.Context,
	msg api.ArcDeleteRequest,
) (*gapi.ArcDeleteRequest, error) {
	keys := lo.Map(msg.Keys, func(k uuid.UUID, _ int) string { return k.String() })
	return &gapi.ArcDeleteRequest{Keys: keys}, nil
}

func (arcDeleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ArcDeleteRequest,
) (api.ArcDeleteRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.GetKeys()))
	for _, keyStr := range msg.GetKeys() {
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
	if msg.GetKey() != "" {
		parsed, err := uuid.Parse(msg.GetKey())
		if err != nil {
			return api.Arc{}, err
		}
		key = parsed
	}

	graphGo, err := translateGraphFromPB(msg.GetGraph())
	if err != nil {
		return api.Arc{}, err
	}
	textGo := translateTextFromPB(msg.GetText())
	moduleGo, err := translateModuleFromPB(msg.GetModule())
	if err != nil {
		return api.Arc{}, err
	}

	return api.Arc{
		Arc: arc.Arc{
			Key:     key,
			Name:    msg.GetName(),
			Graph:   graphGo,
			Text:    textGo,
			Module:  moduleGo,
			Deploy:  msg.GetDeploy(),
			Version: msg.GetVersion(),
		},
	}, nil
}

// translateGraphToPB converts arcgraph.Graph to *arcgraph.PBGraph.
func translateGraphToPB(g arcgraph.Graph) (*arcgraph.PBGraph, error) {
	viewportPb := &arcgraph.PBViewport{
		Position: &spatial.PBXY{X: float32(g.Viewport.Position.X), Y: float32(g.Viewport.Position.Y)},
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

	edgesPb := make([]*arcir.PBEdge, len(g.Edges))
	for i, edge := range g.Edges {
		edgesPb[i] = translateEdgeToPB(edge)
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

// translateGraphFromPB converts *arcgraph.PBGraph to arcgraph.Graph.
func translateGraphFromPB(pb *arcgraph.PBGraph) (arcgraph.Graph, error) {
	if pb == nil {
		return arcgraph.Graph{}, nil
	}

	viewport := arcgraph.Viewport{}
	if pb.GetViewport() != nil {
		if pb.GetViewport().GetPosition() != nil {
			viewport.Position = spatial.XY{X: float64(pb.GetViewport().GetPosition().GetX()), Y: float64(pb.GetViewport().GetPosition().GetY())}
		}
		viewport.Zoom = pb.GetViewport().GetZoom()
	}

	functions := make([]arcir.Function, len(pb.GetFunctions()))
	for i, fnPb := range pb.GetFunctions() {
		fn, err := translateFunctionFromPB(fnPb)
		if err != nil {
			return arcgraph.Graph{}, err
		}
		functions[i] = fn
	}

	edges := make([]arcir.Edge, len(pb.GetEdges()))
	for i, edgePb := range pb.GetEdges() {
		edges[i] = translateEdgeFromPB(edgePb)
	}

	nodes := make([]arcgraph.Node, len(pb.GetNodes()))
	for i, nodePb := range pb.GetNodes() {
		node := translateGraphNodeFromPB(nodePb)
		nodes[i] = node
	}

	return arcgraph.Graph{
		Viewport:  viewport,
		Functions: functions,
		Edges:     edges,
		Nodes:     nodes,
	}, nil
}

// translateTextToPB converts arctext.Text to *arctext.PBText.
func translateTextToPB(t arctext.Text) *arctext.PBText {
	return &arctext.PBText{Raw: t.Raw}
}

// translateTextFromPB converts *arctext.PBText to arctext.Text.
func translateTextFromPB(pb *arctext.PBText) arctext.Text {
	if pb == nil {
		return arctext.Text{}
	}
	return arctext.Text{Raw: pb.GetRaw()}
}

// translateFunctionToPB converts arcir.Function to *arcir.PBFunction.
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

// translateFunctionFromPB converts *arcir.PBFunction to arcir.Function.
func translateFunctionFromPB(pb *arcir.PBFunction) (arcir.Function, error) {
	if pb == nil {
		return arcir.Function{}, nil
	}
	config, err := translateParamsFromPB(pb.GetConfig())
	if err != nil {
		return arcir.Function{}, err
	}
	inputs, err := translateParamsFromPB(pb.GetInputs())
	if err != nil {
		return arcir.Function{}, err
	}
	outputs, err := translateParamsFromPB(pb.GetOutputs())
	if err != nil {
		return arcir.Function{}, err
	}
	channels := translateChannelsFromPB(pb.GetChannels())

	bodyRaw := ""
	if pb.GetBody() != nil {
		bodyRaw = pb.GetBody().GetRaw()
	}

	return arcir.Function{
		Key:      pb.GetKey(),
		Body:     arcir.Body{Raw: bodyRaw},
		Config:   config,
		Inputs:   inputs,
		Outputs:  outputs,
		Channels: channels,
	}, nil
}

// translateParamsToPB converts arctypes.Params to []*arctypes.PBParam.
func translateParamsToPB(p arctypes.Params) ([]*arctypes.PBParam, error) {
	params := make([]*arctypes.PBParam, len(p))
	for i, param := range p {
		typePb, err := translateTypeToPB(param.Type)
		if err != nil {
			return nil, err
		}
		pbParam := &arctypes.PBParam{
			Name: param.Name,
			Type: typePb,
		}
		// Translate Value if present
		if param.Value != nil {
			val, err := structpb.NewValue(param.Value)
			if err != nil {
				return nil, err
			}
			pbParam.Value = val
		}
		params[i] = pbParam
	}
	return params, nil
}

// translateParamsFromPB converts []*arctypes.PBParam to arctypes.Params.
func translateParamsFromPB(pbParams []*arctypes.PBParam) (arctypes.Params, error) {
	if len(pbParams) == 0 {
		return arctypes.Params{}, nil
	}
	params := make(arctypes.Params, 0, len(pbParams))
	for _, pbParam := range pbParams {
		if pbParam == nil {
			continue
		}
		typ, err := translateTypeFromPB(pbParam.GetType())
		if err != nil {
			return nil, err
		}
		param := arctypes.Param{
			Name: pbParam.GetName(),
			Type: typ,
		}
		// Extract Value if present
		if pbParam.GetValue() != nil {
			param.Value = pbParam.GetValue().AsInterface()
		}
		params = append(params, param)
	}
	return params, nil
}

// translateTypeToPB converts arctypes.Type to *arctypes.PBType.
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

// translateTypeFromPB converts *arctypes.PBType to arctypes.Type.
func translateTypeFromPB(pb *arctypes.PBType) (arctypes.Type, error) {
	if pb == nil {
		return arctypes.Type{}, nil
	}
	typ := arctypes.Type{Kind: translateTypeKindFromPB(pb.GetKind())}
	if pb.GetElem() != nil {
		elem, err := translateTypeFromPB(pb.GetElem())
		if err != nil {
			return arctypes.Type{}, err
		}
		typ.Elem = &elem
	}
	return typ, nil
}

// translateTypeKindToPB converts arctypes.TypeKind to arctypes.PBKind.
func translateTypeKindToPB(k arctypes.TypeKind) arctypes.PBKind {
	switch k {
	case arctypes.KindU8:
		return arctypes.PBKind_PB_KIND_U8
	case arctypes.KindU16:
		return arctypes.PBKind_PB_KIND_U16
	case arctypes.KindU32:
		return arctypes.PBKind_PB_KIND_U32
	case arctypes.KindU64:
		return arctypes.PBKind_PB_KIND_U64
	case arctypes.KindI8:
		return arctypes.PBKind_PB_KIND_I8
	case arctypes.KindI16:
		return arctypes.PBKind_PB_KIND_I16
	case arctypes.KindI32:
		return arctypes.PBKind_PB_KIND_I32
	case arctypes.KindI64:
		return arctypes.PBKind_PB_KIND_I64
	case arctypes.KindF32:
		return arctypes.PBKind_PB_KIND_F32
	case arctypes.KindF64:
		return arctypes.PBKind_PB_KIND_F64
	case arctypes.KindString:
		return arctypes.PBKind_PB_KIND_STRING
	case arctypes.KindTimeStamp:
		return arctypes.PBKind_PB_KIND_TIMESTAMP
	case arctypes.KindTimeSpan:
		return arctypes.PBKind_PB_KIND_TIMESPAN
	case arctypes.KindChan:
		return arctypes.PBKind_PB_KIND_CHAN
	case arctypes.KindSeries:
		return arctypes.PBKind_PB_KIND_SERIES
	default:
		return arctypes.PBKind_PB_KIND_UNSPECIFIED
	}
}

// translateTypeKindFromPB converts arctypes.PBKind to arctypes.TypeKind.
func translateTypeKindFromPB(k arctypes.PBKind) arctypes.TypeKind {
	switch k {
	case arctypes.PBKind_PB_KIND_U8:
		return arctypes.KindU8
	case arctypes.PBKind_PB_KIND_U16:
		return arctypes.KindU16
	case arctypes.PBKind_PB_KIND_U32:
		return arctypes.KindU32
	case arctypes.PBKind_PB_KIND_U64:
		return arctypes.KindU64
	case arctypes.PBKind_PB_KIND_I8:
		return arctypes.KindI8
	case arctypes.PBKind_PB_KIND_I16:
		return arctypes.KindI16
	case arctypes.PBKind_PB_KIND_I32:
		return arctypes.KindI32
	case arctypes.PBKind_PB_KIND_I64:
		return arctypes.KindI64
	case arctypes.PBKind_PB_KIND_F32:
		return arctypes.KindF32
	case arctypes.PBKind_PB_KIND_F64:
		return arctypes.KindF64
	case arctypes.PBKind_PB_KIND_STRING:
		return arctypes.KindString
	case arctypes.PBKind_PB_KIND_TIMESTAMP:
		return arctypes.KindTimeStamp
	case arctypes.PBKind_PB_KIND_TIMESPAN:
		return arctypes.KindTimeSpan
	case arctypes.PBKind_PB_KIND_CHAN:
		return arctypes.KindChan
	case arctypes.PBKind_PB_KIND_SERIES:
		return arctypes.KindSeries
	default:
		return arctypes.KindInvalid
	}
}

// translateChannelsToPB converts symbol.Channels to *arcsymbol.PBChannels.
func translateChannelsToPB(c arcsymbol.Channels) *arcsymbol.PBChannels {
	readMap := make(map[uint32]string)
	maps.Copy(readMap, c.Read)
	writeMap := make(map[uint32]string)
	maps.Copy(writeMap, c.Write)
	return &arcsymbol.PBChannels{
		Read:  readMap,
		Write: writeMap,
	}
}

// translateChannelsFromPB converts *arcsymbol.PBChannels to symbol.Channels.
func translateChannelsFromPB(pb *arcsymbol.PBChannels) arcsymbol.Channels {
	if pb == nil {
		return arcsymbol.NewChannels()
	}
	c := arcsymbol.NewChannels()
	maps.Copy(c.Read, pb.GetRead())
	maps.Copy(c.Write, pb.GetWrite())
	return c
}

// translateGraphNodeToPB converts arcgraph.Node to *arcgraph.PBNode.
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
		Position: &spatial.PBXY{X: float32(n.Position.X), Y: float32(n.Position.Y)},
	}, nil
}

// translateGraphNodeFromPB converts *arcgraph.PBNode to arcgraph.Node.
func translateGraphNodeFromPB(pb *arcgraph.PBNode) arcgraph.Node {
	if pb == nil {
		return arcgraph.Node{}
	}
	config := make(map[string]any)
	for k, v := range pb.GetConfig() {
		config[k] = v.AsInterface()
	}
	position := spatial.XY{}
	if pb.GetPosition() != nil {
		position = spatial.XY{X: float64(pb.GetPosition().GetX()), Y: float64(pb.GetPosition().GetY())}
	}
	return arcgraph.Node{
		Key:      pb.GetKey(),
		Type:     pb.GetType(),
		Config:   config,
		Position: position,
	}
}

// translateModuleToPB converts module.Module to *arcmodule.PBModule.
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

// translateModuleFromPB converts *arcmodule.PBModule to module.Module.
func translateModuleFromPB(pb *arcmodule.PBModule) (arcmodule.Module, error) {
	if pb == nil {
		return arcmodule.Module{}, nil
	}
	ir, err := translateIRFromPB(pb.GetIr())
	if err != nil {
		return arcmodule.Module{}, err
	}
	return arcmodule.Module{
		IR: ir,
		Output: arccompiler.Output{
			WASM:              pb.GetWasm(),
			OutputMemoryBases: pb.GetOutputMemoryBases(),
		},
	}, nil
}

// translateIRToPB converts ir.IR to *arcir.PBIR.
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

	sequencesPb := make([]*arcir.PBSequence, len(ir.Sequences))
	for i, seq := range ir.Sequences {
		sequencesPb[i] = translateSequenceToPB(seq)
	}

	return &arcir.PBIR{
		Functions: functionsPb,
		Nodes:     nodesPb,
		Edges:     edgesPb,
		Strata:    strataPb,
		Sequences: sequencesPb,
	}, nil
}

// translateIRFromPB converts *arcir.PBIR to ir.IR.
func translateIRFromPB(pb *arcir.PBIR) (arcir.IR, error) {
	if pb == nil {
		return arcir.IR{}, nil
	}

	functions := make(arcir.Functions, len(pb.GetFunctions()))
	for i, fnPb := range pb.GetFunctions() {
		fn, err := translateFunctionFromPB(fnPb)
		if err != nil {
			return arcir.IR{}, err
		}
		functions[i] = fn
	}

	nodes := make(arcir.Nodes, len(pb.GetNodes()))
	for i, nodePb := range pb.GetNodes() {
		node, err := translateIRNodeFromPB(nodePb)
		if err != nil {
			return arcir.IR{}, err
		}
		nodes[i] = node
	}

	edges := make(arcir.Edges, len(pb.GetEdges()))
	for i, edgePb := range pb.GetEdges() {
		edges[i] = translateEdgeFromPB(edgePb)
	}

	strata := make(arcir.Strata, len(pb.GetStrata()))
	for i, stratumPb := range pb.GetStrata() {
		strata[i] = stratumPb.GetNodes()
	}

	sequences := make(arcir.Sequences, len(pb.GetSequences()))
	for i, seqPb := range pb.GetSequences() {
		sequences[i] = translateSequenceFromPB(seqPb)
	}

	return arcir.IR{
		Functions: functions,
		Nodes:     nodes,
		Edges:     edges,
		Strata:    strata,
		Sequences: sequences,
	}, nil
}

// translateIRNodeToPB converts ir.Node to *arcir.PBNode.
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

// translateIRNodeFromPB converts *arcir.PBNode to ir.Node.
func translateIRNodeFromPB(pb *arcir.PBNode) (arcir.Node, error) {
	if pb == nil {
		return arcir.Node{}, nil
	}
	config, err := translateParamsFromPB(pb.GetConfig())
	if err != nil {
		return arcir.Node{}, err
	}
	inputs, err := translateParamsFromPB(pb.GetInputs())
	if err != nil {
		return arcir.Node{}, err
	}
	outputs, err := translateParamsFromPB(pb.GetOutputs())
	if err != nil {
		return arcir.Node{}, err
	}
	channels := translateChannelsFromPB(pb.GetChannels())

	return arcir.Node{
		Key:      pb.GetKey(),
		Type:     pb.GetType(),
		Config:   config,
		Inputs:   inputs,
		Outputs:  outputs,
		Channels: channels,
	}, nil
}

// translateEdgeToPB converts ir.Edge to *arcir.PBEdge.
func translateEdgeToPB(e arcir.Edge) *arcir.PBEdge {
	return &arcir.PBEdge{
		Source: &arcir.PBHandle{Node: e.Source.Node, Param: e.Source.Param},
		Target: &arcir.PBHandle{Node: e.Target.Node, Param: e.Target.Param},
		Kind:   arcir.PBEdgeKind(e.Kind),
	}
}

// translateEdgeFromPB converts *arcir.PBEdge to ir.Edge.
func translateEdgeFromPB(pb *arcir.PBEdge) arcir.Edge {
	source := arcir.Handle{}
	target := arcir.Handle{}
	if pb.GetSource() != nil {
		source = arcir.Handle{Node: pb.GetSource().GetNode(), Param: pb.GetSource().GetParam()}
	}
	if pb.GetTarget() != nil {
		target = arcir.Handle{Node: pb.GetTarget().GetNode(), Param: pb.GetTarget().GetParam()}
	}
	return arcir.Edge{Source: source, Target: target, Kind: arcir.EdgeKind(pb.GetKind())}
}

// translateSequenceToPB converts ir.Sequence to *arcir.PBSequence.
func translateSequenceToPB(s arcir.Sequence) *arcir.PBSequence {
	stagesPb := make([]*arcir.PBStage, len(s.Stages))
	for i, stage := range s.Stages {
		stagesPb[i] = translateStageToPB(stage)
	}
	return &arcir.PBSequence{
		Key:    s.Key,
		Stages: stagesPb,
	}
}

// translateSequenceFromPB converts *arcir.PBSequence to ir.Sequence.
func translateSequenceFromPB(pb *arcir.PBSequence) arcir.Sequence {
	if pb == nil {
		return arcir.Sequence{}
	}
	stages := make([]arcir.Stage, len(pb.GetStages()))
	for i, stagePb := range pb.GetStages() {
		stages[i] = translateStageFromPB(stagePb)
	}
	return arcir.Sequence{
		Key:    pb.GetKey(),
		Stages: stages,
	}
}

// translateStageToPB converts ir.Stage to *arcir.PBStage.
func translateStageToPB(s arcir.Stage) *arcir.PBStage {
	return &arcir.PBStage{
		Key:   s.Key,
		Nodes: s.Nodes,
	}
}

// translateStageFromPB converts *arcir.PBStage to ir.Stage.
func translateStageFromPB(pb *arcir.PBStage) arcir.Stage {
	if pb == nil {
		return arcir.Stage{}
	}
	return arcir.Stage{
		Key:   pb.GetKey(),
		Nodes: pb.GetNodes(),
	}
}

func newArc(a *api.Transport) fgrpc.CompoundBindableTransport {
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
