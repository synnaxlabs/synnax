// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/spatial"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer               = fgrpc.UnaryServer[
		apiarc.CreateRequest,
		*CreateRequest,
		apiarc.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apiarc.RetrieveRequest,
		*RetrieveRequest,
		apiarc.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apiarc.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apiarc.CreateRequest, *CreateRequest]       = (*createRequestTranslator)(nil)
	_ fgrpc.Translator[apiarc.CreateResponse, *CreateResponse]     = (*createResponseTranslator)(nil)
	_ fgrpc.Translator[apiarc.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apiarc.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apiarc.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func (t createRequestTranslator) Forward(
	_ context.Context,
	msg apiarc.CreateRequest,
) (*CreateRequest, error) {
	arcs := make([]*Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcForward(arc, i)
		if err != nil {
			return nil, err
		}
		arcs[i] = translated
	}
	return &CreateRequest{Arcs: arcs}, nil
}

func (t createRequestTranslator) Backward(
	_ context.Context,
	msg *CreateRequest,
) (apiarc.CreateRequest, error) {
	arcs := make([]apiarc.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return apiarc.CreateRequest{}, err
		}
		arcs[i] = translated
	}
	return apiarc.CreateRequest{Arcs: arcs}, nil
}

func (t createResponseTranslator) Forward(
	_ context.Context,
	msg apiarc.CreateResponse,
) (*CreateResponse, error) {
	arcs := make([]*Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcForward(arc, i)
		if err != nil {
			return nil, err
		}
		arcs[i] = translated
	}
	return &CreateResponse{Arcs: arcs}, nil
}

func (t createResponseTranslator) Backward(
	_ context.Context,
	msg *CreateResponse,
) (apiarc.CreateResponse, error) {
	arcs := make([]apiarc.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return apiarc.CreateResponse{}, err
		}
		arcs[i] = translated
	}
	return apiarc.CreateResponse{Arcs: arcs}, nil
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	msg apiarc.RetrieveRequest,
) (*RetrieveRequest, error) {
	keys := lo.Map(msg.Keys, func(k uuid.UUID, _ int) string { return k.String() })
	return &RetrieveRequest{
		Keys:          keys,
		Names:         msg.Names,
		SearchTerm:    msg.SearchTerm,
		Limit:         int32(msg.Limit),
		Offset:        int32(msg.Offset),
		IncludeStatus: msg.IncludeStatus,
		Compile:       msg.Compile,
	}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (apiarc.RetrieveRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.Keys))
	for _, keyStr := range msg.Keys {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return apiarc.RetrieveRequest{}, err
		}
		keys = append(keys, key)
	}
	return apiarc.RetrieveRequest{
		Keys:          keys,
		Names:         msg.Names,
		SearchTerm:    msg.SearchTerm,
		Limit:         int(msg.Limit),
		Offset:        int(msg.Offset),
		IncludeStatus: msg.IncludeStatus,
		Compile:       msg.Compile,
	}, nil
}

func (t retrieveResponseTranslator) Forward(
	_ context.Context,
	msg apiarc.RetrieveResponse,
) (*RetrieveResponse, error) {
	arcs := make([]*Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcForward(arc, i)
		if err != nil {
			return nil, err
		}
		arcs[i] = translated
	}
	return &RetrieveResponse{Arcs: arcs}, nil
}

func (t retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *RetrieveResponse,
) (apiarc.RetrieveResponse, error) {
	arcs := make([]apiarc.Arc, len(msg.Arcs))
	for i, arc := range msg.Arcs {
		translated, err := translateArcBackward(arc, i)
		if err != nil {
			return apiarc.RetrieveResponse{}, err
		}
		arcs[i] = translated
	}
	return apiarc.RetrieveResponse{Arcs: arcs}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	msg apiarc.DeleteRequest,
) (*DeleteRequest, error) {
	keys := lo.Map(msg.Keys, func(k uuid.UUID, _ int) string { return k.String() })
	return &DeleteRequest{Keys: keys}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (apiarc.DeleteRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.Keys))
	for _, keyStr := range msg.Keys {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return apiarc.DeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return apiarc.DeleteRequest{Keys: keys}, nil
}

// Protobuf conversion helpers

func translateArcForward(msg apiarc.Arc, _ int) (*Arc, error) {
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
	return &Arc{
		Key:     keyStr,
		Name:    msg.Name,
		Graph:   graphPb,
		Text:    textPb,
		Module:  modulePb,
		Deploy:  msg.Deploy,
		Version: msg.Version,
	}, nil
}

func translateArcBackward(msg *Arc, _ int) (apiarc.Arc, error) {
	// Parse key, treating empty string as zero UUID (to be generated by server)
	var key uuid.UUID
	if msg.Key != "" {
		parsed, err := uuid.Parse(msg.Key)
		if err != nil {
			return apiarc.Arc{}, err
		}
		key = parsed
	}

	graphGo, err := translateGraphFromPB(msg.Graph)
	if err != nil {
		return apiarc.Arc{}, err
	}
	textGo := translateTextFromPB(msg.Text)
	moduleGo, err := translateModuleFromPB(msg.Module)
	if err != nil {
		return apiarc.Arc{}, err
	}

	return apiarc.Arc{
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

	edges := make([]arcir.Edge, len(pb.Edges))
	for i, edgePb := range pb.Edges {
		edges[i] = translateEdgeFromPB(edgePb)
	}

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
		param := arctypes.Param{
			Name: pbParam.Name,
			Type: typ,
		}
		// Extract Value if present
		if pbParam.Value != nil {
			param.Value = pbParam.Value.AsInterface()
		}
		params = append(params, param)
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
		return arctypes.PBKind_PB_KIND_UNSPECIFIED
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

// translateTypeKindFromPB converts arctypes.PBKind to arctypes.TypeKind
func translateTypeKindFromPB(k arctypes.PBKind) arctypes.TypeKind {
	switch k {
	case arctypes.PBKind_PB_KIND_UNSPECIFIED:
		return arctypes.KindInvalid
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
		Position: &spatial.PBXY{X: float32(n.Position.X), Y: float32(n.Position.Y)},
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

	sequences := make(arcir.Sequences, len(pb.Sequences))
	for i, seqPb := range pb.Sequences {
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
		Kind:   arcir.PBEdgeKind(e.Kind),
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
	return arcir.Edge{Source: source, Target: target, Kind: arcir.EdgeKind(pb.Kind)}
}

// translateSequenceToPB converts ir.Sequence to *arcir.PBSequence
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

// translateSequenceFromPB converts *arcir.PBSequence to ir.Sequence
func translateSequenceFromPB(pb *arcir.PBSequence) arcir.Sequence {
	if pb == nil {
		return arcir.Sequence{}
	}
	stages := make([]arcir.Stage, len(pb.Stages))
	for i, stagePb := range pb.Stages {
		stages[i] = translateStageFromPB(stagePb)
	}
	return arcir.Sequence{
		Key:    pb.Key,
		Stages: stages,
	}
}

// translateStageToPB converts ir.Stage to *arcir.PBStage
func translateStageToPB(s arcir.Stage) *arcir.PBStage {
	return &arcir.PBStage{
		Key:   s.Key,
		Nodes: s.Nodes,
	}
}

// translateStageFromPB converts *arcir.PBStage to ir.Stage
func translateStageFromPB(pb *arcir.PBStage) arcir.Stage {
	if pb == nil {
		return arcir.Stage{}
	}
	return arcir.Stage{
		Key:   pb.Key,
		Nodes: pb.Nodes,
	}
}

func New(a *api.Transport) fgrpc.BindableTransport {
	c := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &ArcCreateService_ServiceDesc,
	}
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ArcRetrieveService_ServiceDesc,
	}
	d := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &ArcDeleteService_ServiceDesc,
	}
	a.ArcCreate = c
	a.ArcRetrieve = r
	a.ArcDelete = d
	return fgrpc.CompoundBindableTransport{c, r, d}
}
