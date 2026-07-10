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
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// keyGenerator produces globally unique IR node keys. It maintains a running
// per-role counter so that successive channel reads, writes, or function
// invocations with the same logical name receive distinct keys.
type keyGenerator struct {
	occurrences map[string]int
	synthFuncs  *ir.Functions
}

func newKeyGenerator(synthFuncs *ir.Functions) *keyGenerator {
	return &keyGenerator{occurrences: make(map[string]int), synthFuncs: synthFuncs}
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
	// inlineNodes and inlineEdges accumulate the flat IR of every lowered inline
	// body; the bodies' scopes are placed in their enclosing scope's members.
	inlineNodes []ir.Node
	inlineEdges []ir.Edge
	// inlineBodyBases records the stack length at the entry of each enclosing
	// inline routing case body; `next` is rejected if no frame was pushed since.
	inlineBodyBases []int
	// synthByAST maps each inline-body declaration to its synth scope, keyed by
	// the declaration's parser node.
	synthByAST map[antlr.ParserRuleContext]*symbol.Symbol
	// varChannels holds the channel keys backing reactive value variables.
	varChannels set.Set[uint32]
	// reExprs records reactive re-expressions (`r = new_expr`) in walk order, so a
	// post-pass can assemble each reactive var's feeder state machine.
	reExprs []reExprFeeder
	// aliasBacking maps each reassigned channel alias to the reactive channel its reads use.
	aliasBacking map[*symbol.Symbol]*symbol.Symbol
	// aliasWriteBackings maps each reassigned channel alias to its write-redirection machine.
	aliasWriteBackings map[*symbol.Symbol]*aliasWriteMux
	// aliasWriteOrder lists reassigned aliases in first-seen order for deterministic lowering.
	aliasWriteOrder []*symbol.Symbol
	// aliasBindingBody maps a reassigned alias to the leaf body of its most recent binding.
	aliasBindingBody map[*symbol.Symbol]*symbol.Symbol
	// aliasCrossBodyRead and aliasCrossBodyWrite hold aliases whose reads or writes routed
	// through the machine; a direction with none needs none.
	aliasCrossBodyRead  set.Set[*symbol.Symbol]
	aliasCrossBodyWrite set.Set[*symbol.Symbol]
}

// leafBody returns the nearest enclosing stage or sequence scope.
func leafBody(scope *symbol.Symbol) *symbol.Symbol {
	for s := scope; s != nil; s = s.Parent {
		if s.Kind == symbol.KindStage || s.Kind == symbol.KindSequence {
			return s
		}
	}
	return scope
}

// boundInSameBody reports whether alias's current binding was set in refScope's leaf body,
// in which case the reference bakes that binding instead of routing through the machine.
func (s *shellBuilder) boundInSameBody(alias, refScope *symbol.Symbol) bool {
	bind, ok := s.aliasBindingBody[alias]
	if !ok {
		return false
	}
	return leafBody(bind) == leafBody(refScope)
}

// aliasWriteMux holds the write-redirection machine for one reassigned alias.
type aliasWriteMux struct {
	// backing is the reactive channel the alias's writes are redirected to.
	backing *symbol.Symbol
	// initBinding is the channel the alias is bound to at its declaration.
	initBinding *symbol.Symbol
	// feeders forward backing to each rebind's channel, selected by the shared switch.
	feeders []reExprFeeder
}

// reExprFeeder records a reactive re-expression: its feeder subgraph and switch channel.
type reExprFeeder struct {
	// target is the reactive variable being re-expressed.
	target *symbol.Symbol
	// nodes and edges are the feeder subgraph computing the new expression.
	nodes []ir.Node
	edges []ir.Edge
	// switchSym backs a program-local signal channel the reassignment writes and the
	// feeder machine reads to advance to this feeder; it persists across walk cycles.
	switchSym *symbol.Symbol
}

// recordReExpr records a reactive re-expression for the feeder-machine post-pass.
func (s *shellBuilder) recordReExpr(f reExprFeeder) {
	s.reExprs = append(s.reExprs, f)
}

// isReExpressed reports whether sym is the target of any recorded re-expression.
func (s *shellBuilder) isReExpressed(sym *symbol.Symbol) bool {
	for _, f := range s.reExprs {
		if f.target == sym {
			return true
		}
	}
	return false
}

// reExprTargets returns each re-expressed variable once, in first-seen order.
func (s *shellBuilder) reExprTargets() []*symbol.Symbol {
	seen := set.New[*symbol.Symbol]()
	var out []*symbol.Symbol
	for _, f := range s.reExprs {
		if !seen.Contains(f.target) {
			seen.Add(f.target)
			out = append(out, f.target)
		}
	}
	return out
}

// reExprsFor returns target's recorded re-expressions in execution order.
func (s *shellBuilder) reExprsFor(target *symbol.Symbol) []reExprFeeder {
	var out []reExprFeeder
	for _, f := range s.reExprs {
		if f.target == target {
			out = append(out, f)
		}
	}
	return out
}

func newShellBuilder(synthByAST map[antlr.ParserRuleContext]*symbol.Symbol) *shellBuilder {
	return &shellBuilder{
		activations:         map[string]ir.Handle{},
		synthByAST:          synthByAST,
		varChannels:         set.New[uint32](),
		aliasBacking:        map[*symbol.Symbol]*symbol.Symbol{},
		aliasWriteBackings:  map[*symbol.Symbol]*aliasWriteMux{},
		aliasBindingBody:    map[*symbol.Symbol]*symbol.Symbol{},
		aliasCrossBodyRead:  set.New[*symbol.Symbol](),
		aliasCrossBodyWrite: set.New[*symbol.Symbol](),
	}
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

// inlineBoundaryBlocksNext returns true inside an inline stage body (no own
// frame) or at the last step of an inline sequence body (no further step).
func (s *shellBuilder) inlineBoundaryBlocksNext() bool {
	n := len(s.inlineBodyBases)
	if n == 0 {
		return false
	}
	base := s.inlineBodyBases[n-1]
	if len(s.stack) == base {
		return true
	}
	if len(s.stack) == base+1 && s.stack[len(s.stack)-1].nextMember() == "" {
		return true
	}
	return false
}

// addTransition appends a transition to the innermost sequence frame. Panics
// if no sequence is active; callers must check top() first.
func (s *shellBuilder) addTransition(t ir.Transition) {
	s.stack[len(s.stack)-1].transitions = append(
		s.stack[len(s.stack)-1].transitions, t,
	)
}

// addTransitionTo appends a transition to a specific frame, not necessarily
// the innermost. Used when a cross-level `=> X` resolves to a frame further
// up the stack — the transition must live on the frame that owns X so the
// scheduler advances that frame's active step.
func (s *shellBuilder) addTransitionTo(f *seqFrame, t ir.Transition) {
	f.transitions = append(f.transitions, t)
}

// resolveTargetFrame walks the shell stack innermost-first and returns the
// first frame whose memberKeys contain name, or nil if none does. This
// implements the lexical-scope rule for `=> X`: innermost enclosing sequence
// that has X as a member wins (so local names shadow outer ones).
func (s *shellBuilder) resolveTargetFrame(name string) *seqFrame {
	for _, v := range slices.Backward(s.stack) {
		if slices.Contains(v.memberKeys, name) {
			return v
		}
	}
	return nil
}

// registerActivation records that the scope named key should be activated by
// the given handle. The activation is stamped onto the emitted Scope by the
// main Analyze loop once all top-level items have been processed.
func (s *shellBuilder) registerActivation(key string, on ir.Handle) {
	s.activations[key] = on
}

// applyTransitionIntent records a transition and/or activation against the
// shell for a firing handle. Exactly one of isNext, memberKey, activateKey is
// honored, in that priority; a zero intent is a no-op. When the intent is a
// cross-scope activation and the shell is inside a sequence, an additional
// exit transition is appended so the current sequence relinquishes control.
func (s *shellBuilder) applyTransitionIntent(on ir.Handle, intent transitionIntent) {
	switch {
	case intent.isNext:
		next := s.top().nextMember()
		s.addTransition(ir.Transition{On: on, TargetKey: new(next)})
	case intent.memberKey != "":
		frame := intent.targetFrame
		if frame == nil {
			frame = s.top()
		}
		s.addTransitionTo(frame, ir.Transition{On: on, TargetKey: new(intent.memberKey)})
	case intent.activateKey != "":
		s.registerActivation(intent.activateKey, on)
		if s.top() != nil && !intent.suppressExit {
			s.addTransition(ir.Transition{On: on})
		}
	}
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
	// to transition to. The transition lives on targetFrame (or shell.top()
	// when targetFrame is nil — the same-level fallback).
	memberKey string
	// targetFrame is the frame that owns memberKey. When set to a frame
	// further up the stack than shell.top(), the transition fires on an
	// enclosing sequence and runtime deactivation of intermediate scopes
	// cascades via deactivateStep; no explicit exits are needed on inner
	// frames because they freeze when their parent step is deactivated and
	// are reset to step 0 on the next activation (scheduler.go
	// deactivateScope / activateScope).
	targetFrame *seqFrame
	// activateKey, when non-empty, names a top-level scope whose activation
	// should be set to the firing handle. Combined with an exit transition
	// when the intent is consumed inside a sequence.
	activateKey string
	// suppressExit skips the activateKey exit transition so the enclosing
	// sequence keeps running instead of deactivating.
	suppressExit bool
}

// flowNodeResult is what analyzeFlowNode returns: either an actual IR node
// (the usual case) or a transition intent (for `=> next` / `=> scope_name`
// targets). Exactly one of node.node.Key or transition is non-zero.
type flowNodeResult struct {
	node       nodeResult
	transition *transitionIntent
	// inlineScope is the lowered body of an inline stage/sequence flow target,
	// to be placed as a member of the scope enclosing this flow.
	inlineScope *ir.Scope
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
	if s := ctx.AST.StageDeclaration(); s != nil {
		return analyzeInlineBody(ctx, s, kg, shell)
	}
	if s := ctx.AST.SequenceDeclaration(); s != nil {
		return analyzeInlineBody(ctx, s, kg, shell)
	}
	return flowNodeResult{}, true
}

// analyzeInlineBody lowers an anonymous inline stage/sequence body used as a
// flow target, returning an activation intent so the upstream handle gates it.
func analyzeInlineBody(
	ctx acontext.Context[parser.IFlowNodeContext],
	decl antlr.ParserRuleContext,
	kg *keyGenerator,
	shell *shellBuilder,
) (flowNodeResult, bool) {
	synth := shell.synthByAST[decl]
	if synth == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(decl,
			"internal: synth scope not registered for inline body"))
		return flowNodeResult{}, false
	}
	scope, ok := processInlineBody(ctx, synth, kg, shell)
	if !ok {
		return flowNodeResult{}, false
	}
	return flowNodeResult{
		transition: &transitionIntent{
			activateKey:  synth.Name,
			suppressExit: true,
		},
		inlineScope: &scope,
	}, true
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
	case symbol.KindSequence, symbol.KindStage:
		intent, ok := analyzeNamedRef(ctx, sym, shell)
		if !ok {
			return flowNodeResult{}, false
		}
		return flowNodeResult{transition: &intent}, true
	default:
		if isVarChannel(sym) {
			shell.varChannels.Add(channelKey(sym))
		}
		if isSink {
			if mux, ok := shell.aliasWriteBackings[sym]; ok && !shell.boundInSameBody(sym, ctx.Scope) {
				shell.aliasCrossBodyWrite.Add(sym)
				sym = mux.backing
			}
			r, ok := buildChannelWriteNode(name, sym, kg)
			return flowNodeResult{node: r}, ok
		}
		if backing, ok := shell.aliasBacking[sym]; ok && !shell.boundInSameBody(sym, ctx.Scope) {
			shell.aliasCrossBodyRead.Add(sym)
			sym = backing
		}
		r, ok := buildChannelReadNode(name, sym, kg)
		return flowNodeResult{node: r}, ok
	}
}

// analyzeNamedRef builds a transition intent for `=> X` where X is a named
// stage or sequence. Resolution is lexical:
//
//  1. Walk the shell stack innermost-first; the first enclosing sequence
//     frame that has X as a direct member owns the transition (same-level
//     sibling jump — works across any number of intermediate stages or
//     nested sequences because deactivation cascades through
//     deactivateStep).
//  2. If no enclosing frame owns X, X must be a top-level scope (declared
//     directly under the file root). Register a cross-scope activation.
//  3. Otherwise, X is unreachable — emit a diagnostic.
//
// This rule makes sequences behave as black boxes from the outside: you can
// jump around inside your own nesting, you can call a top-level neighbor at
// its front door, but you cannot reach across a structural boundary to land
// on some other sequence's internal step.
func analyzeNamedRef(
	ctx acontext.Context[parser.IIdentifierContext],
	sym *symbol.Symbol,
	shell *shellBuilder,
) (transitionIntent, bool) {
	if frame := shell.resolveTargetFrame(sym.Name); frame != nil {
		return transitionIntent{memberKey: sym.Name, targetFrame: frame}, true
	}
	if isRootLevelScope(sym) {
		if seqDecl, ok := sym.AST.(parser.ISequenceDeclarationContext); ok {
			if len(seqDecl.AllSequenceItem()) == 0 {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					ctx.AST, "sequence '%s' has no steps", sym.Name,
				))
				return transitionIntent{}, false
			}
		}
		return transitionIntent{activateKey: sym.Name}, true
	}
	ctx.Diagnostics.Add(diagnostics.Errorf(
		ctx.AST,
		"'%s' is not reachable from here: it is neither a sibling within an "+
			"enclosing sequence nor a top-level scope",
		sym.Name,
	))
	return transitionIntent{}, false
}

// isRootLevelScope reports whether sym is declared directly under the file
// root (i.e., a top-level sequence or stage). Uses a structural parent
// check so anonymous/auto-named wrapper scopes don't misclassify. The
// user program root's Parent is either nil (no ambient) or KindAmbient
// (the STL prelude).
func isRootLevelScope(sym *symbol.Symbol) bool {
	if sym == nil || sym.Parent == nil {
		return false
	}
	parent := sym.Parent
	if parent.Parent != nil && parent.Parent.Kind != symbol.KindAmbient {
		return false
	}
	return sym.Kind == symbol.KindSequence || sym.Kind == symbol.KindStage
}

// channelKey returns the channel key sym refers to: its SourceID when sym is an
// alias bound to another channel (cpu := some_channel), otherwise its own ID.
func channelKey(sym *symbol.Symbol) uint32 {
	if sym.SourceID != nil {
		return uint32(*sym.SourceID)
	}
	return uint32(sym.ID)
}

// isVarChannel reports whether sym is a value variable backed by an internal channel.
func isVarChannel(sym *symbol.Symbol) bool {
	return sym.VarKind == symbol.VarKindReactive ||
		sym.VarKind == symbol.VarKindConstant
}

// irVarKind maps a symbol's variable kind to its IR node representation.
func irVarKind(k symbol.VarKind) ir.VarKind {
	switch k {
	case symbol.VarKindChannelAlias:
		return ir.VarKindChannelAlias
	case symbol.VarKindReactive:
		return ir.VarKindReactive
	case symbol.VarKindConstant:
		return ir.VarKindConstant
	default:
		return ir.VarKindUnspecified
	}
}

// scopeResetChannels returns the channels of `:=` constant variables declared in
// scopeSym, re-seeded on each entry; `$=` variables are omitted so they persist.
func scopeResetChannels(scopeSym *symbol.Symbol) []uint32 {
	var out []uint32
	for _, c := range scopeSym.Children() {
		if !c.Internal && c.Kind == symbol.KindVariable &&
			c.VarKind == symbol.VarKindConstant && c.DefaultValue != nil {
			out = append(out, channelKey(c))
		}
	}
	slices.Sort(out)
	return out
}

// chanAndValueTypes returns the channel-param and value types for sym, wrapping a value variable's bare type into a channel.
func chanAndValueTypes(sym *symbol.Symbol, write bool) (chanType, valType types.Type) {
	if sym.Type.Kind == types.KindChan {
		return sym.Type, sym.Type.Unwrap()
	}
	if write {
		return types.WriteChan(sym.Type), sym.Type
	}
	return types.ReadChan(sym.Type), sym.Type
}

func buildChannelReadNode(name string, sym *symbol.Symbol, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("on", name)
	chKey := channelKey(sym)
	chanType, valType := chanAndValueTypes(sym, false)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "on",
		Channels: types.NewChannels(),
		Inputs:   types.Params{{Name: "channel", Type: chanType, Value: chKey}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: valType}},
		VarKind:  irVarKind(sym.VarKind),
	}
	n.Channels.Read[chKey] = sym.Name
	return newNodeResult(n, "", ir.DefaultOutputParam), true
}

func buildChannelWriteNode(name string, sym *symbol.Symbol, kg *keyGenerator) (nodeResult, bool) {
	nodeKey := kg.generate("write", name)
	chKey := channelKey(sym)
	chanType, valType := chanAndValueTypes(sym, true)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "write",
		Channels: types.NewChannels(),
		Inputs: types.Params{
			{Name: ir.DefaultInputParam, Type: valType},
			{Name: "channel", Type: chanType, Value: chKey},
		},
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
		VarKind: irVarKind(sym.VarKind),
	}
	n.Channels.Write[chKey] = sym.Name
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

// buildExprReadTriggers builds a read node for each channel the expression
// reads, registering any value-variable channels touched along the way.
func buildExprReadTriggers[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]nodeResult, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(expr)
	if err != nil || sym.Kind == symbol.KindConstant {
		return nil, true
	}
	var reads []nodeResult
	for _, chName := range sym.Channels.Read {
		chanSym, rerr := ctx.Scope.Resolve(ctx, chName)
		if rerr != nil {
			continue
		}
		if backing, ok := shell.aliasBacking[chanSym]; ok && !shell.boundInSameBody(chanSym, ctx.Scope) {
			shell.aliasCrossBodyRead.Add(chanSym)
			chanSym = backing
		}
		read, ok := buildChannelReadNode(chName, chanSym, kg)
		if !ok {
			return nil, false
		}
		if isVarChannel(chanSym) {
			shell.varChannels.Add(channelKey(chanSym))
		}
		reads = append(reads, read)
	}
	return reads, true
}

// flowWriteKey returns the channel key the flow's terminal sink writes to.
func flowWriteKey(ctx acontext.Context[parser.IFlowStatementContext]) (uint32, bool) {
	nodes := ctx.AST.AllFlowNode()
	if len(nodes) == 0 {
		return 0, false
	}
	id := nodes[len(nodes)-1].Identifier()
	if id == nil {
		return 0, false
	}
	sym, err := ctx.Scope.Resolve(ctx, id.IDENTIFIER().GetText())
	if err != nil || (!isVarChannel(sym) && sym.Kind != symbol.KindChannel && sym.Type.Kind != types.KindChan) {
		return 0, false
	}
	return channelKey(sym), true
}

// buildConstantNode builds a constant node emitting value of type t once per activation.
func buildConstantNode(kg *keyGenerator, suffix string, value any, t types.Type) nodeResult {
	n := ir.Node{
		Key:      kg.generate("const", suffix),
		Type:     "constant",
		Channels: types.NewChannels(),
		Inputs:   types.Params{{Name: "value", Type: t, Value: value}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: t}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam)
}

// buildActivationPulse builds a one-shot constant that fires once per activation.
func buildActivationPulse(kg *keyGenerator) nodeResult {
	return buildConstantNode(kg, "pulse", uint8(1), types.U8())
}

// filterSelfWriteTriggers drops self-triggering reads, adding a pulse if none are left.
func filterSelfWriteTriggers(
	reads []nodeResult,
	writeKey uint32,
	hasWrite bool,
	kg *keyGenerator,
) []nodeResult {
	var triggers []nodeResult
	selfWrite := false
	for _, r := range reads {
		if _, writes := r.node.Channels.Read[writeKey]; hasWrite && writes {
			selfWrite = true
			continue
		}
		triggers = append(triggers, r)
	}
	if selfWrite && len(triggers) == 0 {
		triggers = append(triggers, buildActivationPulse(kg))
	}
	return triggers
}

// buildVarSeed records a value variable's declared value and channel key so the
// runtime can pre-fill its channel with the seed before execution.
func buildVarSeed(sym *symbol.Symbol, shell *shellBuilder) ir.VarSeed {
	shell.varChannels.Add(channelKey(sym))
	return ir.VarSeed{
		Channel: channelKey(sym),
		Type:    sym.Type,
		Value:   sym.DefaultValue,
	}
}

// collectSeededVars returns every reactive value variable with a literal
// initializer. Function bodies are skipped: their locals are WASM locals.
func collectSeededVars(root *symbol.Symbol) []*symbol.Symbol {
	var out []*symbol.Symbol
	var walk func(s *symbol.Symbol)
	walk = func(s *symbol.Symbol) {
		for _, c := range s.Children() {
			switch c.Kind {
			case symbol.KindModule, symbol.KindModuleAlias, symbol.KindFunction:
				continue
			}
			if !c.Internal && isVarChannel(c) && c.DefaultValue != nil {
				out = append(out, c)
			}
			walk(c)
		}
	}
	walk(root)
	return out
}

// collectExprVars returns every reactive value variable initialized by a non-literal
// expression rather than a seed.
func collectExprVars(root *symbol.Symbol) []*symbol.Symbol {
	var out []*symbol.Symbol
	var walk func(s *symbol.Symbol)
	walk = func(s *symbol.Symbol) {
		for _, c := range s.Children() {
			switch c.Kind {
			case symbol.KindModule, symbol.KindModuleAlias, symbol.KindFunction:
				continue
			}
			if !c.Internal && c.Kind == symbol.KindVariable && isVarChannel(c) &&
				c.DefaultValue == nil {
				if lv, ok := c.AST.(parser.ILocalVariableContext); ok && lv.Expression() != nil {
					out = append(out, c)
				}
			}
			walk(c)
		}
	}
	walk(root)
	return out
}

// lowerVarInit lowers a value variable's expression initializer into a reactive flow
// that computes it and writes the variable's channel.
func lowerVarInit[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	sym *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	lv, ok := sym.AST.(parser.ILocalVariableContext)
	if !ok {
		return nil, nil, false
	}
	expr := lv.Expression()
	if expr == nil {
		return nil, nil, false
	}
	return lowerVarWrite(ctx.WithScope(sym.Parent), sym, expr, kg, shell)
}

// lowerVarWrite lowers `target = expr` into a reactive flow that computes expr and
// writes target's channel, firing a self-write once per activation instead of looping.
func lowerVarWrite[T antlr.ParserRuleContext](
	base acontext.Context[T],
	target *symbol.Symbol,
	expr parser.IExpressionContext,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	write, ok := buildChannelWriteNode(target.Name, target, kg)
	if !ok {
		return nil, nil, false
	}
	shell.varChannels.Add(channelKey(target))

	// A bare identifier copies the referenced channel's stream directly.
	if primary := parser.GetPrimaryExpression(expr); primary != nil && primary.IDENTIFIER() != nil {
		if src, err := base.Scope.Resolve(base, primary.IDENTIFIER().GetText()); err == nil &&
			(isVarChannel(src) || src.Kind == symbol.KindChannel) {
			read, rok := buildChannelReadNode(src.Name, src, kg)
			if !rok {
				return nil, nil, false
			}
			if isVarChannel(src) {
				shell.varChannels.Add(channelKey(src))
			}
			edge := ir.Edge{Source: read.output, Target: write.input, Kind: ir.EdgeKindContinuous}
			return []ir.Node{read.node, write.node}, []ir.Edge{edge}, true
		}
	}

	exprRes, ok := analyzeExpression(acontext.Child(base, expr), kg)
	if !ok {
		return nil, nil, false
	}
	reads, ok := buildExprReadTriggers(base, expr, kg, shell)
	if !ok {
		return nil, nil, false
	}
	triggers := filterSelfWriteTriggers(reads, channelKey(target), true, kg)
	var (
		nodes []ir.Node
		edges []ir.Edge
	)
	// Each referenced channel triggers the expression node.
	for _, trigger := range triggers {
		nodes = append(nodes, trigger.node)
		edges = append(edges, ir.Edge{Source: trigger.output, Target: exprRes.input, Kind: ir.EdgeKindContinuous})
	}
	nodes = append(nodes, exprRes.node, write.node)
	edges = append(edges, ir.Edge{Source: exprRes.output, Target: write.input, Kind: ir.EdgeKindContinuous})
	return nodes, edges, true
}

// lowerAssignment lowers a variable `=`: a constant write or an in-place alias rebind.
func lowerAssignment[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	assign parser.IAssignmentContext,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	if assign.CompoundOp() != nil || assign.IndexOrSlice() != nil {
		return nil, nil, true
	}
	expr := assign.Expression()
	if expr == nil {
		return nil, nil, true
	}
	target, err := ctx.Scope.Resolve(ctx, assign.IDENTIFIER().GetText())
	if err != nil {
		return nil, nil, true
	}
	switch target.VarKind {
	case symbol.VarKindConstant:
		return lowerVarWrite(ctx, target, expr, kg, shell)
	case symbol.VarKindChannelAlias:
		rebindAlias(ctx, target, expr)
		shell.aliasBindingBody[target] = ctx.Scope
		return lowerAliasRebind(ctx, target, expr, kg, shell)
	case symbol.VarKindReactive:
		return lowerReExpr(ctx, target, expr, kg, shell)
	default:
		return nil, nil, true
	}
}

// lowerReExpr records target's feeder subgraph and lowers the reassignment to a pulse
// that writes a switch channel, advancing both the sequence and target's feeder machine.
func lowerReExpr[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	target *symbol.Symbol,
	expr parser.IExpressionContext,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	feederNodes, feederEdges, ok := lowerVarWrite(ctx, target, expr, kg, shell)
	if !ok {
		return nil, nil, false
	}
	sw, stepNodes, stepEdges, ok := buildFeederSwitch(ctx, target, kg, shell)
	if !ok {
		return nil, nil, false
	}
	shell.recordReExpr(reExprFeeder{
		target:    target,
		nodes:     feederNodes,
		edges:     feederEdges,
		switchSym: sw,
	})
	return stepNodes, stepEdges, true
}

// buildFeederSwitch creates the program-local switch a reassignment writes to advance its
// feeder machine, returning the switch symbol and the pulse->write step nodes.
func buildFeederSwitch[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	target *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (*symbol.Symbol, []ir.Node, []ir.Edge, bool) {
	// The switch channel is compiler-internal: it stays out of user-facing
	// resolution and the value-variable collectors (seed, flow, reset).
	sw, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
		Kind:     symbol.KindVariable,
		VarKind:  symbol.VarKindReactive,
		Type:     types.U8(),
		Internal: true,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nil, nil, nil, false
	}
	sw.Name = fmt.Sprintf("__reexpr_switch_%s_%d", target.Name, sw.ID)
	write, ok := buildChannelWriteNode(sw.Name, sw, kg)
	if !ok {
		return nil, nil, nil, false
	}
	shell.varChannels.Add(channelKey(sw))
	pulse := buildActivationPulse(kg)
	stepEdge := ir.Edge{Source: pulse.output, Target: write.input, Kind: ir.EdgeKindContinuous}
	return sw, []ir.Node{pulse.node, write.node}, []ir.Edge{stepEdge}, true
}

// lowerAliasRebind records a reassigned alias's read and write feeders off one shared
// switch, so both its read and write machines advance to the new binding together.
func lowerAliasRebind[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	alias *symbol.Symbol,
	expr parser.IExpressionContext,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	readBacking, ok := shell.aliasBacking[alias]
	if !ok {
		return nil, nil, true
	}
	mux, ok := shell.aliasWriteBackings[alias]
	if !ok {
		return nil, nil, true
	}
	readNodes, readEdges, ok := lowerVarWrite(ctx, readBacking, expr, kg, shell)
	if !ok {
		return nil, nil, false
	}
	writeNodes, writeEdges, ok := buildAliasWriteFeeder(ctx, mux.backing, expr, kg)
	if !ok {
		return nil, nil, false
	}
	sw, stepNodes, stepEdges, ok := buildFeederSwitch(ctx, alias, kg, shell)
	if !ok {
		return nil, nil, false
	}
	shell.recordReExpr(reExprFeeder{
		target:    readBacking,
		nodes:     readNodes,
		edges:     readEdges,
		switchSym: sw,
	})
	mux.feeders = append(mux.feeders, reExprFeeder{
		target:    mux.backing,
		nodes:     writeNodes,
		edges:     writeEdges,
		switchSym: sw,
	})
	return stepNodes, stepEdges, true
}

// buildAliasWriteFeeder forwards a reassigned alias's write backing to the channel named
// by expr, the write machine's per-rebind feeder.
func buildAliasWriteFeeder[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	writeBacking *symbol.Symbol,
	expr parser.IExpressionContext,
	kg *keyGenerator,
) ([]ir.Node, []ir.Edge, bool) {
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil, nil, false
	}
	binding, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return nil, nil, false
	}
	return buildBackingForward(writeBacking, binding, kg)
}

// buildBackingForward builds read(backing) -> write(dst), the write machine's per-state forward.
func buildBackingForward(backing, dst *symbol.Symbol, kg *keyGenerator) ([]ir.Node, []ir.Edge, bool) {
	read, ok := buildChannelReadNode(backing.Name, backing, kg)
	if !ok {
		return nil, nil, false
	}
	write, ok := buildChannelWriteNode(dst.Name, dst, kg)
	if !ok {
		return nil, nil, false
	}
	edge := ir.Edge{Source: read.output, Target: write.input, Kind: ir.EdgeKindContinuous}
	return []ir.Node{read.node, write.node}, []ir.Edge{edge}, true
}

// buildReExprMachine builds target's feeder machine (init + reassignment states) and
// returns its switch-reader nodes; a fresh switch write jumps to that feeder's state.
func buildReExprMachine(
	target *symbol.Symbol,
	initNodes []ir.Node,
	feeders []reExprFeeder,
	kg *keyGenerator,
) (ir.Scope, []ir.Node, bool) {
	machineKey := kg.generate("reexpr", target.Name)
	machine := ir.Scope{
		Key:      machineKey,
		Mode:     ir.ScopeModeSequential,
		Liveness: ir.LivenessAlways,
	}
	stateNodes := make([][]ir.Node, len(feeders)+1)
	stateNodes[0] = initNodes
	for k := range feeders {
		stateNodes[k+1] = feeders[k].nodes
	}
	var readers []ir.Node
	for i := range stateNodes {
		for k, f := range slices.Backward(feeders) {
			read, ok := buildChannelReadNode(f.switchSym.Name, f.switchSym, kg)
			if !ok {
				return ir.Scope{}, nil, false
			}
			stateNodes[i] = append(stateNodes[i], read.node)
			readers = append(readers, read.node)
			targetKey := fmt.Sprintf("%s_s%d", machineKey, k+1)
			machine.Transitions = append(machine.Transitions,
				ir.Transition{On: read.output, TargetKey: new(targetKey)})
		}
	}
	for i, nodes := range stateNodes {
		state := flowScope(fmt.Sprintf("%s_s%d", machineKey, i), nodes)
		machine.Steps = append(machine.Steps, ir.Member{Scope: &state})
	}
	return machine, readers, true
}

// rebindAlias re-points target's alias binding to the channel named by expr, so later
// references resolve to it. The analyzer has validated expr names a matching channel.
func rebindAlias[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	target *symbol.Symbol,
	expr parser.IExpressionContext,
) {
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil {
		return
	}
	src, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return
	}
	id := src.ID
	target.SourceID = &id
}

// createAliasBacking adds a reactive channel that redirects a reassigned alias's reads
// or writes; role distinguishes the two so each alias gets a distinct backing.
func createAliasBacking[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	alias *symbol.Symbol,
	role string,
	shell *shellBuilder,
) (*symbol.Symbol, bool) {
	backing, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
		Kind:     symbol.KindVariable,
		VarKind:  symbol.VarKindReactive,
		Type:     alias.Type.Unwrap(),
		Internal: true,
		AST:      alias.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, alias.AST))
		return nil, false
	}
	backing.Name = fmt.Sprintf("__alias_backing_%s_%s_%d", role, alias.Name, backing.ID)
	shell.varChannels.Add(channelKey(backing))
	return backing, true
}

// resolveAliasInit resolves the channel an alias is bound to at its declaration.
func resolveAliasInit[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	alias *symbol.Symbol,
) (*symbol.Symbol, bool) {
	lv, ok := alias.AST.(parser.ILocalVariableContext)
	if !ok || lv.Expression() == nil {
		return nil, false
	}
	primary := parser.GetPrimaryExpression(lv.Expression())
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil, false
	}
	src, err := alias.Parent.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return nil, false
	}
	return src, true
}

// collectAliasBackings backs every reassigned alias before the main walk.
func collectAliasBackings[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	shell *shellBuilder,
) {
	var walk func(scope *symbol.Symbol, node antlr.Tree)
	walk = func(scope *symbol.Symbol, node antlr.Tree) {
		if assign, ok := node.(parser.IAssignmentContext); ok {
			recordAliasRebind(ctx, scope, assign, shell)
		}
		next := scope
		switch node.(type) {
		case parser.ISequenceDeclarationContext, parser.IStageDeclarationContext:
			if c, err := scope.GetChildByParserRule(node.(antlr.ParserRuleContext)); err == nil {
				next = c
			}
		}
		for i := 0; i < node.GetChildCount(); i++ {
			walk(next, node.GetChild(i))
		}
	}
	walk(ctx.Scope.Root(), ctx.AST)
}

// recordAliasRebind backs assign's target if it is an alias not already backed.
func recordAliasRebind[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	scope *symbol.Symbol,
	assign parser.IAssignmentContext,
	shell *shellBuilder,
) {
	if assign.Expression() == nil || assign.CompoundOp() != nil || assign.IndexOrSlice() != nil {
		return
	}
	id := assign.IDENTIFIER()
	if id == nil {
		return
	}
	sym, err := scope.Resolve(ctx, id.GetText())
	if err != nil || sym.VarKind != symbol.VarKindChannelAlias {
		return
	}
	if _, ok := shell.aliasBacking[sym]; ok {
		return
	}
	readBacking, ok := createAliasBacking(ctx, sym, "read", shell)
	if !ok {
		return
	}
	writeBacking, ok := createAliasBacking(ctx, sym, "write", shell)
	if !ok {
		return
	}
	initBinding, ok := resolveAliasInit(ctx, sym)
	if !ok {
		return
	}
	shell.aliasBacking[sym] = readBacking
	shell.aliasWriteBackings[sym] = &aliasWriteMux{backing: writeBacking, initBinding: initBinding}
	shell.aliasWriteOrder = append(shell.aliasWriteOrder, sym)
	shell.aliasBindingBody[sym] = sym.Parent
}

// analyzeNextToken emits a transition intent that advances the enclosing
// sequence to the next sibling member. The target member is resolved against
// the innermost sequence frame at intent-consumption time.
func analyzeNextToken(
	ctx acontext.Context[parser.IFlowNodeContext],
	shell *shellBuilder,
) (flowNodeResult, bool) {
	if shell.inlineBoundaryBlocksNext() {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
			"'next' is not valid inside an inline routing case body"))
		return flowNodeResult{}, false
	}
	frame := shell.top()
	if frame == nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST, "'next' used outside of a sequence"))
		return flowNodeResult{}, false
	}
	if frame.nextMember() == "" {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"'next' in last stage '%s' has no next stage",
			frame.memberKeys[frame.activeIdx],
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
	head, tail := parser.FunctionNameParts(ctx.AST)
	name := head
	if tail != "" {
		name = head + "." + tail
	}
	key := kg.generate(name, "")
	sym, err := ctx.ResolveQualified(head, tail)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}
	if sym.Exec == symbol.ExecWASM {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"function '%s' cannot be used as a flow statement. Call it inside a func block instead: %s()",
			name, name,
		))
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
	// Node.Type is the canonical module path so factories find it. After
	// alias indirection in Resolve, sym's tree position already points at
	// the canonical module member; QualifiedName joins the parts.
	nodeType := sym.QualifiedName()
	freshType := types.Freshen(sym.Type, key)
	n := ir.Node{
		Key:      key,
		Type:     nodeType,
		Channels: sym.Channels.Copy(),
		Inputs:   slices.Clone(freshType.Inputs),
		Outputs:  slices.Clone(freshType.Outputs),
	}
	var ok bool
	n.Inputs, ok = extractInputValues(acontext.Child(ctx, ctx.AST.InputValues()), n.Inputs, n, sym)
	if !ok {
		return nodeResult{}, false
	}
	inputParam := ir.DefaultInputParam
	if sym.Trigger.Target != "" {
		inputParam = sym.Trigger.Target
	}
	return newNodeResult(n, inputParam, firstOutputParam(n.Outputs)), true
}

// tryAnalyzeFmtStrLiteral handles the format-string-with-placeholders case by
// emitting a synthetic function. handled=true means the literal was a format
// string with placeholders and the result is authoritative; handled=false
// means callers should fall through to other literal handling.
func tryAnalyzeFmtStrLiteral(
	ctx acontext.Context[parser.IExpressionContext],
	sym *symbol.Symbol,
	kg *keyGenerator,
) (nodeResult, bool, bool) {
	literalCtx := parser.GetLiteral(ctx.AST)
	if literalCtx == nil {
		return nodeResult{}, false, false
	}
	strTerm := parser.StringTerminal(literalCtx)
	if strTerm == nil {
		return nodeResult{}, false, false
	}
	_, flags, ok := literal.StripQuotes(strTerm.GetText())
	if !ok || !flags.Format {
		return nodeResult{}, false, false
	}
	outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
	parsedValue, err := literal.Parse(literalCtx, outputType)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, true, false
	}
	body := parsedValue.Value.(string)
	segments, err := literal.FmtStrParse(body)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, true, false
	}
	if !literal.FmtStrHasPlaceholder(segments) {
		return nodeResult{}, false, false
	}
	key := kg.generate("fmt", "")
	synthKey := compiler.FmtStrSyntheticPrefix + key
	sym.Name = synthKey
	*kg.synthFuncs = append(*kg.synthFuncs, ir.Function{
		Key:      synthKey,
		Body:     ir.Body{Raw: body},
		Inputs:   types.Params{},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
		Channels: sym.Channels.Copy(),
	})
	n := ir.Node{
		Key:      key,
		Type:     synthKey,
		Channels: sym.Channels.Copy(),
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true, true
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

	if sym.Kind == symbol.KindFunction && parser.IsLiteral(ctx.AST) {
		if n, handled, ok := tryAnalyzeFmtStrLiteral(ctx, sym, kg); handled {
			return n, ok
		}
	}

	if sym.Kind == symbol.KindConstant {
		outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
		parsedValue, err := literal.ParseConst(ctx.AST, outputType)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return nodeResult{}, false
		}
		return buildConstantNode(kg, "", parsedValue.Value, outputType), true
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
//
// The root parameter is the pre-built program root. Callers construct it
// (typically via stl.NewRoot) so the text package never needs to know
// which built-ins are loaded or where external symbols come from.
func Analyze(
	ctx context.Context,
	t Text,
	root *symbol.Symbol,
	cfgs ...parser.Config,
) (ir.IR, *diagnostics.Diagnostics) {
	var (
		aCtx = acontext.NewRoot(ctx, t.AST, root).WithConfig(parser.ConfigOf(cfgs...))
		i    = ir.IR{Symbols: aCtx.Scope, TypeMap: aCtx.TypeMap}
	)

	analyzer.AnalyzeProgram(aCtx)
	i.Authorities = authority.Analyze(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return i, aCtx.Diagnostics
	}

	for _, c := range i.Symbols.Children() {
		if c.Kind != symbol.KindFunction || c.AST == nil {
			continue
		}
		if expr, ok := c.AST.(parser.IExpressionContext); ok && parser.IsLiteral(expr) {
			continue
		}
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
			Inputs:   c.Type.Inputs,
			Outputs:  c.Type.Outputs,
			Channels: c.Channels,
		})
	}
	kg := newKeyGenerator(&i.Functions)
	shell := newShellBuilder(collectSynthByAST(aCtx.Scope.Root()))
	// Back reassigned aliases before lowering so reads compiled ahead of a rebind resolve.
	collectAliasBackings(aCtx, shell)

	// The root scope is always parallel and always-live.
	i.Root = ir.Scope{
		Mode:          ir.ScopeModeParallel,
		Liveness:      ir.LivenessAlways,
		ResetChannels: scopeResetChannels(aCtx.Scope.Root()),
	}
	// rootMembers accumulates every top-level item as a Member of the root
	// scope. Module-scope flow nodes become leaf-node members; top-level
	// sequence and stage declarations become nested Scope members.
	var rootMembers ir.Members

	for _, item := range t.AST.AllTopLevelItem() {
		if flow := item.FlowStatement(); flow != nil {
			nodes, edges, inlineMembers, _, ok := analyzeFlow(acontext.Child(aCtx, flow), kg, shell)
			if !ok {
				return i, aCtx.Diagnostics
			}
			for _, n := range nodes {
				rootMembers = append(rootMembers, ir.Member{NodeKey: new(n.Key)})
			}
			rootMembers = append(rootMembers, inlineMembers...)
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

	// Seed each literal-initialized value variable so a read before any write
	// still observes its declared value. Seeding is applied at runtime setup.
	for _, v := range collectSeededVars(aCtx.Scope.Root()) {
		i.VarSeeds = append(i.VarSeeds, buildVarSeed(v, shell))
	}

	// Lower each expression-initialized value variable into a reactive flow. A
	// re-expressed variable is assembled into a feeder machine below instead.
	for _, v := range collectExprVars(aCtx.Scope.Root()) {
		if shell.isReExpressed(v) {
			continue
		}
		nodes, edges, ok := lowerVarInit(aCtx, v, kg, shell)
		if !ok {
			return i, aCtx.Diagnostics
		}
		for _, n := range nodes {
			rootMembers = append(rootMembers, ir.Member{NodeKey: new(n.Key)})
		}
		i.Nodes = append(i.Nodes, nodes...)
		i.Edges = append(i.Edges, edges...)
	}

	// Assemble each re-expressed reactive variable's feeder state machine.
	aliasByReadBacking := make(map[*symbol.Symbol]*symbol.Symbol, len(shell.aliasBacking))
	for alias, backing := range shell.aliasBacking {
		aliasByReadBacking[backing] = alias
	}
	for _, target := range shell.reExprTargets() {
		if alias, ok := aliasByReadBacking[target]; ok && !shell.aliasCrossBodyRead.Contains(alias) {
			continue
		}
		initNodes, initEdges, ok := lowerVarInit(aCtx, target, kg, shell)
		if !ok {
			return i, aCtx.Diagnostics
		}
		feeders := shell.reExprsFor(target)
		machine, readers, ok := buildReExprMachine(target, initNodes, feeders, kg)
		if !ok {
			return i, aCtx.Diagnostics
		}
		i.Nodes = append(i.Nodes, initNodes...)
		i.Edges = append(i.Edges, initEdges...)
		for _, f := range feeders {
			i.Nodes = append(i.Nodes, f.nodes...)
			i.Edges = append(i.Edges, f.edges...)
		}
		i.Nodes = append(i.Nodes, readers...)
		rootMembers = append(rootMembers, ir.Member{Scope: &machine})
	}

	// Assemble each reassigned alias's write machine, forwarding its write backing to the
	// binding current at runtime; the mirror of the read machine above.
	for _, alias := range shell.aliasWriteOrder {
		// An alias whose writes all baked needs no machine.
		if !shell.aliasCrossBodyWrite.Contains(alias) {
			continue
		}
		mux := shell.aliasWriteBackings[alias]
		initNodes, initEdges, ok := buildBackingForward(mux.backing, mux.initBinding, kg)
		if !ok {
			return i, aCtx.Diagnostics
		}
		machine, readers, ok := buildReExprMachine(mux.backing, initNodes, mux.feeders, kg)
		if !ok {
			return i, aCtx.Diagnostics
		}
		i.Nodes = append(i.Nodes, initNodes...)
		i.Edges = append(i.Edges, initEdges...)
		for _, f := range mux.feeders {
			i.Nodes = append(i.Nodes, f.nodes...)
			i.Edges = append(i.Edges, f.edges...)
		}
		i.Nodes = append(i.Nodes, readers...)
		rootMembers = append(rootMembers, ir.Member{Scope: &machine})
	}

	// Inline bodies live as members of their enclosing scope; their flat IR
	// still registers in the program's global node and edge lists.
	i.Nodes = append(i.Nodes, shell.inlineNodes...)
	i.Edges = append(i.Edges, shell.inlineEdges...)

	i.VarChannels = shell.varChannels.Slice()
	slices.Sort(i.VarChannels)

	if len(rootMembers) > 0 {
		i.Root.Strata = []ir.Members{rootMembers}
	}

	// Stamp each deferred activation (`trigger => main`, or an inline body gated
	// by its upstream handle) onto the matching scope member wherever it lives.
	bound := set.New[string]()
	var bindActivations func(s *ir.Scope)
	bindActivations = func(s *ir.Scope) {
		visit := func(m *ir.Member) {
			if m.Scope == nil {
				return
			}
			if handle, ok := shell.activations[m.Scope.Key]; ok {
				m.Scope.Activation = new(handle)
				bound.Add(m.Scope.Key)
			}
			bindActivations(m.Scope)
		}
		for si := range s.Strata {
			for mi := range s.Strata[si] {
				visit(&s.Strata[si][mi])
			}
		}
		for mi := range s.Steps {
			visit(&s.Steps[mi])
		}
	}
	if len(shell.activations) > 0 {
		bindActivations(&i.Root)
	}
	// Safety net: analyzeNamedRef should reject any `=> X` whose X is
	// neither an enclosing-sequence member nor a top-level scope, so every
	// registered activation should bind. If one does not, something upstream
	// registered an activation without going through analyzeNamedRef.
	for key := range shell.activations {
		if !bound.Contains(key) {
			aCtx.Diagnostics.Add(diagnostics.Errorf(
				t.AST,
				"internal: activation target '%s' did not bind to any "+
					"scope; this should have been rejected by analyzeNamedRef",
				key,
			))
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
	// inlineMembers collects lowered inline-body scopes for placement as members
	// of the scope enclosing this flow.
	inlineMembers []ir.Member
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
	reads, ok := buildExprReadTriggers(p.ctx, expr, p.kg, p.shell)
	if !ok {
		return false
	}
	writeKey, hasWrite := flowWriteKey(p.ctx)
	triggers := filterSelfWriteTriggers(reads, writeKey, hasWrite, p.kg)
	for _, result := range triggers {
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

	if result.inlineScope != nil {
		p.inlineMembers = append(p.inlineMembers, ir.Member{Scope: result.inlineScope})
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
	// When a multi-channel expression drives a transition there is no IR
	// node to route the extra triggers into; the primary channel output
	// already carries the firing signal, so the extras are dropped.
	p.additionalTriggers = nil

	p.shell.applyTransitionIntent(p.prevOutput, intent)
	p.transitionEmitted = true
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
	newNodes, newEdges, inlineMembers, ok := analyzeOutputRoutingTable(
		acontext.Child(p.ctx, rt),
		*p.prevNode,
		p.kg,
		p.shell,
	)
	if !ok {
		return false
	}
	p.nodes = append(p.nodes, newNodes...)
	p.edges = append(p.edges, newEdges...)
	p.inlineMembers = append(p.inlineMembers, inlineMembers...)
	p.prevNode = nil
	// Routing entries dispatch via their own transitions/activations; suppress
	// the enclosing sequence's auto-advance so it does not double-fire.
	p.transitionEmitted = true
	return true
}

// analyzeFlow processes a single flow statement. In addition to the IR nodes
// and edges it produced, it reports whether the flow chain terminated in an
// explicit transition target (`=> next`, `=> name`, or a scope-valued
// identifier). Callers that auto-wire a terminal transition for the flow
// step should suppress that auto-wire when transitionEmitted is true, since
// the explicit transition already covers (and may target a different frame
// than) the auto-wire would.
func analyzeFlow(
	ctx acontext.Context[parser.IFlowStatementContext],
	kg *keyGenerator,
	shell *shellBuilder,
) (nodes []ir.Node, edges []ir.Edge, inlineMembers []ir.Member, transitionEmitted bool, ok bool) {
	p := newFlowChainProcessor(ctx, kg, shell)
	for i, child := range ctx.AST.GetChildren() {
		switch c := child.(type) {
		case parser.IFlowNodeContext:
			if !p.processFlowNode(c) {
				return nil, nil, nil, false, false
			}
		case parser.IFlowOperatorContext:
			p.lastOpIndex = i
		case parser.IRoutingTableContext:
			if !p.processRoutingTable(c) {
				return nil, nil, nil, false, false
			}
		}
	}
	if len(p.edges) < 1 && !p.transitionEmitted {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"flow statement requires at least two nodes",
		))
		return nil, nil, nil, false, false
	}
	return p.nodes, p.edges, p.inlineMembers, p.transitionEmitted, true
}

func extractInputValues(
	ctx acontext.Context[parser.IInputValuesContext],
	input types.Params,
	node ir.Node,
	fnSym *symbol.Symbol,
) (types.Params, bool) {
	if ctx.AST == nil {
		return input, true
	}

	parseInputExpr := func(
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
			symbol.ResolveInputChannel(&node.Channels, fnSym, paramName, channelKey, sym.Name)
			return channelKey, true
		}

		if primary := parser.GetPrimaryExpression(expr); primary != nil {
			if id := primary.IDENTIFIER(); id != nil {
				sym, err := ctx.Scope.Resolve(ctx, id.GetText())
				if err != nil {
					ctx.Diagnostics.Add(diagnostics.Error(err, expr))
					return nil, false
				}
				isValueVar := sym.Kind == symbol.KindVariable ||
					sym.Kind == symbol.KindStatefulVariable
				if isValueVar && sym.DefaultValue != nil {
					return sym.DefaultValue, true
				}
			}
		}

		if !parser.IsLiteral(expr) {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				expr,
				"input value for '%s' must be a literal or variable",
				paramName,
			))
			return nil, false
		}

		parsedValue, err := literal.ParseConst(expr, paramType)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, expr))
			return nil, false
		}
		return parsedValue.Value, true
	}

	if named := ctx.AST.NamedInputValues(); named != nil {
		for _, cv := range named.AllNamedInputValue() {
			key := cv.IDENTIFIER().GetText()
			idx := input.GetIndex(key)
			if expr := cv.Expression(); expr != nil {
				value, ok := parseInputExpr(expr, input[idx].Type, key)
				if !ok {
					return nil, false
				}
				input[idx].Value = value
			}
		}
	} else if anon := ctx.AST.AnonymousInputValues(); anon != nil {
		exprs := anon.AllExpression()
		pos := 0
		for i := range input {
			if input[i].Name == fnSym.Trigger.Target {
				continue
			}
			if pos >= len(exprs) {
				break
			}
			value, ok := parseInputExpr(exprs[pos], input[i].Type, fmt.Sprintf("position %d", pos))
			if !ok {
				return nil, false
			}
			input[i].Value = value
			pos++
		}
	}

	return input, true
}

// collectSynthByAST returns a map from each inline-body declaration in the tree
// rooted at root to its synth scope, keyed by the declaration's parser node.
func collectSynthByAST(root *symbol.Symbol) map[antlr.ParserRuleContext]*symbol.Symbol {
	m := map[antlr.ParserRuleContext]*symbol.Symbol{}
	var walk func(s *symbol.Symbol)
	walk = func(s *symbol.Symbol) {
		for _, child := range s.Children() {
			if child.AST != nil && strings.HasPrefix(child.Name, ir.InlinePrefix) {
				m[child.AST] = child
			}
			walk(child)
		}
	}
	walk(root)
	return m
}

// processInlineBody lowers an inline stage/sequence body to IR with the current
// shell stack live, returning the gated scope for the caller to place in its
// lexically enclosing scope.
func processInlineBody(
	ctx acontext.Context[parser.IFlowNodeContext],
	synth *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, bool) {
	shell.inlineBodyBases = append(shell.inlineBodyBases, len(shell.stack))
	defer func() {
		shell.inlineBodyBases = shell.inlineBodyBases[:len(shell.inlineBodyBases)-1]
	}()
	var (
		scope ir.Scope
		nodes []ir.Node
		edges []ir.Edge
		ok    bool
	)
	switch decl := synth.AST.(type) {
	case parser.IStageDeclarationContext:
		scope, nodes, edges, ok = analyzeTopLevelStage(
			acontext.Child(ctx, decl).WithScope(synth.Parent), kg, shell)
	case parser.ISequenceDeclarationContext:
		scope, nodes, edges, ok = analyzeSequence(
			acontext.Child(ctx, decl).WithScope(synth.Parent), kg, shell)
	}
	if !ok {
		return ir.Scope{}, false
	}
	scope.Liveness = ir.LivenessGated
	shell.inlineNodes = append(shell.inlineNodes, nodes...)
	shell.inlineEdges = append(shell.inlineEdges, edges...)
	return scope, true
}

func analyzeOutputRoutingTable(
	ctx acontext.Context[parser.IRoutingTableContext],
	sourceNode ir.Node,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, []ir.Member, bool) {
	var (
		nodes         []ir.Node
		edges         []ir.Edge
		inlineMembers []ir.Member
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
			return nil, nil, nil, false
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
				return nil, nil, nil, false
			}

			if result.inlineScope != nil {
				inlineMembers = append(inlineMembers, ir.Member{Scope: result.inlineScope})
			}

			if result.transition != nil {
				shell.applyTransitionIntent(prevOutputHandle, *result.transition)
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
					return nil, nil, nil, false
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

	return nodes, edges, inlineMembers, true
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
		// Variable declarations are ambient (seeded or reactive), not sequential
		// steps; a reassignment, like a flow write, is a step.
		if item.VariableDeclaration() != nil {
			continue
		}
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

// addInlineMembers appends inline-body scope members to scope's stratum 0,
// creating the stratum when the scope has none yet.
func addInlineMembers(scope *ir.Scope, members []ir.Member) {
	if len(members) == 0 {
		return
	}
	if len(scope.Strata) == 0 {
		scope.Strata = []ir.Members{members}
		return
	}
	scope.Strata[0] = append(scope.Strata[0], members...)
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
	seqScope, err := acontext.ResolveOwnScope(ctx)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return ir.Scope{}, nil, nil, false
	}
	seqName := seqScope.Name
	liveness := ir.LivenessAlways
	if ctx.AST.IDENTIFIER() != nil {
		liveness = ir.LivenessGated
	}
	scope := ir.Scope{
		Key:           seqName,
		Mode:          ir.ScopeModeSequential,
		Liveness:      liveness,
		ResetChannels: scopeResetChannels(seqScope),
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
			nodes, edges, inlineMembers, transitionEmitted, ok := analyzeFlow(
				acontext.Child(ctx, flowStmt).WithScope(seqScope),
				kg,
				shell,
			)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			child := flowScope(si.key, nodes)
			addInlineMembers(&child, inlineMembers)
			scope.Steps = append(scope.Steps, ir.Member{Scope: &child})
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
			// Only auto-wire the step-advance transition when the flow did
			// not already terminate in an explicit `=> X`. The explicit
			// transition may target an outer frame; emitting the auto-wire
			// alongside it causes the inner frame's evaluator to clear the
			// firing-node's mark before the outer frame's evaluator sees
			// it, preventing the cross-level jump from firing.
			if !transitionEmitted && len(nodes) > 0 {
				autoWireTransition(shell, nodes[len(nodes)-1], nextKey)
			}
			continue
		}

		if assign := item.Assignment(); assign != nil {
			nodes, edges, ok := lowerAssignment(
				acontext.Child(ctx, assign).WithScope(seqScope),
				assign,
				kg,
				shell,
			)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			// An alias rebind produces no IR, but a sequence step still needs a
			// firing node to advance; clock it once with an activation pulse.
			if len(nodes) == 0 {
				nodes = []ir.Node{buildActivationPulse(kg).node}
			}
			child := flowScope(si.key, nodes)
			scope.Steps = append(scope.Steps, ir.Member{Scope: &child})
			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
			autoWireTransition(shell, nodes[len(nodes)-1], nextKey)
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
			// Anonymous inline nested sequences must use the outer's
			// synthesized step key so autoWireTransition from the preceding
			// flow step can address them; analyzeSequence otherwise stamps
			// the scope.Key with an AutoName (seq_N) that does not match
			// collectStepKeys' step_N. Named nested sequences keep their
			// source-level name.
			if nestedSeqDecl.IDENTIFIER() == nil {
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
	stageSym, err := acontext.ResolveOwnScope(ctx)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return ir.Scope{}, nil, nil, false
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
	liveness := ir.LivenessAlways
	if id := ctx.AST.IDENTIFIER(); id != nil {
		stageName = id.GetText()
		liveness = ir.LivenessGated
	}
	scope := ir.Scope{
		Key:      stageName,
		Mode:     ir.ScopeModeParallel,
		Liveness: liveness,
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

	stageCtx := ctx
	if stageScope, err := acontext.ResolveOwnScope(ctx); err == nil {
		stageCtx = ctx.WithScope(stageScope)
		scope.ResetChannels = scopeResetChannels(stageScope)
	}

	for _, item := range stageBody.AllStageItem() {
		if flowStmt := item.FlowStatement(); flowStmt != nil {
			itemNodes, itemEdges, inlineMembers, _, ok := analyzeFlow(
				acontext.Child(stageCtx, flowStmt),
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
			members = append(members, inlineMembers...)
			continue
		}
		if single := item.SingleInvocation(); single != nil {
			node, ok := analyzeSingleInvocation(acontext.Child(stageCtx, single), kg)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			nodes = append(nodes, node)
			members = append(members, ir.Member{NodeKey: new(node.Key)})
			continue
		}
		if nestedSeqDecl := item.SequenceDeclaration(); nestedSeqDecl != nil {
			subScope, subNodes, subEdges, ok := analyzeSequence(
				acontext.Child(stageCtx, nestedSeqDecl), kg, shell)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			nodes = append(nodes, subNodes...)
			edges = append(edges, subEdges...)
			members = append(members, ir.Member{Scope: &subScope})
			continue
		}
		if assign := item.Assignment(); assign != nil {
			itemNodes, itemEdges, ok := lowerAssignment(stageCtx, assign, kg, shell)
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
