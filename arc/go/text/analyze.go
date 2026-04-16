// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/authority"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// keyGenerator produces globally unique IR node keys. It maintains a running
// per-role counter so that successive channel reads, writes, or function
// invocations with the same logical name receive distinct keys.
type keyGenerator struct {
	occurrences map[string]int
}

func newKeyGenerator() *keyGenerator {
	return &keyGenerator{occurrences: make(map[string]int)}
}

func (kg *keyGenerator) generate(role, name string) string {
	base := role
	if name != "" {
		base = role + "_" + name
	}
	count := kg.occurrences[base]
	kg.occurrences[base]++
	return fmt.Sprintf("%s_%d", base, count)
}

// seqFrame tracks state for a single sequential Scope that is currently being
// analyzed. Transitions collected on the frame are copied onto the emitted
// Scope when the frame is popped.
type seqFrame struct {
	// name is the scope key of the sequence being built.
	name string
	// memberKeys holds the ordered keys of each member declared by the source
	// (stage name, nested sequence name, or synthesized step_N). Used to
	// validate transition targets and resolve `=> next`.
	memberKeys []string
	// activeIdx is the index into memberKeys currently being analyzed.
	activeIdx int
	// transitions accumulates the scope's transitions in source order.
	transitions []ir.Transition
}

// currentMember returns the key of the step being analyzed by this frame.
func (f *seqFrame) currentMember() string {
	if f == nil || f.activeIdx < 0 || f.activeIdx >= len(f.memberKeys) {
		return ""
	}
	return f.memberKeys[f.activeIdx]
}

// nextMember returns the key of the member that follows the currently active
// one, or the empty string if the current member is the last.
func (f *seqFrame) nextMember() string {
	if f == nil || f.activeIdx+1 >= len(f.memberKeys) {
		return ""
	}
	return f.memberKeys[f.activeIdx+1]
}

// shellBuilder tracks the Layer-2 execution-shell state while the analyzer
// walks flow, stage, and sequence constructs. It records transitions against
// the enclosing sequential scopes and registers pending activations that will
// be stamped onto top-level Scope members once the main loop is done.
type shellBuilder struct {
	stack       []*seqFrame
	activations map[string]ir.Handle
}

func newShellBuilder() *shellBuilder {
	return &shellBuilder{activations: map[string]ir.Handle{}}
}

// pushSeq declares a new sequential frame with the given member keys.
func (s *shellBuilder) pushSeq(name string, memberKeys []string) *seqFrame {
	frame := &seqFrame{name: name, memberKeys: memberKeys}
	s.stack = append(s.stack, frame)
	return frame
}

// popSeq removes the innermost sequence frame.
func (s *shellBuilder) popSeq() {
	if len(s.stack) > 0 {
		s.stack = s.stack[:len(s.stack)-1]
	}
}

// top returns the innermost sequence frame, or nil when no sequence is being
// analyzed (module-scope flow).
func (s *shellBuilder) top() *seqFrame {
	if len(s.stack) == 0 {
		return nil
	}
	return s.stack[len(s.stack)-1]
}

// inSequence reports whether the analyzer is currently inside a sequential
// scope.
func (s *shellBuilder) inSequence() bool { return s.top() != nil }

// addTransition appends a transition to the innermost sequence frame. Panics
// if no sequence is active; callers must check inSequence first.
func (s *shellBuilder) addTransition(t ir.Transition) {
	s.stack[len(s.stack)-1].transitions = append(
		s.stack[len(s.stack)-1].transitions, t,
	)
}

// registerActivation records that the scope named key should be activated by
// the given handle. The activation is stamped onto the emitted Scope by the
// main Analyze loop once all top-level items have been processed.
func (s *shellBuilder) registerActivation(key string, on ir.Handle) {
	s.activations[key] = on
}

// nodeResult describes an IR node produced by a flow-node analysis.
type nodeResult struct {
	node   ir.Node
	input  ir.Handle
	output ir.Handle
}

func newNodeResult(node ir.Node, inputParam, outputParam string) nodeResult {
	return nodeResult{
		node:   node,
		input:  ir.Handle{Node: node.Key, Param: inputParam},
		output: ir.Handle{Node: node.Key, Param: outputParam},
	}
}

// transitionIntent is emitted by `=> next` and `=> scope_name` targets. The
// flow-chain processor consumes the intent and, rather than emitting a
// dataflow edge, records a Transition on the enclosing sequence (for
// intra-sequence jumps and `next`) and/or registers an activation on the
// target scope (for cross-scope jumps).
type transitionIntent struct {
	// isNext is true when the intent came from the `next` token. The target
	// member is resolved against the innermost sequence frame at the time the
	// intent is consumed.
	isNext bool
	// memberKey, when non-empty and isNext is false, names a sibling member
	// in the enclosing sequence to transition to.
	memberKey string
	// activateKey, when non-empty, names a top-level scope whose activation
	// should be set to the firing handle. Combined with an exit transition
	// when the intent is consumed inside a sequence.
	activateKey string
}

// flowNodeResult is what analyzeFlowNode returns: either an actual IR node
// (the usual case) or a transition intent (for `=> next` / `=> scope_name`
// targets). Exactly one of node.node.Key or transition is non-zero.
type flowNodeResult struct {
	node       nodeResult
	transition *transitionIntent
}

func firstInputParam(inputs types.Params) string {
	if len(inputs) > 0 {
		return inputs[0].Name
	}
	return ir.DefaultInputParam
}

func firstOutputParam(outputs types.Params) string {
	if len(outputs) > 0 {
		return outputs[0].Name
	}
	return ir.DefaultOutputParam
}

func analyzeFlowNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
	shell *shellBuilder,
	isSink bool,
) (flowNodeResult, bool) {
	if id := ctx.AST.Identifier(); id != nil {
		return analyzeIdentifierByRole(acontext.Child(ctx, id), kg, shell, isSink)
	}
	if fn := ctx.AST.Function(); fn != nil {
		r, ok := analyzeFunctionNode(acontext.Child(ctx, fn), kg)
		return flowNodeResult{node: r}, ok
	}
	if expr := ctx.AST.Expression(); expr != nil {
		r, ok := analyzeExpression(acontext.Child(ctx, expr), kg)
		return flowNodeResult{node: r}, ok
	}
	if ctx.AST.NEXT() != nil {
		return analyzeNextToken(ctx, shell)
	}
	return flowNodeResult{}, true
}

func analyzeIdentifierByRole(
	ctx acontext.Context[parser.IIdentifierContext],
	kg *keyGenerator,
	shell *shellBuilder,
	isSink bool,
) (flowNodeResult, bool) {
	name := ctx.AST.IDENTIFIER().GetText()
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return flowNodeResult{}, false
	}

	switch sym.Kind {
	case symbol.KindSequence:
		intent, ok := analyzeSequenceRef(ctx, sym)
		if !ok {
			return flowNodeResult{}, false
		}
		return flowNodeResult{transition: &intent}, true
	case symbol.KindStage:
		intent := analyzeStageRef(sym, shell)
		return flowNodeResult{transition: &intent}, true
	case symbol.KindGlobalConstant:
		r, ok := buildGlobalConstantNode(name, sym, kg)
		return flowNodeResult{node: r}, ok
	default:
		if isSink {
			r, ok := buildChannelWriteNode(name, sym, kg)
			return flowNodeResult{node: r}, ok
		}
		r, ok := buildChannelReadNode(name, sym, kg)
		return flowNodeResult{node: r}, ok
	}
}

// analyzeSequenceRef builds a transition intent for an identifier that
// resolves to a sequence (or a top-level stage whose symbol node happens to
// be an IStageDeclarationContext). The intent says "when the firing handle
// is truthy, activate `sym.Name`". If the reference appears inside another
// sequence, the consumer will additionally emit an exit transition.
func analyzeSequenceRef(
	ctx acontext.Context[parser.IIdentifierContext],
	seqSym *symbol.Scope,
) (transitionIntent, bool) {
	if _, ok := seqSym.AST.(parser.IStageDeclarationContext); ok {
		return transitionIntent{activateKey: seqSym.Name}, true
	}
	seqDecl, ok := seqSym.AST.(parser.ISequenceDeclarationContext)
	if !ok || len(seqDecl.AllSequenceItem()) == 0 {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "sequence '%s' has no steps", seqSym.Name))
		return transitionIntent{}, false
	}
	return transitionIntent{activateKey: seqSym.Name}, true
}

// analyzeStageRef builds a transition intent for an identifier that resolves
// to a stage. A reference to a sibling stage in the enclosing sequence jumps
// to that sibling; a reference to a stage in a different top-level scope
// activates that top-level scope and exits the current sequence.
func analyzeStageRef(stageSym *symbol.Scope, shell *shellBuilder) transitionIntent {
	if frame := shell.top(); frame != nil && stageSym.Parent != nil && stageSym.Parent.Name == frame.name {
		return transitionIntent{memberKey: stageSym.Name}
	}
	topName := stageSym.Name
	if stageSym.Parent != nil && stageSym.Parent.Name != "" {
		topName = stageSym.Parent.Name
	}
	return transitionIntent{activateKey: topName}
}

func buildChannelReadNode(name string, sym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("on", name)
	chKey := uint32(sym.ID)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "on",
		Channels: types.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: sym.Type.Unwrap()}},
	}
	n.Channels.Read[chKey] = sym.Name
	return newNodeResult(n, "", ir.DefaultOutputParam), true
}

func buildChannelWriteNode(name string, sym *symbol.Scope, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("write", name)
	chKey := uint32(sym.ID)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "write",
		Channels: types.NewChannels(),
		Config:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Inputs:   types.Params{{Name: ir.DefaultInputParam, Type: sym.Type.Unwrap()}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
	}
	n.Channels.Write[chKey] = sym.Name
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

func buildGlobalConstantNode(
	name string,
	sym *symbol.Scope,
	kg *keyGenerator,
) (nodeResult, bool) {
	key := kg.generate("const", name)
	n := ir.Node{
		Key:      key,
		Type:     "constant",
		Channels: types.NewChannels(),
		Config:   types.Params{{Name: "value", Type: sym.Type, Value: sym.DefaultValue}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: sym.Type}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

// analyzeNextToken emits a transition intent that advances the enclosing
// sequence to the next sibling member. The target member is resolved against
// the innermost sequence frame at intent-consumption time.
func analyzeNextToken(
	ctx acontext.Context[parser.IFlowNodeContext],
	shell *shellBuilder,
) (flowNodeResult, bool) {
	frame := shell.top()
	if frame == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "'next' used outside of a sequence"))
		return flowNodeResult{}, false
	}
	if frame.nextMember() == "" {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"'next' in last stage '%s' has no next stage",
			frame.currentMember(),
		))
		return flowNodeResult{}, false
	}
	intent := transitionIntent{isNext: true}
	return flowNodeResult{transition: &intent}, true
}

func analyzeFunctionNode(
	ctx acontext.Context[parser.IFunctionContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	name := parser.FunctionName(ctx.AST)
	key := kg.generate(name, "")
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}
	if sym.Type.Kind != types.KindFunction {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"expected function type, got %s",
			sym.Type,
		))
		return nodeResult{}, false
	}
	freshType := types.Freshen(sym.Type, key)
	n := ir.Node{
		Key:      key,
		Type:     name,
		Channels: sym.Channels.Copy(),
		Config:   slices.Clone(freshType.Config),
		Outputs:  slices.Clone(freshType.Outputs),
		Inputs:   slices.Clone(freshType.Inputs),
	}
	var ok bool
	n.Config, ok = extractConfigValues(acontext.Child(ctx, ctx.AST.ConfigValues()), n.Config, n, sym)
	if !ok {
		return nodeResult{}, false
	}
	return newNodeResult(n, firstInputParam(n.Inputs), firstOutputParam(n.Outputs)), true
}

func analyzeExpression(
	ctx acontext.Context[parser.IExpressionContext],
	kg *keyGenerator,
) (nodeResult, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}

	if sym.Kind == symbol.KindConstant {
		outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
		literalCtx := parser.GetLiteral(ctx.AST)
		parsedValue, err := literal.Parse(literalCtx, outputType)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return nodeResult{}, false
		}
		key := kg.generate("const", "")
		n := ir.Node{
			Key:      key,
			Type:     "constant",
			Channels: types.NewChannels(),
			Config:   types.Params{{Name: "value", Type: outputType, Value: parsedValue.Value}},
			Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
		}
		return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
	}

	key := kg.generate(sym.Name, "")
	freshType := types.Freshen(sym.Type, key)
	if len(freshType.Outputs) == 0 || !freshType.Outputs[0].Type.IsValid() {
		exprText := strings.TrimSuffix(ctx.AST.GetText(), "()")
		d := diagnostics.Errorf(
			ctx.AST,
			"functions in flow statements use {} not ()",
		)
		d.Notes = append(d.Notes, diagnostics.Note{
			Message: fmt.Sprintf("did you mean: %s{}?", exprText),
		})
		ctx.Diagnostics.Add(d)
		return nodeResult{}, false
	}
	outputType := ctx.Constraints.ApplySubstitutions(freshType.Outputs[0].Type)
	n := ir.Node{
		Key:      key,
		Type:     sym.Name,
		Channels: sym.Channels.Copy(),
		Inputs:   freshType.Inputs,
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

// Analyze performs semantic analysis on parsed Arc code and builds the IR.
// Returns a partially complete IR even on errors for LSP support.
func Analyze(
	ctx context.Context,
	t Text,
	resolver symbol.Resolver,
) (ir.IR, *diagnostics.Diagnostics) {
	var (
		aCtx = acontext.CreateRoot(ctx, t.AST, resolver)
		i    = ir.IR{Symbols: aCtx.Scope, TypeMap: aCtx.TypeMap}
	)

	analyzer.AnalyzeProgram(aCtx)
	i.Authorities = authority.Analyze(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return i, aCtx.Diagnostics
	}

	for _, c := range i.Symbols.Children {
		if c.Kind == symbol.KindFunction {
			fnDecl, ok := c.AST.(parser.IFunctionDeclarationContext)
			var bodyAst antlr.ParserRuleContext = fnDecl
			if ok {
				bodyAst = fnDecl.Block()
			}
			exprDecl, ok := c.AST.(parser.IExpressionContext)
			if ok {
				bodyAst = exprDecl
			}
			i.Functions = append(i.Functions, ir.Function{
				Key:      c.Name,
				Body:     ir.Body{Raw: bodyAst.GetText(), AST: bodyAst},
				Config:   c.Type.Config,
				Inputs:   c.Type.Inputs,
				Outputs:  c.Type.Outputs,
				Channels: c.Channels,
			})
		}
	}
	kg := newKeyGenerator()
	shell := newShellBuilder()

	// The root scope is always parallel and always-live.
	i.Root = ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}
	// rootMembers accumulates every top-level item as a Member of the root
	// scope. Module-scope flow nodes become leaf-node members; top-level
	// sequence and stage declarations become nested Scope members.
	var rootMembers ir.Members

	for _, item := range t.AST.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			nodes, edges, ok := analyzeFlow(acontext.Child(aCtx, flow), kg, shell)
			if !ok {
				return i, aCtx.Diagnostics
			}
			for _, n := range nodes {
				rootMembers = append(rootMembers, ir.Member{NodeKey: new(n.Key)})
			}
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		} else if seqDecl := item.SequenceDeclaration(); seqDecl != nil {
			seqScope, nodes, edges, ok := analyzeSequence(
				acontext.Child(aCtx, seqDecl),
				kg,
				shell,
			)
			if !ok {
				return i, aCtx.Diagnostics
			}
			rootMembers = append(rootMembers, ir.Member{Scope: &seqScope})
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		} else if stageDecl := item.StageDeclaration(); stageDecl != nil {
			stgScope, nodes, edges, ok := analyzeTopLevelStage(
				acontext.Child(aCtx, stageDecl),
				kg,
				shell,
			)
			if !ok {
				return i, aCtx.Diagnostics
			}
			rootMembers = append(rootMembers, ir.Member{Scope: &stgScope})
			i.Nodes = append(i.Nodes, nodes...)
			i.Edges = append(i.Edges, edges...)
		}
	}

	if len(rootMembers) > 0 {
		i.Root.Strata = []ir.Members{rootMembers}
	}

	// Apply deferred activations collected by flow statements that target
	// top-level scopes (for example `trigger => main`). The activation is
	// stamped directly onto the corresponding nested Scope member.
	if len(shell.activations) > 0 && len(i.Root.Strata) > 0 {
		stratum := i.Root.Strata[0]
		for idx := range stratum {
			m := &stratum[idx]
			if m.Scope == nil {
				continue
			}
			if handle, ok := shell.activations[m.Scope.Key]; ok {
				m.Scope.Activation = new(handle)
			}
		}
	}

	if len(i.Nodes) > 0 {
		if !analyzer.ResolveNodeTypes(i.Nodes, i.Edges, aCtx.Constraints, aCtx.Diagnostics) {
			return i, aCtx.Diagnostics
		}
		if d := stratifier.Stratify(ctx, &i, aCtx.Diagnostics); d != nil && !d.Ok() {
			return i, d
		}
	}
	return i, aCtx.Diagnostics
}

type flowChainProcessor struct {
	kg                 *keyGenerator
	shell              *shellBuilder
	prevNode           *ir.Node
	ctx                acontext.Context[parser.IFlowStatementContext]
	prevOutput         ir.Handle
	nodes              []ir.Node
	edges              []ir.Edge
	additionalTriggers []nodeResult
	totalFlowNodes     int
	currentIndex       int
	lastOpIndex        int
	// transitionEmitted is set when the chain terminated in a transition
	// target (e.g. `=> main`, `=> next`). Used to distinguish valid chains
	// that emit zero edges (source -> scope activation) from orphan chains.
	transitionEmitted bool
}

func newFlowChainProcessor(
	ctx acontext.Context[parser.IFlowStatementContext],
	kg *keyGenerator,
	shell *shellBuilder,
) *flowChainProcessor {
	var total int
	for _, child := range ctx.AST.GetChildren() {
		if _, ok := child.(parser.IFlowNodeContext); ok {
			total++
		}
	}
	return &flowChainProcessor{ctx: ctx, kg: kg, shell: shell, totalFlowNodes: total}
}

func (p *flowChainProcessor) edgeKind() ir.EdgeKind {
	children := p.ctx.AST.GetChildren()
	if p.lastOpIndex < 0 || p.lastOpIndex >= len(children) {
		return ir.EdgeKindContinuous
	}
	if opCtx, ok := children[p.lastOpIndex].(parser.IFlowOperatorContext); ok && opCtx.TRANSITION() != nil {
		return ir.EdgeKindConditional
	}
	return ir.EdgeKindContinuous
}

// injectImplicitTriggers creates channel read nodes for all channels referenced
// in an expression when that expression is the first node in a flow statement.
// This enables the shorthand syntax: `ox_pt_1 > 20 => do_something{}`
// which expands to: `ox_pt_1 -> ox_pt_1 > 20 => do_something{}`
func (p *flowChainProcessor) injectImplicitTriggers(expr parser.IExpressionContext) bool {
	sym, err := p.ctx.Scope.Root().GetChildByParserRule(expr)
	if err != nil || sym.Kind == symbol.KindConstant {
		return true // Constants don't need triggers
	}

	if len(sym.Channels.Read) == 0 {
		return true // No channels referenced
	}

	// Create trigger node for each channel
	for _, chName := range sym.Channels.Read {
		chanSym, err := p.ctx.Scope.Resolve(p.ctx, chName)
		if err != nil {
			continue
		}
		result, ok := buildChannelReadNode(chName, chanSym, p.kg)
		if !ok {
			return false
		}
		p.nodes = append(p.nodes, result.node)

		if p.prevNode == nil {
			p.prevOutput = result.output
			p.prevNode = &result.node
		} else {
			p.additionalTriggers = append(p.additionalTriggers, result)
		}
	}
	return true
}

func (p *flowChainProcessor) processFlowNode(flowNode parser.IFlowNodeContext) bool {
	p.currentIndex++
	isLast := p.currentIndex == p.totalFlowNodes
	isSink := isLast && flowNode.Identifier() != nil

	// Inject implicit triggers for expression as first node
	if p.currentIndex == 1 && p.prevNode == nil {
		if expr := flowNode.Expression(); expr != nil {
			if !p.injectImplicitTriggers(expr) {
				return false
			}
		}
	}

	result, ok := analyzeFlowNode(acontext.Child(p.ctx, flowNode), p.kg, p.shell, isSink)
	if !ok {
		return false
	}

	if result.transition != nil {
		return p.consumeTransition(*result.transition, flowNode)
	}

	node := result.node
	if p.prevNode != nil {
		if len(p.prevNode.Outputs) == 0 {
			p.ctx.Diagnostics.Add(diagnostics.Errorf(
				flowNode,
				"function '%s' has no output to connect in flow chain",
				p.prevNode.Type,
			))
			return false
		}
		p.edges = append(p.edges, ir.Edge{
			Source: p.prevOutput,
			Target: node.input,
			Kind:   p.edgeKind(),
		})
	}

	// Handle additional triggers (for expressions with multiple channel references)
	if len(p.additionalTriggers) > 0 {
		for _, trigger := range p.additionalTriggers {
			p.edges = append(p.edges, ir.Edge{
				Source: trigger.output,
				Target: node.input,
				Kind:   ir.EdgeKindContinuous,
			})
		}
		p.additionalTriggers = nil
	}

	if len(node.node.Outputs) > 0 {
		p.prevOutput = node.output
	}
	p.prevNode = &node.node
	if node.node.Key != "" {
		p.nodes = append(p.nodes, node.node)
	}
	return true
}

// consumeTransition records a transition and/or activation for a flow chain
// whose terminal token is `=> next`, `=> scope_name`, or a scope-valued
// identifier. The firing handle is the previous node's output.
func (p *flowChainProcessor) consumeTransition(
	intent transitionIntent,
	ast parser.IFlowNodeContext,
) bool {
	if p.prevNode == nil {
		p.ctx.Diagnostics.Add(diagnostics.Errorf(
			ast, "transition target requires a source",
		))
		return false
	}
	if len(p.prevNode.Outputs) == 0 {
		p.ctx.Diagnostics.Add(diagnostics.Errorf(
			ast,
			"function '%s' has no output to drive a transition",
			p.prevNode.Type,
		))
		return false
	}
	on := p.prevOutput

	// When a multi-channel expression drives a transition there is no IR
	// node to route the extra triggers into; the primary channel output
	// already carries the firing signal, so the extras are dropped.
	p.additionalTriggers = nil

	switch {
	case intent.isNext:
		next := p.shell.top().nextMember()
		p.shell.addTransition(ir.Transition{
			On:        on,
			TargetKey: new(next),
		})
	case intent.memberKey != "":
		p.shell.addTransition(ir.Transition{
			On:        on,
			TargetKey: new(intent.memberKey),
		})
	case intent.activateKey != "":
		p.shell.registerActivation(intent.activateKey, on)
		if p.shell.inSequence() {
			p.shell.addTransition(ir.Transition{
				On:        on,
				TargetKey: nil,
			})
		}
	}
	p.transitionEmitted = true
	// prevNode stays set so that subsequent chain logic could detect misuse;
	// in practice a transition token must be the last node in the chain.
	return true
}

func (p *flowChainProcessor) processRoutingTable(rt parser.IRoutingTableContext) bool {
	if p.prevNode == nil {
		p.ctx.Diagnostics.Add(diagnostics.Errorf(
			p.ctx.AST,
			"input routing tables not yet implemented in text compiler",
		))
		return false
	}
	newNodes, newEdges, ok := analyzeOutputRoutingTable(
		acontext.Child(p.ctx, rt),
		*p.prevNode,
		p.prevOutput,
		p.kg,
		p.shell,
	)
	if !ok {
		return false
	}
	p.nodes = append(p.nodes, newNodes...)
	p.edges = append(p.edges, newEdges...)
	p.prevNode = nil
	return true
}

func analyzeFlow(
	ctx acontext.Context[parser.IFlowStatementContext],
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	p := newFlowChainProcessor(ctx, kg, shell)
	for i, child := range ctx.AST.GetChildren() {
		switch c := child.(type) {
		case parser.IFlowNodeContext:
			if !p.processFlowNode(c) {
				return nil, nil, false
			}
		case parser.IFlowOperatorContext:
			p.lastOpIndex = i
		case parser.IRoutingTableContext:
			if !p.processRoutingTable(c) {
				return nil, nil, false
			}
		}
	}
	if len(p.edges) < 1 && !p.transitionEmitted {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"flow statement requires at least two nodes",
		))
		return nil, nil, false
	}
	return p.nodes, p.edges, true
}

func extractConfigValues(
	ctx acontext.Context[parser.IConfigValuesContext],
	config types.Params,
	node ir.Node,
	fnSym *symbol.Scope,
) (types.Params, bool) {
	if ctx.AST == nil {
		return config, true
	}

	parseConfigExpr := func(
		expr parser.IExpressionContext,
		paramType types.Type,
		paramName string,
	) (any, bool) {
		if paramType.Kind == types.KindChan {
			channelName := parser.GetExpressionText(expr)
			sym, err := ctx.Scope.Resolve(ctx, channelName)
			if err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, expr))
				return nil, false
			}
			if err := paramType.ChanDirection.CheckCompatibility(sym.Type.ChanDirection); err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, expr))
				return nil, false
			}
			channelKey := uint32(sym.ID)
			symbol.ResolveConfigChannel(&node.Channels, fnSym, paramName, channelKey, sym.Name)
			return channelKey, true
		}

		if primary := parser.GetPrimaryExpression(expr); primary != nil {
			if id := primary.IDENTIFIER(); id != nil {
				sym, err := ctx.Scope.Resolve(ctx, id.GetText())
				if err != nil {
					ctx.Diagnostics.Add(diagnostics.Error(err, expr))
					return nil, false
				}
				if sym.Kind == symbol.KindGlobalConstant {
					return sym.DefaultValue, true
				}
			}
		}

		if !parser.IsLiteral(expr) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				expr,
				"config value for '%s' must be a literal or global constant",
				paramName,
			))
			return nil, false
		}

		literalCtx := parser.GetLiteral(expr)
		parsedValue, err := literal.Parse(literalCtx, paramType)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, expr))
			return nil, false
		}
		return parsedValue.Value, true
	}

	if named := ctx.AST.NamedConfigValues(); named != nil {
		for _, cv := range named.AllNamedConfigValue() {
			key := cv.IDENTIFIER().GetText()
			idx := config.GetIndex(key)
			if expr := cv.Expression(); expr != nil {
				value, ok := parseConfigExpr(expr, config[idx].Type, key)
				if !ok {
					return nil, false
				}
				config[idx].Value = value
			}
		}
	} else if anon := ctx.AST.AnonymousConfigValues(); anon != nil {
		for i, expr := range anon.AllExpression() {
			value, ok := parseConfigExpr(expr, config[i].Type, fmt.Sprintf("position %d", i))
			if !ok {
				return nil, false
			}
			config[i].Value = value
		}
	}

	return config, true
}

func analyzeOutputRoutingTable(
	ctx acontext.Context[parser.IRoutingTableContext],
	sourceNode ir.Node,
	sourceHandle ir.Handle,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	var (
		nodes []ir.Node
		edges []ir.Edge
	)

	for _, entry := range ctx.AST.AllRoutingEntry() {
		outputName := entry.IDENTIFIER(0).GetText()
		if !sourceNode.Outputs.Has(outputName) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				entry,
				"node '%s' does not have output '%s'",
				sourceNode.Key,
				outputName,
			))
			return nil, nil, false
		}

		flowNodes := entry.AllFlowNode()
		if len(flowNodes) == 0 {
			continue
		}

		var targetParamName string
		if len(entry.AllIDENTIFIER()) > 1 {
			targetParamName = entry.IDENTIFIER(1).GetText()
		}

		sourceOutput := ir.Handle{Node: sourceNode.Key, Param: outputName}
		prevOutputHandle := sourceOutput
		for i, flowNode := range flowNodes {
			isLast := i == len(flowNodes)-1
			isSink := isLast && flowNode.Identifier() != nil

			result, ok := analyzeFlowNode(acontext.Child(ctx, flowNode), kg, shell, isSink)
			if !ok {
				return nil, nil, false
			}

			if result.transition != nil {
				// Terminal scope reference in a routing branch: translate
				// into an activation (and, if the branch originates inside
				// a sequence, an exit transition).
				intent := *result.transition
				if intent.activateKey != "" {
					shell.registerActivation(intent.activateKey, prevOutputHandle)
					if shell.inSequence() {
						shell.addTransition(ir.Transition{
							On:        prevOutputHandle,
							TargetKey: nil,
						})
					}
					continue
				}
				if intent.memberKey != "" {
					shell.addTransition(ir.Transition{
						On:        prevOutputHandle,
						TargetKey: new(intent.memberKey),
					})
					continue
				}
				if intent.isNext {
					next := shell.top().nextMember()
					shell.addTransition(ir.Transition{
						On:        prevOutputHandle,
						TargetKey: new(next),
					})
					continue
				}
				continue
			}

			node := result.node
			edges = append(edges, ir.Edge{
				Source: prevOutputHandle,
				Target: node.input,
				Kind:   ir.EdgeKindContinuous,
			})

			if isLast && targetParamName != "" {
				if !node.node.Inputs.Has(targetParamName) {
					ctx.Diagnostics.Add(diagnostics.Errorf(
						entry,
						"node '%s' does not have input '%s'",
						node.node.Key,
						targetParamName,
					))
					return nil, nil, false
				}
				edges[len(edges)-1].Target.Param = targetParamName
			}

			if len(node.node.Outputs) > 0 {
				prevOutputHandle = node.output
			}
			if node.node.Key != "" {
				nodes = append(nodes, node.node)
			}
		}
	}

	return nodes, edges, true
}

// stepInfo collects metadata about a step for computing member keys.
type stepInfo struct {
	key  string
	item parser.ISequenceItemContext
}

// collectStepKeys pre-scans a sequence's items to compute their member keys.
// Named stages and nested sequences keep their source-level name; anonymous
// flow and single-invocation steps receive a synthesized "step_N" key.
func collectStepKeys(items []parser.ISequenceItemContext) []stepInfo {
	steps := make([]stepInfo, 0, len(items))
	for i, item := range items {
		key := fmt.Sprintf("step_%d", i)
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			if id := stageDecl.IDENTIFIER(); id != nil {
				key = id.GetText()
			}
		}
		if nestedSeq := item.SequenceDeclaration(); nestedSeq != nil {
			if id := nestedSeq.IDENTIFIER(); id != nil {
				key = id.GetText()
			}
		}
		steps = append(steps, stepInfo{key: key, item: item})
	}
	return steps
}

// flowScope wraps a set of flow-step nodes into a parallel+gated scope whose
// single stratum contains them in source order. The stratifier will later
// re-layer this stratum; for now all members sit in stratum 0.
func flowScope(key string, nodes []ir.Node) ir.Scope {
	scope := ir.Scope{
		Key:      key,
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessGated,
	}
	if len(nodes) == 0 {
		return scope
	}
	members := make(ir.Members, 0, len(nodes))
	for _, n := range nodes {
		members = append(members, ir.Member{NodeKey: new(n.Key)})
	}
	scope.Strata = []ir.Members{members}
	return scope
}

// autoWireTransition appends an auto-wired transition for a flow-step in a
// sequence: when the step's last node fires, advance to the next step or
// exit the sequence if the step is terminal.
func autoWireTransition(shell *shellBuilder, lastNode ir.Node, nextMemberKey string) {
	if len(lastNode.Outputs) == 0 {
		return
	}
	on := ir.Handle{
		Node:  lastNode.Key,
		Param: firstOutputParam(lastNode.Outputs),
	}
	var targetKey *string
	if nextMemberKey != "" {
		targetKey = new(nextMemberKey)
	}
	shell.addTransition(ir.Transition{On: on, TargetKey: targetKey})
}

func analyzeSequence(
	ctx acontext.Context[parser.ISequenceDeclarationContext],
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, []ir.Node, []ir.Edge, bool) {
	var seqScope *symbol.Scope
	if id := ctx.AST.IDENTIFIER(); id != nil {
		resolved, err := ctx.Scope.Resolve(ctx, id.GetText())
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return ir.Scope{}, nil, nil, false
		}
		seqScope = resolved
	} else {
		resolved, err := ctx.Scope.GetChildByParserRule(ctx.AST)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return ir.Scope{}, nil, nil, false
		}
		seqScope = resolved
	}
	seqName := seqScope.Name
	scope := ir.Scope{
		Key:      seqName,
		Mode:     ir.ScopeModeSequential,
		Liveness: ir.LivenessGated,
	}

	items := ctx.AST.AllSequenceItem()
	steps := collectStepKeys(items)
	memberKeys := make([]string, len(steps))
	for i, s := range steps {
		memberKeys[i] = s.key
	}

	frame := shell.pushSeq(seqName, memberKeys)
	defer shell.popSeq()

	var (
		allNodes []ir.Node
		allEdges []ir.Edge
	)

	for i, si := range steps {
		frame.activeIdx = i
		nextKey := ""
		if i+1 < len(steps) {
			nextKey = steps[i+1].key
		}

		item := si.item
		if stageDecl := item.StageDeclaration(); stageDecl != nil {
			stgScope, nodes, edges, ok := analyzeStage(
				acontext.Child(ctx, stageDecl).WithScope(seqScope),
				kg,
				shell,
			)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			// Anonymous inline stages inherit the synthesized step key as
			// their own Key so Member.Key() is derivable from the scope.
			if stgScope.Key == "" {
				stgScope.Key = si.key
			}
			scope.Steps = append(scope.Steps, ir.Member{Scope: &stgScope})
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
			continue
		}

		if flowStmt := item.FlowStatement(); flowStmt != nil {
			nodes, edges, ok := analyzeFlow(
				acontext.Child(ctx, flowStmt).WithScope(seqScope),
				kg,
				shell,
			)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			child := flowScope(si.key, nodes)
			scope.Steps = append(scope.Steps, ir.Member{Scope: &child})
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
			if len(nodes) > 0 {
				autoWireTransition(shell, nodes[len(nodes)-1], nextKey)
			}
			continue
		}

		if single := item.SingleInvocation(); single != nil {
			node, ok := analyzeSingleInvocation(
				acontext.Child(ctx, single).WithScope(seqScope),
				kg,
			)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			child := flowScope(si.key, []ir.Node{node})
			scope.Steps = append(scope.Steps, ir.Member{Scope: &child})
			allNodes = append(allNodes, node)
			autoWireTransition(shell, node, nextKey)
			continue
		}

		if nestedSeqDecl := item.SequenceDeclaration(); nestedSeqDecl != nil {
			nestedScope, nodes, edges, ok := analyzeSequence(
				acontext.Child(ctx, nestedSeqDecl).WithScope(seqScope),
				kg,
				shell,
			)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			// Anonymous inline nested sequences inherit the synthesized step
			// key so Member.Key() is derivable from the scope.
			if nestedScope.Key == "" {
				nestedScope.Key = si.key
			}
			scope.Steps = append(scope.Steps, ir.Member{Scope: &nestedScope})
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
		}
	}

	scope.Transitions = frame.transitions
	return scope, allNodes, allEdges, true
}

func analyzeTopLevelStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, []ir.Node, []ir.Edge, bool) {
	// Resolve the symbol scope registered by collectTopLevelStage so that
	// anonymous top-level stages pick up the auto-generated name (e.g.,
	// "stage_0") and the resulting ir.Scope has a non-empty, unique Key —
	// otherwise anonymous stages would collide at the root member level.
	var stageSym *symbol.Scope
	if id := ctx.AST.IDENTIFIER(); id != nil {
		resolved, err := ctx.Scope.Resolve(ctx, id.GetText())
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return ir.Scope{}, nil, nil, false
		}
		stageSym = resolved
	} else {
		resolved, err := ctx.Scope.GetChildByParserRule(ctx.AST)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return ir.Scope{}, nil, nil, false
		}
		stageSym = resolved
	}
	scope, nodes, edges, ok := analyzeStage(ctx, kg, shell)
	if !ok {
		return ir.Scope{}, nil, nil, false
	}
	scope.Key = stageSym.Name
	return scope, nodes, edges, true
}

func analyzeStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, []ir.Node, []ir.Edge, bool) {
	stageName := ""
	if id := ctx.AST.IDENTIFIER(); id != nil {
		stageName = id.GetText()
	}
	scope := ir.Scope{
		Key:      stageName,
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessGated,
	}
	var (
		nodes   []ir.Node
		edges   []ir.Edge
		members []ir.Member
	)

	stageBody := ctx.AST.StageBody()
	if stageBody == nil {
		return scope, nodes, edges, true
	}

	for _, item := range stageBody.AllStageItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			itemNodes, itemEdges, ok := analyzeFlow(
				acontext.Child(ctx, flowStmt),
				kg,
				shell,
			)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			nodes = append(nodes, itemNodes...)
			edges = append(edges, itemEdges...)
			for _, n := range itemNodes {
				members = append(members, ir.Member{NodeKey: new(n.Key)})
			}
			continue
		}
		if single := item.SingleInvocation(); single != nil {
			node, ok := analyzeSingleInvocation(acontext.Child(ctx, single), kg)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			nodes = append(nodes, node)
			members = append(members, ir.Member{NodeKey: new(node.Key)})
			continue
		}
		if nestedSeqDecl := item.SequenceDeclaration(); nestedSeqDecl != nil {
			// The inline sequence is registered as a child of this stage's
			// scope, not the parent sequence's scope. Look up the stage's own
			// scope so analyzeSequence can resolve the nested seq via parser
			// rule. Top-level stages don't have their own scope entry; in
			// that case keep ctx.Scope.
			seqCtx := acontext.Child(ctx, nestedSeqDecl)
			if stageScope, err := ctx.Scope.GetChildByParserRule(ctx.AST); err == nil {
				seqCtx = seqCtx.WithScope(stageScope)
			}
			subScope, subNodes, subEdges, ok := analyzeSequence(seqCtx, kg, shell)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			nodes = append(nodes, subNodes...)
			edges = append(edges, subEdges...)
			members = append(members, ir.Member{Scope: &subScope})
			continue
		}
	}

	if len(members) > 0 {
		scope.Strata = []ir.Members{members}
	}
	return scope, nodes, edges, true
}

func analyzeSingleInvocation(
	ctx acontext.Context[parser.ISingleInvocationContext],
	kg *keyGenerator,
) (ir.Node, bool) {
	if fn := ctx.AST.Function(); fn != nil {
		result, ok := analyzeFunctionNode(acontext.Child(ctx, fn), kg)
		if !ok {
			return ir.Node{}, false
		}
		return result.node, true
	}
	if expr := ctx.AST.Expression(); expr != nil {
		result, ok := analyzeExpression(acontext.Child(ctx, expr), kg)
		if !ok {
			return ir.Node{}, false
		}
		return result.node, true
	}
	return ir.Node{}, false
}
