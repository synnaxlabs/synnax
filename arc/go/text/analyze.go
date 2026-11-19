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
	"maps"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/authority"
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/analyzer/flow"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// keyGenerator produces globally unique IR node keys. It maintains a running
// per-role counter so that successive channel reads, writes, or function
// invocations with the same logical name receive distinct keys.
type keyGenerator struct {
	occurrences map[string]int
	synthFuncs  *ir.Functions
	// prefix is the import path of the module being lowered, empty for an
	// automation. It keeps a module's keys apart from its importer's.
	prefix string
}

func newKeyGenerator(synthFuncs *ir.Functions, prefix string) *keyGenerator {
	return &keyGenerator{
		occurrences: make(map[string]int),
		synthFuncs:  synthFuncs,
		prefix:      prefix,
	}
}

func (kg *keyGenerator) generate(role, name string) string {
	base := role
	if name != "" {
		base = role + "_" + name
	}
	count := kg.occurrences[base]
	kg.occurrences[base]++
	key := fmt.Sprintf("%s_%d", base, count)
	if kg.prefix != "" {
		return kg.prefix + "." + key
	}
	return key
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
	// escalatesCompletion hands the sequence's completion to enclosing frames
	// instead of exiting here; set when the sequence is a step of its parent.
	escalatesCompletion bool
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
	activations map[string][]ir.Activation
	// bodyNodes accumulates the flat IR of every lowered anonymous body and instance.
	bodyNodes []ir.Node
	// instanceKey is the scope key of the instance being lowered, empty outside one.
	instanceKey string
	// args holds the constant or the channel of each parameter of the instance.
	args map[*symbol.Symbol]argument
	// exits holds the transitions of the stage being lowered when it is not a step,
	// so `next` ends that stage. Nil in a step, where `next` completes the step.
	exits *[]ir.Transition
	// synthByAST maps each anonymous-body declaration to its synth scope, keyed by
	// the declaration's parser node.
	synthByAST map[antlr.ParserRuleContext]*symbol.Symbol
	// varNodes maps a variable's symbol to its runtime nodes.
	varNodes map[*symbol.Symbol]varEntry
	// rootNodes hold always-live nodes: registers, derefs, and dispatchers.
	rootNodes []*ir.Node
	// edges accumulates every walk-emitted edge, flushed into the program.
	edges []ir.Edge
}

// varEntry holds a variable's runtime nodes: every reassignment writes the
// register; deref serves reads when the register holds an indirection.
type varEntry struct {
	init     *ir.Node
	register *ir.Node
	deref    *ir.Node
	disp     *dispatcher
}

// argument is what a call binds to a parameter.
type argument struct {
	// value is the constant of a value parameter.
	value any
	// variable is the node that holds a value parameter that can change.
	variable *ir.Node
	// channel is the key of the channel bound to a channel parameter.
	channel uint32
}

// dispatcher pairs a variable's mux node with its ordered branch function keys.
type dispatcher struct {
	node     *ir.Node
	branches []string
}

// readNode returns the node a read of the variable references: the deref for
// an exprRead, the register for a const; an alias reads through channel nodes.
func (e varEntry) readNode(sym *symbol.Symbol) *ir.Node {
	if e.deref != nil {
		return e.deref
	}
	if sym.Type.Kind != types.KindChan {
		return e.register
	}
	return nil
}

func newShellBuilder(
	synthByAST map[antlr.ParserRuleContext]*symbol.Symbol,
) *shellBuilder {
	return &shellBuilder{
		activations: map[string][]ir.Activation{},
		synthByAST:  synthByAST,
		varNodes:    map[*symbol.Symbol]varEntry{},
	}
}

// registerVar records sym's nodes for the end-of-analysis flush.
func (sb *shellBuilder) registerVar(sym *symbol.Symbol, e varEntry) {
	sb.varNodes[sym] = e
	if e.register != nil {
		sb.rootNodes = append(sb.rootNodes, e.register)
	}
	if e.deref != nil {
		sb.rootNodes = append(sb.rootNodes, e.deref)
	}
	if e.disp != nil {
		sb.rootNodes = append(sb.rootNodes, e.disp.node)
	}
}

// enter starts the lowering of an instance and returns the function that ends it.
func (s *shellBuilder) enter(key string, args map[*symbol.Symbol]argument) func() {
	outerKey, outerArgs, outerVars := s.instanceKey, s.args, s.varNodes
	// The body reads the variables declared outside it through the copy.
	s.instanceKey, s.varNodes = key, maps.Clone(s.varNodes)
	s.args = make(map[*symbol.Symbol]argument, len(args))
	for param, arg := range args {
		if arg.variable == nil {
			s.args[param] = arg
			continue
		}
		// The body reads a parameter that can change as it reads a variable.
		s.varNodes[param] = varEntry{register: arg.variable}
	}
	return func() { s.instanceKey, s.args, s.varNodes = outerKey, outerArgs, outerVars }
}

// scopeKey returns the key of the anonymous body synth, unique for each instance.
func (s *shellBuilder) scopeKey(synth *symbol.Symbol) string {
	if s.instanceKey == "" {
		return synth.QualifiedName()
	}
	return s.instanceKey + "." + synth.Name
}

// channelKey returns the key of the channel that sym names, aliases, or is bound to.
func (s *shellBuilder) channelKey(sym *symbol.Symbol) uint32 {
	if arg, ok := s.args[sym]; ok {
		return arg.channel
	}
	if sym.SourceID != nil {
		return uint32(*sym.SourceID)
	}
	return uint32(sym.ID)
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

// addCompletion wires a terminal step's completion with the given kind to the nearest
// frame that can take it.
func (s *shellBuilder) addCompletion(on ir.Handle, kind ir.EdgeKind) {
	for i := len(s.stack) - 1; ; i-- {
		f := s.stack[i]
		if next := f.nextMember(); next != "" {
			s.addTransitionTo(f, ir.Transition{
				On:     on,
				Kind:   kind,
				Target: ir.Target{Variant: ir.StepTarget{Key: next}},
			})
			return
		}
		if !f.escalatesCompletion || i == 0 {
			s.addTransitionTo(f, ir.Transition{
				On:     on,
				Kind:   kind,
				Target: ir.Target{Variant: ir.ExitTarget{}},
			})
			return
		}
	}
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
func (s *shellBuilder) registerActivation(key string, on ir.Handle, kind ir.EdgeKind) {
	s.activations[key] = append(s.activations[key], ir.Activation{On: on, Kind: kind})
}

// ownExits makes exits the destination of `next` and returns the function that
// restores the destination before it.
func (s *shellBuilder) ownExits(exits *[]ir.Transition) func() {
	outer := s.exits
	s.exits = exits
	return func() { s.exits = outer }
}

// applyTransitionIntent records a transition or an activation against the shell
// for a firing handle. Exactly one of isNext, memberKey, activateKey is honored,
// in that priority; a zero intent is a no-op.
func (s *shellBuilder) applyTransitionIntent(
	on ir.Handle,
	kind ir.EdgeKind,
	intent transitionIntent,
) {
	switch {
	case intent.isNext:
		if s.exits == nil {
			s.addCompletion(on, kind)
			return
		}
		*s.exits = append(*s.exits, ir.Transition{
			On:     on,
			Kind:   kind,
			Target: ir.Target{Variant: ir.ExitTarget{}},
		})
	case intent.memberKey != "":
		frame := intent.targetFrame
		if frame == nil {
			frame = s.top()
		}
		s.addTransitionTo(
			frame,
			ir.Transition{
				On:     on,
				Kind:   kind,
				Target: ir.Target{Variant: ir.StepTarget{Key: intent.memberKey}},
			},
		)
	case intent.activateKey != "":
		s.registerActivation(intent.activateKey, on, kind)
	}
}

// nodeResult describes an IR node produced by a flow-node analysis. args are
// the computed-input nodes the call samples; they fire with the call's trigger.
type nodeResult struct {
	node   ir.Node
	input  ir.Handle
	output ir.Handle
	args   []ir.Node
}

func newNodeResult(node ir.Node, inputParam, outputParam string) nodeResult {
	return nodeResult{
		node:   node,
		input:  ir.Handle{Node: node.Key, Param: inputParam},
		output: ir.Handle{Node: node.Key, Param: outputParam},
	}
}

// transitionIntent is emitted by `=> next`, `=> step_name`, and a flow that starts a
// stage or sequence. The flow-chain processor consumes the intent and, rather than
// emitting a dataflow edge, records a Transition on the enclosing sequence or
// registers an activation on the scope that the flow starts.
type transitionIntent struct {
	// isNext is true when the intent came from the `next` token. It ends the stage
	// that holds it, or completes the step that holds it.
	isNext bool
	// memberKey, when non-empty and isNext is false, names a sibling member
	// to transition to. The transition lives on targetFrame (or shell.top()
	// when targetFrame is nil — the same-level fallback).
	memberKey string
	// targetFrame is the frame that owns memberKey. When it is further up the
	// stack than shell.top(), the transition fires on an enclosing sequence, and
	// deactivating the step stops every scope inside it.
	targetFrame *seqFrame
	// activateKey, when non-empty, names the scope the firing handle activates.
	activateKey string
}

// flowNodeResult is what analyzeFlowNode returns: either an actual IR node
// (the usual case) or a transition intent (for `=> next` / `=> scope_name`
// targets). Exactly one of node.node.Key or transition is non-zero.
type flowNodeResult struct {
	node       nodeResult
	transition *transitionIntent
	// bodyScope is the stage or sequence that the flow starts, to be placed as a
	// member of the scope enclosing this flow.
	bodyScope *ir.Scope
}

func firstOutputParam(outputs types.Params) string {
	if len(outputs) > 0 {
		return outputs[0].Name
	}
	return ir.DefaultOutputParam
}

// identifierAST is the common shape of sequence and stage declarations: both
// carry an optional IDENTIFIER token that is absent for anonymous
// blocks.
type identifierAST interface {
	antlr.ParserRuleContext
	IDENTIFIER() antlr.TerminalNode
}

// resolveDeclScope returns the symbol scope for a named or anonymous
// declaration. Named declarations resolve by identifier; anonymous ones are
// looked up via their parser rule (registered during the collect pass under a
// synthesized AutoName key).
func resolveDeclScope[T identifierAST](ctx acontext.Context[T]) (*symbol.Symbol, bool) {
	var (
		scope *symbol.Symbol
		err   error
	)
	if id := ctx.AST.IDENTIFIER(); id != nil {
		scope, err = ctx.Scope.Resolve(ctx, id.GetText())
	} else {
		scope, err = ctx.Scope.GetChildByParserRule(ctx.AST)
	}
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nil, false
	}
	return scope, true
}

func analyzeFlowNode(
	ctx acontext.Context[parser.IFlowNodeContext],
	kg *keyGenerator,
	shell *shellBuilder,
	isFirst, isSink bool,
) (flowNodeResult, bool) {
	if id := ctx.AST.Identifier(); id != nil {
		if body := flow.UsedBody(ctx, id); body != nil {
			return startBody(ctx.Child(id), body, kg, shell)
		}
		return analyzeIdentifierByRole(
			ctx.Child(id),
			kg,
			shell,
			isFirst,
			isSink,
		)
	}
	if fn := ctx.AST.Function(); fn != nil {
		if body := flow.UsedBody(ctx, fn); body != nil {
			return startBody(ctx.Child(fn), body, kg, shell)
		}
		r, ok := analyzeFunctionNode(ctx.Child(fn), kg, shell, !isFirst)
		return flowNodeResult{node: r}, ok
	}
	if expr := ctx.AST.Expression(); expr != nil {
		r, ok := analyzeExpression(ctx.Child(expr), kg, shell, !isFirst)
		return flowNodeResult{node: r}, ok
	}
	if ctx.AST.NEXT() != nil {
		return analyzeNextToken(ctx, shell)
	}
	if decl := declarationOf(ctx.AST); decl != nil {
		synth := shell.synthByAST[decl]
		if synth == nil {
			ctx.Diagnostics.Add(diagnostics.Errorf(decl,
				"internal: synth scope not registered for anonymous body"))
			return flowNodeResult{}, false
		}
		return startBody(ctx.Child(decl), synth, kg, shell)
	}
	return flowNodeResult{}, true
}

// declarationOf returns the stage or sequence that node declares, or nil.
func declarationOf(node interface {
	StageDeclaration() parser.IStageDeclarationContext
	SequenceDeclaration() parser.ISequenceDeclarationContext
},
) identifierAST {
	if stage := node.StageDeclaration(); stage != nil {
		return stage
	}
	if sequence := node.SequenceDeclaration(); sequence != nil {
		return sequence
	}
	return nil
}

// startBody lowers the stage or sequence that the flow node at ctx starts, and gates
// it on the flow.
func startBody[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	body *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (flowNodeResult, bool) {
	scope, ok := lowerUse(ctx, body, kg, shell)
	if !ok {
		return flowNodeResult{}, false
	}
	scope.Liveness = ir.LivenessGated
	return flowNodeResult{
		transition: &transitionIntent{activateKey: scope.Key},
		bodyScope:  &scope,
	}, true
}

// lowerUse lowers body at the place of ctx. An anonymous body lowers in place. A named
// body lowers to a copy, with the arguments of a call at ctx bound.
func lowerUse[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	body *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, bool) {
	if !body.IsTemplate() {
		scope, nodes, edges, ok := lowerBody(ctx, body, kg, shell)
		scope.Key = shell.scopeKey(body)
		shell.bodyNodes = append(shell.bodyNodes, nodes...)
		shell.edges = append(shell.edges, edges...)
		return scope, ok
	}
	if seq, ok := body.AST.(parser.ISequenceDeclarationContext); ok &&
		len(parser.BodyItems(seq.Body())) == 0 {
		ctx.Diagnostics.Add(codes.EmptySequence.Atf(
			ctx.AST, "sequence '%s' has no steps", body.Name,
		))
		return ir.Scope{}, false
	}
	var args map[*symbol.Symbol]argument
	if call, ok := any(ctx.AST).(parser.IFunctionContext); ok {
		if args, ok = bindArgs(ctx.Child(call), body, kg, shell); !ok {
			return ir.Scope{}, false
		}
	}
	// A declared name has no "$", so the key cannot match one.
	key := kg.generate("$"+body.QualifiedName(), "")
	return instantiate(ctx, body, args, key, kg, shell)
}

// bindArgs returns the argument that the call binds to each parameter of body.
func bindArgs(
	ctx acontext.Context[parser.IFunctionContext],
	body *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (map[*symbol.Symbol]argument, bool) {
	exprs := make(map[string]parser.IExpressionContext)
	for _, a := range symbol.FunctionArguments(ctx.AST) {
		name := a.Name
		if name == "" {
			name = body.Type.Inputs[a.Index].Name
		}
		exprs[name] = a.Expr
	}
	args := make(map[*symbol.Symbol]argument, len(body.Type.Inputs))
	for _, param := range body.Type.Inputs {
		arg := argument{value: param.Value}
		if expr, bound := exprs[param.Name]; bound {
			var ok bool
			if arg, ok = bindArg(ctx, expr, body, param, kg, shell); !ok {
				return nil, false
			}
		}
		args[body.FindChild(param.Name)] = arg
	}
	return args, true
}

// bindArg returns what the call binds to the parameter param of body for expr.
func bindArg(
	ctx acontext.Context[parser.IFunctionContext],
	expr parser.IExpressionContext,
	body *symbol.Symbol,
	param types.Param,
	kg *keyGenerator,
	shell *shellBuilder,
) (argument, bool) {
	if param.Type.Kind != types.KindChan && readsChannel(ctx, expr) {
		ctx.Diagnostics.Add(codes.FuncArgValue.Atf(expr,
			"argument for parameter '%s' of %s '%s' cannot read a channel",
			param.Name, flow.BodyNoun(body), parser.FunctionName(ctx.AST),
		).WithNote("pass the channel to a chan parameter"))
		return argument{}, false
	}
	sym := body.FindChild(param.Name)
	if named := namedSymbol(ctx, expr); named != nil {
		if param.Type.Kind == types.KindChan {
			return argument{channel: shell.channelKey(named)}, true
		}
		return bindVariable(ctx, expr, named, sym, kg, shell)
	}
	if value, ok := ctx.FloatConstant(expr); ok {
		return argument{value: types.EncodeFloat(value)}, true
	}
	if !parser.IsNegatedLiteral(expr) && !literal.IsConstant(expr) {
		res, ok := analyzeExpression(ctx.Child(expr), kg, shell, false)
		if !ok {
			return argument{}, false
		}
		return follow(ctx, expr, res, sym, kg, shell)
	}
	parsed, err := literal.ParseConst(expr, param.Type)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, expr))
		return argument{}, false
	}
	return argument{value: parsed.Value}, true
}

// readsChannel reports whether expr reads a channel, by name or through a function.
func readsChannel(
	ctx acontext.Context[parser.IFunctionContext],
	expr parser.IExpressionContext,
) bool {
	synth, err := ctx.Scope.Root().GetChildByParserRule(expr)
	if err == nil && len(synth.Channels.Read) > 0 {
		return true
	}
	for _, primary := range parser.CollectNamedPrimaries(expr, ctx.FmtStrings) {
		head, tail := parser.PrimaryNameParts(primary)
		sym, err := ctx.Scope.ResolveMember(ctx, head, tail)
		if err == nil && sym.Type.Kind == types.KindChan {
			return true
		}
	}
	return false
}

// namedSymbol returns the variable or channel that expr names, or nil.
func namedSymbol(
	ctx acontext.Context[parser.IFunctionContext],
	expr parser.IExpressionContext,
) *symbol.Symbol {
	head, tail := parser.ExpressionNameParts(expr)
	if head == "" {
		return nil
	}
	sym, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil || sym.Kind == symbol.KindConstant {
		return nil
	}
	return sym
}

// bindVariable returns what the call binds to the value parameter param for sym.
func bindVariable(
	ctx acontext.Context[parser.IFunctionContext],
	expr parser.IExpressionContext,
	sym, param *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (argument, bool) {
	var (
		from     = sym.Type.UnwrapChan()
		converts = types.UnitsConvertible(from.Unit, param.Type.Unit)
		write    = expression.Write{AST: expr, Source: sym.Type, Target: param.Type}
	)
	outer, constant := shell.args[sym]
	if register := shell.varNodes[sym].register; !constant && register != nil {
		if !converts {
			return argument{variable: register}, true
		}
		expression.CheckUnits(ctx, write)
		scale := buildScaleNode(from, param.Type, kg)
		shell.edges = append(shell.edges, ir.Edge{
			Source: ir.Handle{Node: register.Key, Param: ir.DefaultOutputParam},
			Target: scale.input,
			Kind:   ir.EdgeKindContinuous,
		})
		return follow(ctx, expr, scale, param, kg, shell)
	}
	value := sym.DefaultValue
	if constant {
		value = outer.value
	}
	// A value that is not finite is text, the same in each unit.
	if _, text := value.(string); !converts || value == nil || text {
		return argument{value: value}, true
	}
	parsed, err := literal.Convert(
		telem.CastNumeric[float64](value), from.Unit, param.Type,
	)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, expr))
		return argument{}, false
	}
	if parsed.Truncated {
		expression.CheckUnits(ctx, write)
	}
	return argument{value: parsed.Value}, true
}

// follow returns the argument that holds each value res emits for the parameter param.
func follow(
	ctx acontext.Context[parser.IFunctionContext],
	expr parser.IExpressionContext,
	res nodeResult,
	param *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (argument, bool) {
	zero := types.Zero(param.Type)
	if zero == nil {
		ctx.Diagnostics.Add(codes.NotConstant.Atf(expr,
			"parameter '%s' of type %s takes only a constant", param.Name, param.Type))
		return argument{}, false
	}
	variable := buildVariableNode(param, kg, "f0", zero)
	// Both nodes stay live, so the parameter follows what res reads.
	shell.rootNodes = append(shell.rootNodes, variable, &res.node)
	shell.edges = append(shell.edges, ir.Edge{
		Source: res.output,
		Target: varFeederRef(variable).input,
		Kind:   ir.EdgeKindContinuous,
	})
	return argument{variable: variable}, true
}

// instantiate lowers a copy of the named stage or sequence body, keyed key, with args
// bound.
func instantiate[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	body *symbol.Symbol,
	args map[*symbol.Symbol]argument,
	key string,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, bool) {
	defer shell.enter(key, args)()
	collectVarNodes(body, kg, shell)
	scope, nodes, edges, ok := lowerBody(ctx, body, kg, shell)
	if !ok {
		return ir.Scope{}, false
	}
	scope.Key = key
	finishDispatchers(ctx.WithScope(body), bodyDispatchers(body, shell), kg, shell)
	shell.bodyNodes = append(shell.bodyNodes, nodes...)
	shell.edges = append(shell.edges, edges...)
	return scope, true
}

// checkDeclaration reports the errors of a named stage or sequence that only a use
// lowers.
func checkDeclaration[T identifierAST](
	ctx acontext.Context[T],
	shell *shellBuilder,
) bool {
	body, ok := resolveDeclScope(ctx)
	if !ok {
		return false
	}
	args := make(map[*symbol.Symbol]argument, len(body.Type.Inputs))
	for _, param := range body.Type.Inputs {
		sym := body.FindChild(param.Name)
		args[sym] = argument{value: types.Zero(param.Type), channel: uint32(sym.ID)}
	}
	// A builder and a key generator of its own keep the output out of the program.
	scratch := newShellBuilder(shell.synthByAST)
	scratch.varNodes = detachDispatchers(shell.varNodes)
	kg := newKeyGenerator(new(ir.Functions), "")
	_, ok = instantiate(ctx, body, args, body.QualifiedName(), kg, scratch)
	return ok
}

// detachDispatchers returns varNodes with its own copy of each dispatcher.
func detachDispatchers(
	varNodes map[*symbol.Symbol]varEntry,
) map[*symbol.Symbol]varEntry {
	detached := make(map[*symbol.Symbol]varEntry, len(varNodes))
	for sym, e := range varNodes {
		if e.disp != nil {
			node := *e.disp.node
			node.Inputs = slices.Clone(node.Inputs)
			node.Channels = node.Channels.Copy()
			e.disp = &dispatcher{node: &node, branches: slices.Clone(e.disp.branches)}
		}
		detached[sym] = e
	}
	return detached
}

// bodyDispatchers returns the dispatcher of each variable declared under scope.
func bodyDispatchers(scope *symbol.Symbol, shell *shellBuilder) []*dispatcher {
	var disps []*dispatcher
	for _, c := range scope.Children() {
		if c.Kind == symbol.KindModuleAlias {
			continue
		}
		if e := shell.varNodes[c]; e.disp != nil {
			disps = append(disps, e.disp)
		}
		disps = append(disps, bodyDispatchers(c, shell)...)
	}
	return disps
}

// lowerBody lowers the sequence or stage that body declares.
func lowerBody[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	body *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, []ir.Node, []ir.Edge, bool) {
	step := declaredAsStep(ctx.AST)
	switch decl := body.AST.(type) {
	case parser.IStageDeclarationContext:
		return lowerStage(ctx.Child(decl), body, step, kg, shell)
	case parser.ISequenceDeclarationContext:
		return lowerSequence(ctx.Child(decl), body, step, kg, shell)
	}
	ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
		"internal: '%s' declares no sequence or stage", body.Name))
	return ir.Scope{}, nil, nil, false
}

func analyzeIdentifierByRole(
	ctx acontext.Context[parser.IIdentifierContext],
	kg *keyGenerator,
	shell *shellBuilder,
	isFirst, isSink bool,
) (flowNodeResult, bool) {
	name := parser.IdentifierName(ctx.AST)
	sym, err := ctx.ResolveQualified(parser.IdentifierParts(ctx.AST))
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return flowNodeResult{}, false
	}

	switch sym.Kind {
	case symbol.KindConstant:
		value, ok := sym.DefaultValue.(float64)
		if !ok {
			ctx.Diagnostics.Add(diagnostics.Error(errors.Newf(
				"constant %s holds a %T, not a float64",
				sym.Name,
				sym.DefaultValue,
			), ctx.AST))
			return flowNodeResult{}, false
		}
		t := ctx.TypeMap[ctx.AST]
		n := newConstNode(
			kg.generate("const", sym.Name),
			t,
			t,
			types.EncodeFloat(value),
		)
		return flowNodeResult{
			node: newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam),
		}, true
	case symbol.KindSequence, symbol.KindStage:
		intent, ok := analyzeNamedRef(ctx, sym, shell)
		if !ok {
			return flowNodeResult{}, false
		}
		return flowNodeResult{transition: &intent}, true
	case symbol.KindVariable, symbol.KindStatefulVariable, symbol.KindChannel,
		symbol.KindFunction, symbol.KindBlock, symbol.KindInput, symbol.KindOutput,
		symbol.KindLoop, symbol.KindLoopVariable, symbol.KindModule,
		symbol.KindModuleAlias, symbol.KindAmbient:
		if isSink && sym.Root().UserDefined && sym.Root() != ctx.Scope.Root() {
			ctx.Diagnostics.Add(codes.ReadOnlyWrite.Atf(ctx.AST,
				"cannot write to '%s': it belongs to an imported module", name))
			return flowNodeResult{}, false
		}
		if rn := shell.varNodes[sym].readNode(sym); rn != nil {
			if isSink && isExprReadVar(sym) {
				ctx.Diagnostics.Add(codes.ReadOnlyWrite.Atf(ctx.AST,
					"cannot write to channel-read variable '%s'", name))
				return flowNodeResult{}, false
			}
			if isSink {
				return flowNodeResult{node: varFeederRef(rn)}, true
			}
			if isFirst {
				return flowNodeResult{node: varReadRef(rn)}, true
			}
			return flowNodeResult{node: buildVarReadNode(sym, rn, kg)}, true
		}
		if !isSink && isConstVar(sym) {
			return flowNodeResult{node: buildVarConstNode(sym, kg)}, true
		}
		if arg, bound := shell.args[sym]; bound && sym.Type.Kind != types.KindChan {
			n := newConstNode(
				kg.generate("const", sym.Name),
				sym.Type,
				sym.Type,
				arg.value,
			)
			return flowNodeResult{
				node: newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam),
			}, true
		}
		if isSink {
			r, ok := buildChannelWriteNode(name, sym, kg, shell)
			if e := shell.varNodes[sym]; ok && e.register != nil {
				bindWriteToAlias(shell, e.register, &r.node, sym)
			}
			return flowNodeResult{node: r}, ok
		}
		r, ok := buildChannelReadNode(name, sym, kg, shell)
		if e := shell.varNodes[sym]; ok && e.register != nil {
			bindReadToAlias(shell, e.register, &r.node, sym)
		}
		return flowNodeResult{node: r}, ok
	}
	panic("unknown symbol kind " + sym.Kind.String())
}

// bindReadToAlias feeds a read node's channel param from the alias's register
// node and declares every rebind candidate for dependency streaming.
func bindReadToAlias(shell *shellBuilder, vn, r *ir.Node, sym *symbol.Symbol) {
	shell.edges = append(shell.edges, ir.Edge{
		Source: ir.Handle{Node: vn.Key, Param: ir.DefaultOutputParam},
		Target: ir.Handle{Node: r.Key, Param: "channel"},
		Kind:   ir.EdgeKindContinuous,
	})
	maps.Copy(r.Channels.Read, sym.Channels.Read)
}

// bindWriteToAlias feeds a write node's channel param from the alias's register
// node and declares every rebind candidate for dependency streaming.
func bindWriteToAlias(shell *shellBuilder, vn, w *ir.Node, sym *symbol.Symbol) {
	shell.edges = append(shell.edges, ir.Edge{
		Source: ir.Handle{Node: vn.Key, Param: ir.DefaultOutputParam},
		Target: ir.Handle{Node: w.Key, Param: "channel"},
		Kind:   ir.EdgeKindContinuous,
	})
	maps.Copy(w.Channels.Write, sym.Channels.Write)
}

// isConstVar reports whether sym is a value variable holding a compile-time
// value, so a read of it collapses to a constant.
func isConstVar(sym *symbol.Symbol) bool {
	return sym.IsValueVariable() && !sym.IsReactive() && sym.SourceID == nil &&
		sym.DefaultValue != nil
}

// isExprReadVar reports whether sym holds a channel-read expression's value.
func isExprReadVar(sym *symbol.Symbol) bool {
	return sym.IsReactive() && sym.SourceID == nil
}

// collectVarNodes registers each variable's runtime nodes under s: a const
// holds its value in its register, an alias its key, an exprRead an index.
func collectVarNodes(s *symbol.Symbol, kg *keyGenerator, shell *shellBuilder) {
	for _, c := range s.Children() {
		// An alias lists its module's members. Their nodes come with the module.
		if c.Kind == symbol.KindModuleAlias {
			continue
		}
		// A named stage or sequence gets its variable nodes in each copy.
		if c.IsTemplate() {
			continue
		}
		if c.Reassigned && isConstVar(c) {
			shell.registerVar(c, varEntry{
				register: buildVariableNode(c, kg, "f0", c.DefaultValue),
			})
		}
		if zero := types.Zero(c.Type); c.IsComputed() && zero != nil {
			shell.registerVar(c, varEntry{
				register: buildVariableNode(c, kg, "f0", zero),
			})
		}
		if c.Reassigned && c.Kind == symbol.KindVariable &&
			c.Type.Kind == types.KindChan && c.SourceID != nil {
			shell.registerVar(
				c,
				varEntry{register: buildIndirectRegister(c, kg, shell)},
			)
		}
		if isExprReadVar(c) {
			e := varEntry{deref: buildVariableNode(c, kg, "value", nil)}
			if c.Reassigned {
				e.register = buildIndirectRegister(c, kg, shell)
				e.disp = buildDispatcher(c, e, kg, shell)
			}
			shell.registerVar(c, e)
		}
		collectVarNodes(c, kg, shell)
	}
}

// buildIndirectRegister builds the register naming sym's active source: the
// bound channel key for an alias, the active derivation index otherwise.
func buildIndirectRegister(
	sym *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) *ir.Node {
	value, outType := any(uint32(0)), types.U32()
	if sym.SourceID != nil {
		value, outType = shell.channelKey(sym), sym.Type
	}
	return &ir.Node{
		Key:      kg.generate("bind", sym.Name),
		Type:     "variable",
		Channels: types.NewChannels(),
		Inputs:   types.Params{{Name: "f0", Type: types.U32(), Value: value}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outType}},
	}
}

// scopeVarMembers returns members for the variable nodes declared in scopeSym,
// in declaration order with each init node before its register.
func scopeVarMembers(scopeSym *symbol.Symbol, shell *shellBuilder) ir.Members {
	var members ir.Members
	for _, c := range scopeSym.Children() {
		if c.Kind != symbol.KindVariable && c.Kind != symbol.KindStatefulVariable {
			continue
		}
		e := shell.varNodes[c]
		for _, n := range []*ir.Node{e.init, e.register, e.deref} {
			if n != nil {
				members = append(members, ir.Member{NodeKey: new(n.Key)})
			}
		}
	}
	return members
}

// memberNodeKeys returns every node key owned as a member under members.
func memberNodeKeys(members ir.Members) set.Set[string] {
	owned := set.New[string]()
	var walk func(members ir.Members)
	walk = func(members ir.Members) {
		for i := range members {
			if m := &members[i]; m.NodeKey != nil {
				owned.Add(*m.NodeKey)
			} else if m.Scope != nil {
				for _, stratum := range m.Scope.Strata {
					walk(stratum)
				}
				walk(m.Scope.Steps)
			}
		}
	}
	walk(members)
	return owned
}

// buildVariableNode builds sym's variable node. Its first param holds the initial
// value, edge-fed for a computed `$=`. Value-variable assignments append edge-fed
// feeder params.
func buildVariableNode(
	sym *symbol.Symbol,
	kg *keyGenerator,
	param string,
	value any,
) *ir.Node {
	nodeType := "variable"
	if sym.Kind == symbol.KindStatefulVariable {
		nodeType = "stateful_variable"
	}
	// A reactive variable holds its derivation's value type; f0 is edge-fed.
	t := sym.Type
	if t.Kind == types.KindChan {
		t = t.Unwrap()
	}
	return &ir.Node{
		Key:      kg.generate("var", sym.Name),
		Type:     nodeType,
		Channels: types.NewChannels(),
		Inputs:   types.Params{{Name: param, Type: t, Value: value}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: t}},
	}
}

// lowerVarDecl lowers a declaration's initializer: a channel-read derivation
// feeds the deref; a computed expression's entry node feeds the register.
func lowerVarDecl(
	ctx acontext.Context[parser.IVariableDeclarationContext],
	kg *keyGenerator,
	shell *shellBuilder,
) (varEntry, bool) {
	var (
		ident string
		expr  parser.IExpressionContext
	)
	switch {
	case ctx.AST.LocalVariable() != nil:
		local := ctx.AST.LocalVariable()
		ident, expr = local.IDENTIFIER().GetText(), local.Expression()
	case ctx.AST.StatefulVariable() != nil:
		stateful := ctx.AST.StatefulVariable()
		ident, expr = stateful.IDENTIFIER().GetText(), stateful.Expression()
	default:
		return varEntry{}, true
	}
	sym, err := ctx.Scope.Resolve(ctx, ident)
	if err != nil || expr == nil {
		return varEntry{}, true
	}
	e := shell.varNodes[sym]
	if sym.IsComputed() {
		return lowerComputedDecl(ctx, expr, sym, e, kg, shell)
	}
	if e.deref == nil || !isExprReadVar(sym) {
		return varEntry{}, true
	}
	if sym.Reassigned {
		return varEntry{}, collectBranch(ctx, expr, sym, kg, shell)
	}
	return varEntry{}, lowerDerivation(ctx, expr, e.deref, "value", kg, shell)
}

// lowerComputedDecl lowers a computed initializer as an entry node feeding the
// register. Its variable reads bind as VarRefs so it stays an entry node. A `:=`
// init feeds a feeder param and re-runs on every entry. A `$=` init feeds f0, which
// the schedulers run on the first entry only.
func lowerComputedDecl(
	ctx acontext.Context[parser.IVariableDeclarationContext],
	expr parser.IExpressionContext,
	sym *symbol.Symbol,
	e varEntry,
	kg *keyGenerator,
	shell *shellBuilder,
) (varEntry, bool) {
	if e.register == nil {
		ctx.Diagnostics.Add(codes.NotConstant.Atf(ctx.AST,
			"variable '%s' of type %s cannot have a computed initializer",
			sym.Name, sym.Type))
		return varEntry{}, false
	}
	res, ok := analyzeExpression(ctx.Child(expr), kg, shell, true)
	if !ok {
		return varEntry{}, false
	}
	e.init = &res.node
	shell.varNodes[sym] = e
	var target ir.Handle
	if sym.Kind == symbol.KindStatefulVariable {
		target = ir.Handle{Node: e.register.Key, Param: e.register.Inputs[0].Name}
	} else {
		target = varFeederRef(e.register).input
	}
	shell.edges = append(shell.edges, ir.Edge{
		Source: res.output,
		Target: target,
		Kind:   ir.EdgeKindContinuous,
	})
	return e, true
}

// collectBranch folds a derivation into sym's dispatcher: the branch function
// joins its body and the expression's lifted-var edges re-target its node.
func collectBranch[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
	sym *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) bool {
	res, ok := analyzeExpression(ctx.Child(expr), kg, shell, false)
	if !ok {
		return false
	}
	d := shell.varNodes[sym].disp
	maps.Copy(d.node.Channels.Read, res.node.Channels.Read)
	if branch, err := ctx.Scope.Root().GetChildByParserRule(expr); err == nil &&
		branch.Kind == symbol.KindFunction {
		for _, p := range branch.Type.Inputs {
			if _, exists := d.node.Inputs.Get(p.Name); exists {
				continue
			}
			// A parameter of the instance has its argument on the input of the branch.
			bound, _ := res.node.Inputs.Get(p.Name)
			d.node.Inputs = append(
				d.node.Inputs,
				types.Param{Name: p.Name, Type: p.Type, Value: bound.Value},
			)
		}
	}
	retargetEdges(ctx, shell, res.node.Key, d.node)
	d.branches = append(d.branches, res.node.Type)
	return true
}

// lowerDerivation lowers a channel-read expression into an always-live root
// subgraph, its output edged into vn's param and its channels wired as triggers.
func lowerDerivation[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
	vn *ir.Node,
	param string,
	kg *keyGenerator,
	shell *shellBuilder,
) bool {
	res, ok := analyzeExpression(ctx.Child(expr), kg, shell, false)
	if !ok {
		return false
	}
	shell.rootNodes = append(shell.rootNodes, &res.node)
	shell.edges = append(shell.edges, ir.Edge{
		Source: res.output,
		Target: ir.Handle{Node: vn.Key, Param: param},
		Kind:   ir.EdgeKindContinuous,
	})
	if exprSym, err := ctx.Scope.Root().GetChildByParserRule(expr); err == nil {
		tNodes, tEdges := lowerExprTriggers(ctx, exprSym, res.node.Key, kg, shell)
		for i := range tNodes {
			shell.rootNodes = append(shell.rootNodes, &tNodes[i])
		}
		shell.edges = append(shell.edges, tEdges...)
	}
	return true
}

// lowerExprTriggers builds an on node per channel exprSym reads, each edged
// into the expression's node so channel data drives recomputation.
func lowerExprTriggers[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	exprSym *symbol.Symbol,
	nodeKey string,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge) {
	names := make([]string, 0, len(exprSym.Channels.Read))
	for _, name := range exprSym.Channels.Read {
		names = append(names, name)
	}
	slices.Sort(names)
	var (
		nodes []ir.Node
		edges []ir.Edge
	)
	for _, name := range names {
		chanSym, err := ctx.Scope.Resolve(ctx, name)
		if err != nil {
			continue
		}
		result, ok := buildChannelReadNode(name, chanSym, kg, shell)
		if !ok {
			continue
		}
		if e := shell.varNodes[chanSym]; e.register != nil {
			bindReadToAlias(shell, e.register, &result.node, chanSym)
		}
		nodes = append(nodes, result.node)
		edges = append(edges, ir.Edge{
			Source: result.output,
			Target: ir.Handle{Node: nodeKey, Param: ir.DefaultInputParam},
			Kind:   ir.EdgeKindContinuous,
		})
	}
	return nodes, edges
}

// varReadRef references vn's held value. The empty node Key tells the flow
// processor to wire the handle without re-appending the node.
func varReadRef(vn *ir.Node) nodeResult {
	return nodeResult{
		node:   ir.Node{Outputs: vn.Outputs},
		output: ir.Handle{Node: vn.Key, Param: ir.DefaultOutputParam},
	}
}

// buildScaleNode builds the node that converts a value of type from to the unit of to.
func buildScaleNode(from, to types.Type, kg *keyGenerator) nodeResult {
	key := kg.generate("scale", "")
	inputs := types.Params{{Name: "value", Type: from}}
	outputs := types.Params{{Name: ir.DefaultOutputParam, Type: to}}
	*kg.synthFuncs = append(*kg.synthFuncs, ir.Function{
		Key:      compiler.ScaleSyntheticPrefix + key,
		Inputs:   inputs,
		Outputs:  outputs,
		Channels: types.NewChannels(),
	})
	return newNodeResult(ir.Node{
		Key:      key,
		Type:     compiler.ScaleSyntheticPrefix + key,
		Channels: types.NewChannels(),
		Inputs:   inputs,
		Outputs:  outputs,
	}, "value", ir.DefaultOutputParam)
}

// varFeederRef appends a fresh feeder param to vn and returns a reference
// targeting it, so each write gets its own last-write-wins input.
func varFeederRef(vn *ir.Node) nodeResult {
	param := fmt.Sprintf("f%d", len(vn.Inputs))
	vn.Inputs = append(vn.Inputs, types.Param{Name: param, Type: vn.Inputs[0].Type})
	return nodeResult{
		node:   ir.Node{Outputs: vn.Outputs},
		input:  ir.Handle{Node: vn.Key, Param: param},
		output: ir.Handle{Node: vn.Key, Param: ir.DefaultOutputParam},
	}
}

// lowerAssignment lowers `x = <rhs>` to a feeder: the RHS subgraph edged into
// a fresh input param on x's variable node.
func lowerAssignment(
	ctx acontext.Context[parser.IAssignmentContext],
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	sym, err := ctx.Scope.Resolve(ctx, ctx.AST.IDENTIFIER().GetText())
	if err != nil {
		return nil, nil, true
	}
	e := shell.varNodes[sym]
	expr := ctx.AST.Expression()
	if e.register == nil || expr == nil {
		return nil, nil, true
	}
	if sym.Type.Kind == types.KindChan {
		if sym.SourceID != nil {
			return lowerAliasRebind(ctx, e.register, expr, kg, shell)
		}
		return lowerExprReadReassign(ctx, sym, e, expr, kg, shell)
	}
	if v := statement.CastConstValue(ctx, expr, sym.Type); v != nil {
		n := newConstNode(kg.generate("const", sym.Name), sym.Type, sym.Type, v)
		ref := varFeederRef(e.register)
		return []ir.Node{n}, []ir.Edge{{
			Source: ir.Handle{Node: n.Key, Param: ir.DefaultOutputParam},
			Target: ref.input,
			Kind:   ir.EdgeKindContinuous,
		}}, true
	}
	res, ok := analyzeExpression(ctx.Child(expr), kg, shell, false)
	if !ok {
		return nil, nil, false
	}
	ref := varFeederRef(e.register)
	return []ir.Node{res.node}, []ir.Edge{{
		Source: res.output,
		Target: ref.input,
		Kind:   ir.EdgeKindContinuous,
	}}, true
}

// lowerRebind re-points a register: a constant feeder carrying the new source
// fires once per entry of the scope it appears in.
func lowerRebind(
	register *ir.Node,
	source uint32,
	kg *keyGenerator,
) ([]ir.Node, []ir.Edge, bool) {
	n := newConstNode(kg.generate("const", ""), types.U32(), types.U32(), source)
	ref := varFeederRef(register)
	return []ir.Node{n}, []ir.Edge{{
		Source: ir.Handle{Node: n.Key, Param: ir.DefaultOutputParam},
		Target: ref.input,
		Kind:   ir.EdgeKindContinuous,
	}}, true
}

// lowerExprReadReassign lowers `x = <expr>` on an exprRead variable: the RHS
// joins x's dispatcher as a branch and the step re-points x's register to it.
func lowerExprReadReassign(
	ctx acontext.Context[parser.IAssignmentContext],
	sym *symbol.Symbol,
	e varEntry,
	expr parser.IExpressionContext,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	if e.deref == nil {
		return nil, nil, true
	}
	idx := len(e.disp.branches)
	if !collectBranch(ctx, expr, sym, kg, shell) {
		return nil, nil, false
	}
	return lowerRebind(e.register, uint32(idx), kg)
}

// lowerAliasRebind lowers `alias = chan`: the step re-points the alias's
// register to the named channel's key.
func lowerAliasRebind(
	ctx acontext.Context[parser.IAssignmentContext],
	register *ir.Node,
	expr parser.IExpressionContext,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, bool) {
	if register == nil {
		return nil, nil, true
	}
	head, tail := parser.ExpressionNameParts(expr)
	if head == "" {
		return nil, nil, true
	}
	rhs, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, expr))
		return nil, nil, false
	}
	return lowerRebind(register, shell.channelKey(rhs), kg)
}

// buildDispatcher builds the $sel-keyed node folding sym's derivations,
// wired between the register ($sel) and the deref (value).
func buildDispatcher(
	sym *symbol.Symbol,
	e varEntry,
	kg *keyGenerator,
	shell *shellBuilder,
) *dispatcher {
	nodeKey := kg.generate("disp", sym.Name)
	n := &ir.Node{
		Key:      nodeKey,
		Type:     ir.DispatcherSyntheticPrefix + nodeKey,
		Channels: types.NewChannels(),
		Inputs:   types.Params{{Name: "$sel", Type: types.U32()}},
		Outputs: types.Params{
			{Name: ir.DefaultOutputParam, Type: e.deref.Outputs[0].Type},
		},
	}
	e.deref.Inputs = append(e.deref.Inputs, types.Param{Name: "sel", Type: types.U32()})
	shell.edges = append(shell.edges,
		ir.Edge{
			Source: ir.Handle{Node: e.register.Key, Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: nodeKey, Param: "$sel"},
			Kind:   ir.EdgeKindContinuous,
		},
		ir.Edge{
			Source: ir.Handle{Node: nodeKey, Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: e.deref.Key, Param: "value"},
			Kind:   ir.EdgeKindContinuous,
		},
		ir.Edge{
			Source: ir.Handle{Node: e.register.Key, Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: e.deref.Key, Param: "sel"},
			Kind:   ir.EdgeKindContinuous,
		},
	)
	return &dispatcher{node: n}
}

// buildDispatcherTriggers wires one channel-read trigger per union channel.
func buildDispatcherTriggers[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	nodeKey string,
	union types.Channels,
	kg *keyGenerator,
	shell *shellBuilder,
) types.Params {
	names := make([]string, 0, len(union.Read))
	for _, name := range union.Read {
		names = append(names, name)
	}
	slices.Sort(names)
	params := make(types.Params, 0, len(names))
	for ti, name := range names {
		chanSym, err := ctx.Scope.Resolve(ctx, name)
		if err != nil {
			continue
		}
		result, ok := buildChannelReadNode(name, chanSym, kg, shell)
		if !ok {
			continue
		}
		if e := shell.varNodes[chanSym]; e.register != nil {
			bindReadToAlias(shell, e.register, &result.node, chanSym)
		}
		param := fmt.Sprintf("$t%d", ti)
		params = append(
			params,
			types.Param{Name: param, Type: result.node.Outputs[0].Type},
		)
		shell.rootNodes = append(shell.rootNodes, &result.node)
		shell.edges = append(shell.edges, ir.Edge{
			Source: result.output,
			Target: ir.Handle{Node: nodeKey, Param: param},
			Kind:   ir.EdgeKindContinuous,
		})
	}
	return params
}

// finishDispatchers adds each dispatcher's channel triggers and emits its function.
func finishDispatchers[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	disps []*dispatcher,
	kg *keyGenerator,
	shell *shellBuilder,
) {
	slices.SortFunc(disps, func(a, b *dispatcher) int {
		return strings.Compare(a.node.Key, b.node.Key)
	})
	for _, d := range disps {
		d.node.Inputs = append(d.node.Inputs,
			buildDispatcherTriggers(ctx, d.node.Key, d.node.Channels, kg, shell)...)
		*kg.synthFuncs = append(*kg.synthFuncs, ir.Function{
			Key:      d.node.Type,
			Body:     ir.Body{Raw: strings.Join(d.branches, ",")},
			Inputs:   d.node.Inputs,
			Outputs:  d.node.Outputs,
			Channels: d.node.Channels.Copy(),
		})
	}
}

// retargetEdges re-points edges targeting a folded branch node at the
// dispatcher, deduping repeats and rejecting conflicting param sources.
func retargetEdges[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	shell *shellBuilder,
	branchKey string,
	disp *ir.Node,
) {
	kept := shell.edges[:0]
	for _, e := range shell.edges {
		if e.Target.Node != branchKey {
			kept = append(kept, e)
			continue
		}
		e.Target.Node = disp.Key
		if prior, found := ir.Edges(kept).FindByTarget(e.Target); found {
			if prior.Source != e.Source {
				ctx.Diagnostics.Add(codes.TriggerBinding.Atf(ctx.AST,
					"conflicting sources for dispatcher input %q", e.Target.Param))
			}
			continue
		}
		kept = append(kept, e)
	}
	shell.edges = kept
}

// newConstNode builds a constant node emitting v, with the given input and
// output types.
func newConstNode(key string, inType, outType types.Type, v any) ir.Node {
	return ir.Node{
		Key:      key,
		Type:     "constant",
		Channels: types.NewChannels(),
		Inputs:   types.Params{{Name: "value", Type: inType, Value: v}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outType}},
	}
}

// varValue returns sym's folded initializer, or the zero of its value type.
func varValue(sym *symbol.Symbol) any {
	if sym.DefaultValue != nil {
		return sym.DefaultValue
	}
	return types.Zero(sym.Type.UnwrapChan())
}

// buildVarReadNode lowers a triggered mid-chain variable read: a constant
// whose value input references the register, emitting the live value on fire.
func buildVarReadNode(sym *symbol.Symbol, vn *ir.Node, kg *keyGenerator) nodeResult {
	n := newConstNode(
		kg.generate("read", sym.Name),
		types.VarRef(sym.Type, vn.Key),
		sym.Type,
		varValue(sym),
	)
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam)
}

func buildVarConstNode(sym *symbol.Symbol, kg *keyGenerator) nodeResult {
	n := newConstNode(
		kg.generate("const", sym.Name), sym.Type, sym.Type, sym.DefaultValue,
	)
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam)
}

// analyzeNamedRef builds a transition intent for `=> X` where X is a step of an
// enclosing sequence. The innermost sequence that has the step X owns the transition.
func analyzeNamedRef(
	ctx acontext.Context[parser.IIdentifierContext],
	sym *symbol.Symbol,
	shell *shellBuilder,
) (transitionIntent, bool) {
	if frame := shell.resolveTargetFrame(sym.Name); frame != nil {
		return transitionIntent{memberKey: sym.Name, targetFrame: frame}, true
	}
	ctx.Diagnostics.Add(codes.TransitionTarget.Atf(
		ctx.AST,
		"'%s' is not reachable from here: it is neither a sibling within an "+
			"enclosing sequence nor a top-level scope",
		sym.Name,
	))
	return transitionIntent{}, false
}

func buildChannelReadNode(
	name string,
	sym *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (nodeResult, bool) {
	nodeKey := kg.generate("on", name)
	chKey := shell.channelKey(sym)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "on",
		Channels: types.NewChannels(),
		Inputs:   types.Params{{Name: "channel", Type: sym.Type, Value: chKey}},
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: sym.Type.Unwrap()}},
	}
	n.Channels.Read[chKey] = sym.Name
	return newNodeResult(n, "", ir.DefaultOutputParam), true
}

func buildChannelWriteNode(
	name string,
	sym *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (nodeResult, bool) {
	nodeKey := kg.generate("write", name)
	chKey := shell.channelKey(sym)
	n := ir.Node{
		Key:      nodeKey,
		Type:     "write",
		Channels: types.NewChannels(),
		Inputs: types.Params{
			{Name: ir.DefaultInputParam, Type: sym.Type.Unwrap()},
			{Name: "channel", Type: sym.Type, Value: chKey},
		},
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
	}
	n.Channels.Write[chKey] = sym.Name
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

// analyzeNextToken emits the intent of `next`, which ends the stage that holds it or
// completes the step that holds it.
func analyzeNextToken(
	ctx acontext.Context[parser.IFlowNodeContext],
	shell *shellBuilder,
) (flowNodeResult, bool) {
	if shell.exits == nil && shell.top() == nil {
		ctx.Diagnostics.Add(codes.TransitionTarget.Atf(
			ctx.AST, "'next' used outside of a stage or sequence",
		))
		return flowNodeResult{}, false
	}
	return flowNodeResult{transition: &transitionIntent{isNext: true}}, true
}

func analyzeFunctionNode(
	ctx acontext.Context[parser.IFunctionContext],
	kg *keyGenerator,
	shell *shellBuilder,
	triggered bool,
) (nodeResult, bool) {
	head, tail := parser.FunctionNameParts(ctx.AST)
	name := head
	if tail != "" {
		name = head + "." + tail
	}
	sym, err := ctx.ResolveQualified(head, tail)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}
	// A WASM-only callee was registered as a synthetic expression function.
	if sym.Exec == symbol.ExecWASM {
		synth, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return nodeResult{}, false
		}
		return lowerSynthNode(ctx, synth, kg, shell, triggered)
	}
	if sym.Type.Kind != types.KindFunction {
		ctx.Diagnostics.Add(codes.NotCallable.Atf(
			ctx.AST,
			"expected function type, got %s",
			sym.Type,
		))
		return nodeResult{}, false
	}
	key := kg.generate(name, "")
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
	inputs, args, ok := extractInputValues(
		ctx.Child(ctx.AST.InputValues()), n.Inputs, n, sym, kg, shell,
	)
	if !ok {
		return nodeResult{}, false
	}
	n.Inputs = inputs
	inputParam := ir.DefaultInputParam
	if sym.Trigger.Target != "" {
		inputParam = sym.Trigger.Target
	}
	res := newNodeResult(n, inputParam, firstOutputParam(n.Outputs))
	res.args = args
	return res, true
}

// analyzeExpression lowers an expression into a node. triggered means the node
// sits behind a flow trigger, so lifted value-variable params bind as VarRefs
// sampled per fire rather than edges that wake the node on writes.
func analyzeExpression(
	ctx acontext.Context[parser.IExpressionContext],
	kg *keyGenerator,
	shell *shellBuilder,
	triggered bool,
) (nodeResult, bool) {
	sym, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return nodeResult{}, false
	}

	if sym.Kind == symbol.KindConstant {
		outputType := ctx.Constraints.ApplySubstitutions(sym.Type.Outputs[0].Type)
		parsedValue, err := literal.ParseConst(ctx.AST, outputType)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return nodeResult{}, false
		}
		n := newConstNode(
			kg.generate("const", ""), outputType, outputType, parsedValue.Value,
		)
		return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
	}

	return lowerSynthNode(ctx, sym, kg, shell, triggered)
}

// lowerSynthNode builds the node for a synthetic expression function.
func lowerSynthNode[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	sym *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
	triggered bool,
) (nodeResult, bool) {
	key := kg.generate(sym.Name, "")
	freshType := types.Freshen(sym.Type, key)
	if len(freshType.Outputs) == 0 || !freshType.Outputs[0].Type.IsValid() {
		ctx.Diagnostics.Add(codes.FlowConnection.Atf(
			ctx.AST,
			"%s has no output value",
			ctx.AST.GetText(),
		))
		return nodeResult{}, false
	}
	outputType := ctx.Constraints.ApplySubstitutions(freshType.Outputs[0].Type)
	n := ir.Node{
		Key:      key,
		Type:     sym.QualifiedName(),
		Channels: sym.Channels.Copy(),
		Inputs:   freshType.Inputs,
		Outputs:  types.Params{{Name: ir.DefaultOutputParam, Type: outputType}},
	}
	wireVarEdges(ctx, shell, kg, &n, triggered)
	return newNodeResult(n, ir.DefaultInputParam, ir.DefaultOutputParam), true
}

// wireVarEdges feeds each lifted variable-read param from its variable node.
// On a triggered node a register-backed value variable binds as a VarRef
// instead of an edge: sampled per fire, and writes never wake the node.
func wireVarEdges[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	shell *shellBuilder,
	kg *keyGenerator,
	n *ir.Node,
	triggered bool,
) {
	for i, p := range n.Inputs {
		// A lifted input carries the name its expression reads, such as m.doubled.
		head, tail, _ := strings.Cut(p.Name, ".")
		vsym, err := ctx.Scope.ResolveMember(ctx, head, tail)
		if err != nil {
			continue
		}
		if arg, bound := shell.args[vsym]; bound {
			// Cloned so the value stays off the params that every instance shares.
			n.Inputs = slices.Clone(n.Inputs)
			n.Inputs[i].Value = arg.value
			if p.Type.Kind == types.KindChan {
				n.Inputs[i].Value = arg.channel
				bindParameterChannel(&n.Channels, vsym, arg.channel)
			}
			continue
		}
		e := shell.varNodes[vsym]
		// An alias lifts behind a register-bound read node, so the param
		// follows rebinds and gates on data arrival.
		if vsym.Type.Kind == types.KindChan && vsym.SourceID != nil &&
			e.register != nil {
			r, ok := buildChannelReadNode(p.Name, vsym, kg, shell)
			if !ok {
				continue
			}
			bindReadToAlias(shell, e.register, &r.node, vsym)
			shell.rootNodes = append(shell.rootNodes, &r.node)
			shell.edges = append(shell.edges, ir.Edge{
				Source: r.output,
				Target: ir.Handle{Node: n.Key, Param: p.Name},
				Kind:   ir.EdgeKindContinuous,
			})
			continue
		}
		if triggered && e.deref == nil && e.register != nil {
			// Cloned so the stamp stays off the synth function's shared params.
			n.Inputs = slices.Clone(n.Inputs)
			n.Inputs[i].Type = types.VarRef(p.Type, e.register.Key)
			continue
		}
		src := e.deref
		if src == nil {
			src = e.register
		}
		if src != nil {
			shell.edges = append(shell.edges, ir.Edge{
				Source: ir.Handle{Node: src.Key, Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: n.Key, Param: p.Name},
				Kind:   ir.EdgeKindContinuous,
			})
		}
	}
}

// bindParameterChannel replaces the slot of the parameter in channels with its channel.
func bindParameterChannel(channels *types.Channels, param *symbol.Symbol, key uint32) {
	slot := uint32(param.ID)
	if _, ok := channels.Read[slot]; ok {
		delete(channels.Read, slot)
		channels.Read[key] = param.Name
	}
	if _, ok := channels.Write[slot]; ok {
		delete(channels.Write, slot)
		channels.Write[key] = param.Name
	}
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
	return analyze(ctx, t, root, false, cfgs)
}

// AnalyzeModule is Analyze under the module rules.
func AnalyzeModule(
	ctx context.Context,
	t Text,
	root *symbol.Symbol,
	cfgs ...parser.Config,
) (ir.IR, *diagnostics.Diagnostics) {
	return analyze(ctx, t, root, true, cfgs)
}

func analyze(
	ctx context.Context,
	t Text,
	root *symbol.Symbol,
	module bool,
	cfgs []parser.Config,
) (ir.IR, *diagnostics.Diagnostics) {
	u, diag := analyzeUnit(ctx, t, root, module, cfgs)
	return u.ir, diag
}

// analyzeUnit lowers t into a unit. An automation links the modules its root
// loaded. A module lowers alone, so each module enters a program once.
func analyzeUnit(
	ctx context.Context,
	t Text,
	root *symbol.Symbol,
	module bool,
	cfgs []parser.Config,
) (unit, *diagnostics.Diagnostics) {
	var (
		aCtx = acontext.NewRoot(ctx, t.AST, root).
			WithConfig(parser.ConfigOf(cfgs...)).
			WithModule(module)
		i = ir.IR{
			Symbols:    aCtx.Scope,
			TypeMap:    aCtx.TypeMap,
			FmtStrings: aCtx.FmtStrings,
		}
	)

	analyzer.AnalyzeProgram(aCtx)
	i.Authorities = authority.Analyze(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return unit{ir: i}, aCtx.Diagnostics
	}

	i.Functions = collectFunctions(i.Symbols)
	kg := newKeyGenerator(&i.Functions, root.Name)
	shell := newShellBuilder(collectSynthByAST(aCtx.Scope.Root()))
	collectVarNodes(aCtx.Scope.Root(), kg, shell)
	// Every unit reads the variable nodes of the modules loaded before it. Only an
	// automation links their code.
	deps := loadedDeps(root, &i, shell)
	var units []*unit
	if !module {
		units = deps
	}

	// The root scope is always parallel and always-live.
	i.Root = ir.Scope{
		Mode:     ir.ScopeModeParallel,
		Liveness: ir.LivenessAlways,
	}
	rootMembers, ok := lowerTopLevelItems(aCtx, &i, kg, shell)
	if !ok {
		return unit{ir: i}, aCtx.Diagnostics
	}

	// Anonymous bodies live as members of their enclosing scope; their flat IR
	// still registers in the program's global node and edge lists.
	i.Nodes = append(i.Nodes, shell.bodyNodes...)
	appendDispatcherFunctions(aCtx, root, kg, shell)
	rootMembers = appendRootNodes(&i, rootMembers, shell.rootNodes)
	i.Edges = append(i.Edges, shell.edges...)

	if len(rootMembers) > 0 {
		i.Root.Strata = []ir.Members{rootMembers}
	}

	bound := set.New[string]()
	bindActivations(&i.Root, shell.activations, bound)
	link(&i, units)
	checkActivationsBound(aCtx, shell.activations, bound)
	if d := resolveAndStratify(aCtx, &i, deps, module); d != nil {
		return unit{ir: i}, d
	}
	return unit{ir: i, varNodes: shell.varNodes}, aCtx.Diagnostics
}

func collectFunctions(scope *symbol.Symbol) []ir.Function {
	var fns []ir.Function
	for _, c := range scope.Children() {
		if c.Kind != symbol.KindFunction || c.AST == nil {
			continue
		}
		bodyAst := c.AST
		if fnDecl, ok := c.AST.(parser.IFunctionDeclarationContext); ok {
			bodyAst = fnDecl.Block()
		}
		fns = append(fns, ir.Function{
			Key:      c.QualifiedName(),
			Body:     ir.Body{Raw: bodyAst.GetText(), AST: bodyAst},
			Inputs:   c.Type.Inputs,
			Outputs:  c.Type.Outputs,
			Channels: c.Channels,
			Symbol:   c,
		})
	}
	return fns
}

func loadedDeps(root *symbol.Symbol, i *ir.IR, shell *shellBuilder) []*unit {
	l := loaderOf(root)
	if l == nil {
		return nil
	}
	deps := l.loaded()
	for _, u := range deps {
		maps.Copy(shell.varNodes, u.varNodes)
		maps.Copy(i.TypeMap, u.ir.TypeMap)
		maps.Copy(i.FmtStrings, u.ir.FmtStrings)
	}
	return deps
}

// appendRootNodes appends the always-live nodes to i, adds each one not already in
// members to it, and returns members.
func appendRootNodes(i *ir.IR, members ir.Members, nodes []*ir.Node) ir.Members {
	owned := memberNodeKeys(members)
	for _, vn := range nodes {
		i.Nodes = append(i.Nodes, *vn)
		if !owned.Contains(vn.Key) {
			members = append(members, ir.Member{NodeKey: new(vn.Key)})
		}
	}
	return members
}

// checkActivationsBound is a safety net: a flow that starts a stage or sequence
// places its scope in the scope that holds the flow, so every activation binds.
func checkActivationsBound(
	aCtx acontext.Context[parser.IProgramContext],
	activations map[string][]ir.Activation,
	bound set.Set[string],
) {
	for key := range activations {
		if !bound.Contains(key) {
			aCtx.Diagnostics.Add(diagnostics.Errorf(
				aCtx.AST,
				"internal: activation target '%s' did not bind to any scope",
				key,
			))
		}
	}
}

// lowerTopLevelItems lowers every top-level item into i and returns the members of
// the root scope. Module-scope flow nodes become leaf-node members; top-level
// sequence and stage declarations become nested Scope members.
func lowerTopLevelItems(
	aCtx acontext.Context[parser.IProgramContext],
	i *ir.IR,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Members, bool) {
	var rootMembers ir.Members
	for _, item := range aCtx.AST.AllTopLevelItem() {
		members, nodes, edges, ok := lowerTopLevelItem(aCtx, item, kg, shell)
		if !ok {
			return nil, false
		}
		rootMembers = append(rootMembers, members...)
		i.Nodes = append(i.Nodes, nodes...)
		i.Edges = append(i.Edges, edges...)
	}
	return rootMembers, true
}

func lowerTopLevelItem(
	aCtx acontext.Context[parser.IProgramContext],
	item parser.ITopLevelItemContext,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Members, []ir.Node, []ir.Edge, bool) {
	if decl := item.VariableDeclaration(); decl != nil {
		e, ok := lowerVarDecl(aCtx.Child(decl), kg, shell)
		if !ok || e.init == nil {
			return nil, nil, nil, ok
		}
		// The init node and register precede the flows that read them.
		members := ir.Members{
			{NodeKey: new(e.init.Key)},
			{NodeKey: new(e.register.Key)},
		}
		return members, []ir.Node{*e.init}, nil, true
	}
	if flow := item.FlowStatement(); flow != nil {
		nodes, edges, bodyMembers, _, ok := analyzeFlow(aCtx.Child(flow), kg, shell)
		if !ok {
			return nil, nil, nil, false
		}
		members := make(ir.Members, 0, len(nodes)+len(bodyMembers))
		for _, n := range nodes {
			members = append(members, ir.Member{NodeKey: new(n.Key)})
		}
		return append(members, bodyMembers...), nodes, edges, true
	}
	decl := declarationOf(item)
	if decl == nil {
		return nil, nil, nil, true
	}
	// A named declaration is a template, and each use lowers a copy of it.
	if decl.IDENTIFIER() != nil {
		return nil, nil, nil, checkDeclaration(aCtx.Child(decl), shell)
	}
	scope, nodes, edges, ok := lowerDeclaration(aCtx.Child(decl), kg, shell)
	if !ok {
		return nil, nil, nil, false
	}
	return ir.Members{{Scope: &scope}}, nodes, edges, true
}

// appendDispatcherFunctions gives each reassigned variable's dispatcher in root its
// union channel triggers and appends its function to the program. A loaded module's
// dispatchers already carry theirs.
func appendDispatcherFunctions(
	aCtx acontext.Context[parser.IProgramContext],
	root *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) {
	disps := make([]*dispatcher, 0, len(shell.varNodes))
	for sym, e := range shell.varNodes {
		if e.disp != nil && sym.Root() == root {
			disps = append(disps, e.disp)
		}
	}
	finishDispatchers(aCtx, disps, kg, shell)
}

// bindActivations stamps each deferred activation of a stage or sequence that a flow
// starts onto the matching scope member under s, and adds the key of each bound
// scope to bound.
func bindActivations(
	s *ir.Scope,
	activations map[string][]ir.Activation,
	bound set.Set[string],
) {
	visit := func(m *ir.Member) {
		if m.Scope == nil {
			return
		}
		if a, ok := activations[m.Scope.Key]; ok {
			m.Scope.Activations = a
			bound.Add(m.Scope.Key)
		}
		bindActivations(m.Scope, activations, bound)
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

// resolveAndStratify resolves the node types of i and stratifies it. It returns
// the diagnostics to report when either step fails, and nil otherwise.
func resolveAndStratify(
	aCtx acontext.Context[parser.IProgramContext],
	i *ir.IR,
	deps []*unit,
	module bool,
) *diagnostics.Diagnostics {
	if len(i.Nodes) == 0 {
		return nil
	}
	// A module's edges reach the variable nodes of the modules it imports,
	// so it is checked beside them and keeps only its own nodes after.
	own, ownEdges := len(i.Nodes), len(i.Edges)
	if module {
		for _, u := range deps {
			i.Nodes = append(i.Nodes, u.ir.Nodes...)
			i.Edges = append(i.Edges, u.ir.Edges...)
		}
	}
	ok := analyzer.ResolveNodeTypes(
		i.Nodes,
		i.Edges,
		aCtx.Constraints,
		aCtx.Diagnostics,
	)
	var d *diagnostics.Diagnostics
	if ok {
		d = stratifier.Stratify(i, aCtx.Diagnostics)
	}
	i.Nodes, i.Edges = i.Nodes[:own:own], i.Edges[:ownEdges:ownEdges]
	if !ok {
		return aCtx.Diagnostics
	}
	if d != nil && !d.Ok() {
		return d
	}
	return nil
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
	// bodyMembers collects lowered anonymous-body scopes for placement as members
	// of the scope enclosing this flow.
	bodyMembers []ir.Member
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
	op, _ := children[p.lastOpIndex].(parser.IFlowOperatorContext)
	return flowOperatorEdgeKind(op)
}

// injectImplicitTriggers adds the implicit triggers of the flow statement's head.
func (p *flowChainProcessor) injectImplicitTriggers(
	head antlr.ParserRuleContext,
) bool {
	triggers, ok := implicitTriggers(p.ctx, p.kg, p.shell, head)
	if !ok {
		return false
	}
	for _, t := range triggers {
		if t.node.Key != "" {
			p.nodes = append(p.nodes, t.node)
		}
		p.addTrigger(t)
	}
	return true
}

// addTrigger makes result the head's trigger, or an extra one if it has one.
func (p *flowChainProcessor) addTrigger(result nodeResult) {
	if p.prevNode == nil {
		p.prevOutput = result.output
		p.prevNode = &result.node
		return
	}
	p.additionalTriggers = append(p.additionalTriggers, result)
}

// implicitTriggers returns a trigger for each channel or reactive variable that head
// reads, so `f(count * 2)` expands to `count -> f(count * 2)`.
func implicitTriggers[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	kg *keyGenerator,
	shell *shellBuilder,
	head antlr.ParserRuleContext,
) ([]nodeResult, bool) {
	var triggers []nodeResult
	for _, chName := range headChannelReads(ctx, head) {
		chanSym, err := ctx.Scope.Resolve(ctx, chName)
		if err != nil {
			continue
		}
		result, ok := buildChannelReadNode(chName, chanSym, kg, shell)
		if !ok {
			return nil, false
		}
		if e := shell.varNodes[chanSym]; e.register != nil {
			bindReadToAlias(shell, e.register, &result.node, chanSym)
		}
		triggers = append(triggers, result)
	}
	for _, vn := range headVarReads(ctx, shell, head) {
		triggers = append(triggers, varReadRef(vn))
	}
	return triggers, true
}

// headVarReads returns the nodes of the reactive variables a call's inputs read.
func headVarReads[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	shell *shellBuilder,
	head antlr.ParserRuleContext,
) []*ir.Node {
	fn, ok := head.(parser.IFunctionContext)
	if !ok {
		return nil
	}
	var nodes []*ir.Node
	for _, arg := range symbol.FunctionArguments(fn) {
		for _, pe := range parser.CollectNamedPrimaries(arg.Expr, ctx.FmtStrings) {
			head, tail := parser.PrimaryNameParts(pe)
			sym, err := ctx.Scope.ResolveMember(ctx, head, tail)
			if err != nil || !isExprReadVar(sym) {
				continue
			}
			rn := shell.varNodes[sym].readNode(sym)
			if rn != nil && !slices.Contains(nodes, rn) {
				nodes = append(nodes, rn)
			}
		}
	}
	return nodes
}

// headChannelReads returns the channels a flow head reads, through its own
// expression or through a call's computed inputs. A constant reads none.
func headChannelReads[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	head antlr.ParserRuleContext,
) map[uint32]string {
	root := ctx.Scope.Root()
	reads := make(map[uint32]string)
	if sym, err := root.GetChildByParserRule(head); err == nil {
		if sym.Kind == symbol.KindConstant {
			return nil
		}
		maps.Copy(reads, sym.Channels.Read)
	}
	fn, ok := head.(parser.IFunctionContext)
	if !ok {
		return reads
	}
	for _, arg := range symbol.FunctionArguments(fn) {
		if synth, err := root.GetChildByParserRule(arg.Expr); err == nil {
			maps.Copy(reads, synth.Channels.Read)
		}
	}
	return reads
}

func (p *flowChainProcessor) processFlowNode(flowNode parser.IFlowNodeContext) bool {
	p.currentIndex++
	isLast := p.currentIndex == p.totalFlowNodes
	isSink := isLast && flowNode.Identifier() != nil

	// Inject implicit triggers for an expression as the first node. A function
	// node only has a synthetic when its callee is WASM-only.
	if p.currentIndex == 1 && p.prevNode == nil {
		var head antlr.ParserRuleContext
		if expr := flowNode.Expression(); expr != nil {
			head = expr
		} else if fn := flowNode.Function(); fn != nil {
			head = fn
		}
		if head != nil && !p.injectImplicitTriggers(head) {
			return false
		}
	}

	isFirst := p.prevNode == nil
	result, ok := analyzeFlowNode(
		p.ctx.Child(flowNode), p.kg, p.shell, isFirst, isSink,
	)
	if !ok {
		return false
	}

	if result.bodyScope != nil {
		p.bodyMembers = append(p.bodyMembers, ir.Member{Scope: result.bodyScope})
	}

	if result.transition != nil {
		return p.consumeTransition(*result.transition, flowNode)
	}

	node := result.node
	if isSink && node.node.Key == "" && len(p.nodes) == 0 && declaredAsStep(p.ctx.AST) {
		p.readStepSource()
	}
	if p.prevNode != nil {
		if len(p.prevNode.Outputs) == 0 {
			p.ctx.Diagnostics.Add(codes.FlowConnection.Atf(
				flowNode,
				"function '%s' has no output to connect in flow chain",
				p.prevNode.Type,
			))
			return false
		}
		// A value written to a variable of another unit converts on the way.
		if from, to := p.prevNode.Outputs[0].Type, node.node.Outputs; isSink &&
			len(to) > 0 && types.UnitsConvertible(from.Unit, to[0].Type.Unit) {
			scale := buildScaleNode(from, to[0].Type, p.kg)
			p.edges = append(p.edges, ir.Edge{
				Source: p.prevOutput,
				Target: scale.input,
				Kind:   p.edgeKind(),
			})
			p.nodes = append(p.nodes, scale.node)
			p.prevOutput = scale.output
		}
		p.edges = append(p.edges, ir.Edge{
			Source: p.prevOutput,
			Target: node.input,
			Kind:   p.edgeKind(),
		})
	}

	// A computed input fires on the same trigger as its call and precedes it in
	// member order, so the call samples the value computed in the same pass.
	for _, arg := range node.args {
		if p.prevNode != nil {
			p.edges = append(p.edges, ir.Edge{
				Source: p.prevOutput,
				Target: ir.Handle{Node: arg.Key, Param: ir.DefaultInputParam},
				Kind:   p.edgeKind(),
			})
		}
		for _, trigger := range p.additionalTriggers {
			p.edges = append(p.edges, ir.Edge{
				Source: trigger.output,
				Target: ir.Handle{Node: arg.Key, Param: ir.DefaultInputParam},
				Kind:   ir.EdgeKindContinuous,
			})
		}
		p.nodes = append(p.nodes, arg)
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

// readStepSource reads the source variable of a variable-to-variable step through a
// node in the step, so the step has an output to complete on.
func (p *flowChainProcessor) readStepSource() {
	t := p.prevNode.Outputs[0].Type
	read := newConstNode(
		p.kg.generate("read", ""),
		types.VarRef(t, p.prevOutput.Node),
		t,
		types.Zero(t),
	)
	p.edges = append(p.edges, ir.Edge{
		Source: p.prevOutput,
		Target: ir.Handle{Node: read.Key, Param: ir.DefaultInputParam},
		Kind:   ir.EdgeKindContinuous,
	})
	p.nodes = append(p.nodes, read)
	p.prevOutput = ir.Handle{Node: read.Key, Param: ir.DefaultOutputParam}
}

// consumeTransition records a transition and/or activation for a flow chain
// whose terminal token is `=> next`, `=> scope_name`, or a scope-valued
// identifier. The firing handle is the previous node's output.
func (p *flowChainProcessor) consumeTransition(
	intent transitionIntent,
	ast parser.IFlowNodeContext,
) bool {
	if p.prevNode == nil {
		p.ctx.Diagnostics.Add(codes.TransitionTarget.Atf(
			ast, "transition target requires a source",
		))
		return false
	}
	if len(p.prevNode.Outputs) == 0 {
		p.ctx.Diagnostics.Add(codes.FlowConnection.Atf(
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

	p.shell.applyTransitionIntent(p.prevOutput, p.edgeKind(), intent)
	p.transitionEmitted = true
	return true
}

func (p *flowChainProcessor) processRoutingTable(rt parser.IRoutingTableContext) bool {
	// SY-5483: a table with no node before it is an input table, which is unsupported.
	if p.prevNode == nil {
		p.ctx.Diagnostics.Add(codes.RoutingTable.Atf(
			p.ctx.AST,
			"input routing tables are not supported",
		))
		return false
	}
	newNodes, newEdges, bodyMembers, ok := analyzeOutputRoutingTable(
		p.ctx.Child(rt),
		*p.prevNode,
		p.kg,
		p.shell,
	)
	if !ok {
		return false
	}
	p.nodes = append(p.nodes, newNodes...)
	p.edges = append(p.edges, newEdges...)
	p.bodyMembers = append(p.bodyMembers, bodyMembers...)
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
) (nodes []ir.Node, edges []ir.Edge, bodyMembers []ir.Member, transitionEmitted, ok bool) {
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
	return p.nodes, p.edges, p.bodyMembers, p.transitionEmitted, true
}

func extractInputValues(
	ctx acontext.Context[parser.IInputValuesContext],
	input types.Params,
	node ir.Node,
	fnSym *symbol.Symbol,
	kg *keyGenerator,
	shell *shellBuilder,
) (types.Params, []ir.Node, bool) {
	if ctx.AST == nil {
		return input, nil, true
	}
	p := inputParser{ctx: ctx, node: &node, fnSym: fnSym, kg: kg, shell: shell}
	ok := true
	if named := ctx.AST.NamedInputValues(); named != nil {
		ok = p.parseNamed(named, input)
	} else if anon := ctx.AST.AnonymousInputValues(); anon != nil {
		ok = p.parseAnonymous(anon, input)
	}
	if !ok {
		return nil, nil, false
	}
	return input, p.args, true
}

type inputParser struct {
	ctx   acontext.Context[parser.IInputValuesContext]
	node  *ir.Node
	fnSym *symbol.Symbol
	kg    *keyGenerator
	shell *shellBuilder
	// args holds the nodes lowered from non-literal input expressions.
	args []ir.Node
}

func (p *inputParser) parseNamed(
	named parser.INamedInputValuesContext,
	input types.Params,
) bool {
	for _, cv := range named.AllNamedInputValue() {
		expr := cv.Expression()
		if expr == nil {
			continue
		}
		key := cv.IDENTIFIER().GetText()
		idx := input.GetIndex(key)
		value, typ, ok := p.parse(expr, input[idx].Type, key)
		if !ok {
			return false
		}
		input[idx].Value = value
		input[idx].Type = typ
	}
	return true
}

func (p *inputParser) parseAnonymous(
	anon parser.IAnonymousInputValuesContext,
	input types.Params,
) bool {
	exprs := anon.AllExpression()
	pos := 0
	for i := range input {
		if input[i].Name == p.fnSym.Trigger.Target {
			continue
		}
		if pos >= len(exprs) {
			break
		}
		value, typ, ok := p.parse(
			exprs[pos], input[i].Type, fmt.Sprintf("position %d", pos),
		)
		if !ok {
			return false
		}
		input[i].Value = value
		input[i].Type = typ
		pos++
	}
	return true
}

func (p *inputParser) parse(
	expr parser.IExpressionContext,
	paramType types.Type,
	paramName string,
) (any, types.Type, bool) {
	if paramType.Kind == types.KindChan {
		return p.parseChannel(expr, paramType, paramName)
	}
	// Top-level literal variables cannot be reassigned, so they inline; so
	// does any literal variable with no write sites in the source. Any
	// other value variable stamps a var ref to its node instead.
	if head, tail := parser.ExpressionNameParts(expr); head != "" {
		if value, typ, ok, done := p.parseName(
			expr, head, tail, paramType, paramName,
		); done {
			return value, typ, ok
		}
	}
	if value, ok := p.ctx.FloatConstant(expr); ok {
		return types.EncodeFloat(value), paramType, true
	}
	// Any other expression becomes a node that fires with the call and is
	// sampled through a var ref, so its value is the one at fire time.
	if !literal.IsConstant(expr) {
		res, ok := analyzeExpression(p.ctx.Child(expr), p.kg, p.shell, true)
		if !ok {
			return nil, paramType, false
		}
		p.args = append(p.args, res.node)
		t := res.node.Outputs[0].Type
		return types.Zero(t), types.VarRef(t, res.node.Key), true
	}
	parsedValue, err := literal.ParseConst(expr, paramType)
	if err != nil {
		p.ctx.Diagnostics.Add(diagnostics.Error(err, expr))
		return nil, paramType, false
	}
	return parsedValue.Value, paramType, true
}

// parseName parses an input expression that names a symbol. done is false when
// the expression falls through to the other input forms.
func (p *inputParser) parseName(
	expr parser.IExpressionContext,
	head string,
	tail string,
	paramType types.Type,
	paramName string,
) (value any, typ types.Type, ok, done bool) {
	sym, err := p.ctx.Scope.ResolveMember(p.ctx, head, tail)
	if err != nil {
		return nil, paramType, false, false
	}
	if arg, bound := p.shell.args[sym]; bound && sym.Type.Kind != types.KindChan {
		value, ok = p.convertArgument(expr, sym, arg.value, paramType)
		return value, paramType, ok, true
	}
	// A node holds a variable that can change, and a parameter bound to one.
	rn := p.shell.varNodes[sym].readNode(sym)
	if rn != nil || (sym.IsLiteral() && !sym.Reassigned && sym.DefaultValue != nil) {
		value, typ, ok = p.parseVariable(expr, sym, rn, paramType)
		return value, typ, ok, true
	}
	if sym.Type.Kind == types.KindChan {
		// SY-5823: This error goes away when the fix lands, so it has no code.
		p.ctx.Diagnostics.Add(diagnostics.Errorf(
			expr,
			"input value for '%s' cannot be a channel",
			paramName,
		))
		return nil, paramType, false, true
	}
	return nil, paramType, false, false
}

// parseVariable parses an input that reads the variable sym, in the unit of the
// input. rn is the node that holds the variable, or nil for a variable that inlines.
func (p *inputParser) parseVariable(
	expr parser.IExpressionContext,
	sym *symbol.Symbol,
	rn *ir.Node,
	paramType types.Type,
) (any, types.Type, bool) {
	t, value := sym.Type.UnwrapChan(), varValue(sym)
	convertible := types.UnitsConvertible(t.Unit, paramType.Unit)
	check := convertible && rn != nil
	// A value that is not finite is text, the same in each unit.
	_, nonFinite := sym.DefaultValue.(string)
	if convertible && sym.DefaultValue != nil && !nonFinite {
		parsed, err := literal.Convert(
			telem.CastNumeric[float64](sym.DefaultValue), t.Unit, paramType,
		)
		if err != nil {
			p.ctx.Diagnostics.Add(diagnostics.Error(err, expr))
			return nil, paramType, false
		}
		value, check = parsed.Value, check || parsed.Truncated
	}
	if check {
		expression.CheckUnits(p.ctx, expression.Write{
			AST: expr, Source: sym.Type, Target: paramType,
		})
	}
	if rn == nil {
		return value, paramType, true
	}
	ref := types.VarRef(t, rn.Key)
	if !convertible {
		return value, ref, true
	}
	// A native node converts the register when it reads it. Any other node reads a
	// scale node that fires with the call.
	if p.fnSym.Exec != symbol.ExecFlow {
		scale := buildScaleNode(ref, paramType, p.kg)
		p.args = append(p.args, scale.node)
		return value, types.VarRef(paramType, scale.node.Key), true
	}
	ref.Unit = paramType.Unit
	return value, ref, true
}

// convertArgument returns the value bound to the parameter sym in the unit of
// paramType.
func (p *inputParser) convertArgument(
	expr parser.IExpressionContext,
	sym *symbol.Symbol,
	value any,
	paramType types.Type,
) (any, bool) {
	// A value that is not finite is text, the same in each unit.
	_, nonFinite := value.(string)
	if value == nil || nonFinite ||
		!types.UnitsConvertible(sym.Type.Unit, paramType.Unit) {
		return value, true
	}
	parsed, err := literal.Convert(
		telem.CastNumeric[float64](value), sym.Type.Unit, paramType,
	)
	if err != nil {
		p.ctx.Diagnostics.Add(diagnostics.Error(err, expr))
		return nil, false
	}
	return parsed.Value, true
}

func (p *inputParser) parseChannel(
	expr parser.IExpressionContext,
	paramType types.Type,
	paramName string,
) (any, types.Type, bool) {
	channelName := parser.GetExpressionText(expr)
	sym, err := p.ctx.Scope.Resolve(p.ctx, channelName)
	if err != nil {
		p.ctx.Diagnostics.Add(diagnostics.Error(err, expr))
		return nil, paramType, false
	}
	if isExprReadVar(sym) {
		p.ctx.Diagnostics.Add(codes.ChannelExpected.Atf(
			expr,
			"input value for '%s' must be a channel",
			paramName,
		))
		return nil, paramType, false
	}
	if err := paramType.ChanDirection.CheckCompatibility(
		sym.Type.ChanDirection,
	); err != nil {
		p.ctx.Diagnostics.Add(codes.ReadOnlyWrite.Wrap(err, expr))
		return nil, paramType, false
	}
	// Top-level channel aliases cannot be rebound, so they collapse into
	// their declared channel; so does any alias with no rebind sites in
	// the source. Any other alias reads its register's live key instead.
	if e := p.shell.varNodes[sym]; e.register != nil && sym.IsValueVariable() &&
		sym.Type.Kind == types.KindChan {
		p.bindAliasRegister(sym, e.register, paramName)
		return nil, paramType, true
	}
	key := p.shell.channelKey(sym)
	symbol.ResolveInputChannel(&p.node.Channels, p.fnSym, paramName, key, sym.Name)
	return key, paramType, true
}

// bindAliasRegister feeds the live key of a rebindable channel alias into the
// paramName input from the alias's register.
func (p *inputParser) bindAliasRegister(
	sym *symbol.Symbol,
	register *ir.Node,
	paramName string,
) {
	p.shell.edges = append(p.shell.edges, ir.Edge{
		Source: ir.Handle{Node: register.Key, Param: ir.DefaultOutputParam},
		Target: ir.Handle{Node: p.node.Key, Param: paramName},
		Kind:   ir.EdgeKindContinuous,
	})
	channels, bound := &p.node.Channels, p.shell.channelKey(sym)
	symbol.ResolveInputChannel(channels, p.fnSym, paramName, bound, sym.Name)
	for key, name := range sym.Channels.Read {
		symbol.ResolveInputChannel(channels, p.fnSym, paramName, key, name)
	}
	for key, name := range sym.Channels.Write {
		symbol.ResolveInputChannel(channels, p.fnSym, paramName, key, name)
	}
}

// collectSynthByAST returns a map from each anonymous-body declaration in the tree
// rooted at root to its synth scope, keyed by the declaration's parser node.
func collectSynthByAST(root *symbol.Symbol) map[antlr.ParserRuleContext]*symbol.Symbol {
	m := map[antlr.ParserRuleContext]*symbol.Symbol{}
	var walk func(s *symbol.Symbol)
	walk = func(s *symbol.Symbol) {
		for _, child := range s.Children() {
			if child.AST != nil && strings.HasPrefix(child.Name, ir.AnonymousPrefix) {
				m[child.AST] = child
			}
			walk(child)
		}
	}
	walk(root)
	return m
}

// flowOperatorEdgeKind maps a flow operator to its edge kind:
// `=>` conditional, `->` continuous.
func flowOperatorEdgeKind(op parser.IFlowOperatorContext) ir.EdgeKind {
	if op != nil && op.TRANSITION() != nil {
		return ir.EdgeKindConditional
	}
	return ir.EdgeKindContinuous
}

// routingEntryEdgeKinds returns the edge kind feeding each flow node, aligned to
// entry.AllFlowNode() (`=>` conditional, `->` continuous).
func routingEntryEdgeKinds(entry parser.IRoutingEntryContext) []ir.EdgeKind {
	kinds := make([]ir.EdgeKind, len(entry.AllFlowNode()))
	kinds[0] = ir.EdgeKindContinuous
	for i, op := range entry.AllFlowOperator() {
		kinds[i+1] = flowOperatorEdgeKind(op)
	}
	return kinds
}

func analyzeOutputRoutingTable(
	ctx acontext.Context[parser.IRoutingTableContext],
	sourceNode ir.Node,
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, []ir.Edge, []ir.Member, bool) {
	var (
		nodes       []ir.Node
		edges       []ir.Edge
		bodyMembers []ir.Member
	)

	for _, entry := range ctx.AST.AllRoutingEntry() {
		outputName := entry.RoutingKey().GetText()
		if !sourceNode.Outputs.Has(outputName) {
			ctx.Diagnostics.Add(codes.RoutingName.Atf(
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
		if entry.IDENTIFIER() != nil {
			targetParamName = entry.IDENTIFIER().GetText()
		}

		edgeKinds := routingEntryEdgeKinds(entry)
		sourceOutput := ir.Handle{Node: sourceNode.Key, Param: outputName}
		prevOutputHandle := sourceOutput
		for i, flowNode := range flowNodes {
			isLast := i == len(flowNodes)-1
			isSink := isLast && flowNode.Identifier() != nil

			result, ok := analyzeFlowNode(
				ctx.Child(flowNode), kg, shell, false, isSink,
			)
			if !ok {
				return nil, nil, nil, false
			}

			if result.bodyScope != nil {
				bodyMembers = append(
					bodyMembers,
					ir.Member{Scope: result.bodyScope},
				)
			}

			if result.transition != nil {
				shell.applyTransitionIntent(
					prevOutputHandle, edgeKinds[i], *result.transition,
				)
				continue
			}

			node := result.node
			target := node.input
			if isLast && targetParamName != "" {
				if !node.node.Inputs.Has(targetParamName) {
					ctx.Diagnostics.Add(codes.RoutingName.Atf(
						entry,
						"node '%s' does not have input '%s'",
						node.node.Key,
						targetParamName,
					))
					return nil, nil, nil, false
				}
				target.Param = targetParamName
			}
			edges = append(edges, ir.Edge{
				Source: prevOutputHandle,
				Target: target,
				Kind:   edgeKinds[i],
			})
			for _, arg := range node.args {
				edges = append(edges, ir.Edge{
					Source: prevOutputHandle,
					Target: ir.Handle{Node: arg.Key, Param: ir.DefaultInputParam},
					Kind:   edgeKinds[i],
				})
				nodes = append(nodes, arg)
			}

			if len(node.node.Outputs) > 0 {
				prevOutputHandle = node.output
			}
			if node.node.Key != "" {
				nodes = append(nodes, node.node)
			}
		}
	}

	return nodes, edges, bodyMembers, true
}

// stepInfo collects metadata about a step for computing member keys.
type stepInfo struct {
	key  string
	item parser.IBodyItemContext
}

// collectStepKeys pre-scans a sequence's items to compute their member keys.
// Named stages and nested sequences keep their source-level name; anonymous
// flow and single-invocation steps receive a synthesized "step_N" key.
func collectStepKeys(items []parser.IBodyItemContext) []stepInfo {
	steps := make([]stepInfo, 0, len(items))
	for i, item := range items {
		if item.VariableDeclaration() != nil {
			continue
		}
		key := fmt.Sprintf("step_%d", i)
		if decl := declarationOf(item); decl != nil && decl.IDENTIFIER() != nil {
			key = decl.IDENTIFIER().GetText()
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

// addBodyMembers appends anonymous-body scope members to scope's stratum 0,
// creating the stratum when the scope has none yet.
func addBodyMembers(scope *ir.Scope, members []ir.Member) {
	if len(members) == 0 {
		return
	}
	if len(scope.Strata) == 0 {
		scope.Strata = []ir.Members{members}
		return
	}
	scope.Strata[0] = append(scope.Strata[0], members...)
}

// autoWireTransition advances past a step, or completes a terminal one, when its last
// node emits a value that passes kind, or runs with no outputs.
func autoWireTransition(
	shell *shellBuilder,
	lastNode ir.Node,
	nextMemberKey string,
	kind ir.EdgeKind,
) {
	on := ir.Handle{Node: lastNode.Key, Param: ir.DoneParam}
	if len(lastNode.Outputs) > 0 {
		on.Param = firstOutputParam(lastNode.Outputs)
	} else {
		kind = ir.EdgeKindContinuous
	}
	if nextMemberKey == "" {
		shell.addCompletion(on, kind)
		return
	}
	shell.addTransition(ir.Transition{
		On:     on,
		Kind:   kind,
		Target: ir.Target{Variant: ir.StepTarget{Key: nextMemberKey}},
	})
}

// declaredAsStep reports whether node sits in a step position: the nearest stage or
// sequence that holds it is a sequence.
func declaredAsStep(node antlr.ParserRuleContext) bool {
	for n := node.GetParent(); n != nil; n = n.GetParent() {
		switch n.(type) {
		case parser.IStageDeclarationContext:
			return false
		case parser.ISequenceDeclarationContext:
			return true
		}
	}
	return false
}

// lowerDeclaration lowers the stage or sequence that ctx declares, in place.
func lowerDeclaration[T identifierAST](
	ctx acontext.Context[T],
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, []ir.Node, []ir.Edge, bool) {
	body, ok := resolveDeclScope(ctx)
	if !ok {
		return ir.Scope{}, nil, nil, false
	}
	return lowerBody(ctx, body, kg, shell)
}

// lowerSequence lowers a sequence declaration to a sequential scope. A sequence that
// is a step escalates its completion to enclosing frames.
func lowerSequence(
	ctx acontext.Context[parser.ISequenceDeclarationContext],
	seqScope *symbol.Symbol,
	step bool,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, []ir.Node, []ir.Edge, bool) {
	seqName := seqScope.QualifiedName()
	liveness := ir.LivenessAlways
	if ctx.AST.IDENTIFIER() != nil {
		liveness = ir.LivenessGated
	}
	scope := ir.Scope{
		Key:      seqName,
		Mode:     ir.ScopeModeSequential,
		Liveness: liveness,
	}
	var (
		allNodes []ir.Node
		allEdges []ir.Edge
	)
	ctx = ctx.WithScope(seqScope)
	items := parser.BodyItems(ctx.AST.Body())
	for _, item := range items {
		if decl := item.VariableDeclaration(); decl != nil {
			e, ok := lowerVarDecl(ctx.Child(decl), kg, shell)
			if !ok {
				return ir.Scope{}, nil, nil, false
			}
			if e.init != nil {
				allNodes = append(allNodes, *e.init)
			}
		}
	}
	if vm := scopeVarMembers(seqScope, shell); len(vm) > 0 {
		scope.Strata = []ir.Members{vm}
	}
	steps := collectStepKeys(items)
	memberKeys := make([]string, len(steps))
	for i, s := range steps {
		memberKeys[i] = s.key
	}

	frame := shell.pushSeq(seqName, memberKeys)
	frame.escalatesCompletion = step
	defer shell.popSeq()
	defer shell.ownExits(nil)()

	for i, si := range steps {
		frame.activeIdx = i
		members, nodes, edges, ok := lowerStep(ctx, si, frame.nextMember(), kg, shell)
		if !ok {
			return ir.Scope{}, nil, nil, false
		}
		scope.Steps = append(scope.Steps, members...)
		allNodes = append(allNodes, nodes...)
		allEdges = append(allEdges, edges...)
	}

	scope.Transitions = frame.transitions
	return scope, allNodes, allEdges, true
}

// lowerStep lowers the step si of a sequence, which the step keyed nextKey follows.
func lowerStep(
	ctx acontext.Context[parser.ISequenceDeclarationContext],
	si stepInfo,
	nextKey string,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Members, []ir.Node, []ir.Edge, bool) {
	if decl := declarationOf(si.item); decl != nil {
		scope, nodes, edges, ok := lowerDeclaration(ctx.Child(decl), kg, shell)
		if !ok {
			return nil, nil, nil, false
		}
		// An anonymous step takes the key that the steps before it address.
		if decl.IDENTIFIER() == nil {
			scope.Key = si.key
		}
		return ir.Members{{Scope: &scope}}, nodes, edges, true
	}
	if single := si.item.SingleInvocation(); single != nil {
		if use := useOf(single); flow.UsedBody(ctx, use) != nil {
			scope, ok := lowerUse(ctx.Child(use), flow.UsedBody(ctx, use), kg, shell)
			scope.Key = si.key
			return ir.Members{{Scope: &scope}}, nil, nil, ok
		}
	}
	return lowerLeafStep(ctx, si, nextKey, kg, shell)
}

// lowerLeafStep lowers a flow, assignment, or call step and wires its way out.
func lowerLeafStep(
	ctx acontext.Context[parser.ISequenceDeclarationContext],
	si stepInfo,
	nextKey string,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Members, []ir.Node, []ir.Edge, bool) {
	if flowStmt := si.item.FlowStatement(); flowStmt != nil {
		nodes, edges, bodyMembers, transitionEmitted, ok := analyzeFlow(
			ctx.Child(flowStmt),
			kg,
			shell,
		)
		if !ok {
			return nil, nil, nil, false
		}
		child := flowScope(si.key, nodes)
		addBodyMembers(&child, bodyMembers)
		// Only auto-wire the step-advance transition when the flow did
		// not already terminate in an explicit `=> X`. The explicit
		// transition may target an outer frame; emitting the auto-wire
		// alongside it causes the inner frame's evaluator to clear the
		// firing-node's mark before the outer frame's evaluator sees
		// it, preventing the cross-level jump from firing.
		if !transitionEmitted && len(nodes) > 0 {
			kind := ir.EdgeKindConditional
			flowNodes := flowStmt.AllFlowNode()
			if flowNodes[len(flowNodes)-1].Identifier() != nil {
				ops := flowStmt.AllFlowOperator()
				kind = flowOperatorEdgeKind(ops[len(ops)-1])
			}
			autoWireTransition(shell, nodes[len(nodes)-1], nextKey, kind)
		}
		return ir.Members{{Scope: &child}}, nodes, edges, true
	}
	if assign := si.item.Assignment(); assign != nil {
		aNodes, aEdges, ok := lowerAssignment(ctx.Child(assign), kg, shell)
		if !ok {
			return nil, nil, nil, false
		}
		child := flowScope(si.key, aNodes)
		if len(aNodes) > 0 {
			autoWireTransition(
				shell, aNodes[len(aNodes)-1], nextKey, ir.EdgeKindContinuous,
			)
		}
		return ir.Members{{Scope: &child}}, aNodes, aEdges, true
	}
	if single := si.item.SingleInvocation(); single != nil {
		nodes, ok := analyzeSingleInvocation(ctx.Child(single), kg, shell)
		if !ok {
			return nil, nil, nil, false
		}
		child := flowScope(si.key, nodes)
		autoWireTransition(shell, nodes[len(nodes)-1], nextKey, ir.EdgeKindConditional)
		return ir.Members{{Scope: &child}}, nodes, nil, true
	}
	return nil, nil, nil, true
}

// lowerStage lowers a stage declaration to a parallel scope. A stage that is not a
// step holds the exits that `next` makes.
func lowerStage(
	ctx acontext.Context[parser.IStageDeclarationContext],
	stageScope *symbol.Symbol,
	step bool,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Scope, []ir.Node, []ir.Edge, bool) {
	liveness := ir.LivenessAlways
	if ctx.AST.IDENTIFIER() != nil {
		liveness = ir.LivenessGated
	}
	scope := ir.Scope{
		Key:      stageScope.QualifiedName(),
		Mode:     ir.ScopeModeParallel,
		Liveness: liveness,
	}
	var (
		nodes   []ir.Node
		edges   []ir.Edge
		members []ir.Member
	)

	var exits *[]ir.Transition
	if !step {
		exits = &scope.Transitions
	}
	defer shell.ownExits(exits)()
	ctx = ctx.WithScope(stageScope)
	for _, item := range parser.BodyItems(ctx.AST.Body()) {
		itemMembers, itemNodes, itemEdges, ok := lowerStageItem(ctx, item, kg, shell)
		if !ok {
			return ir.Scope{}, nil, nil, false
		}
		members = append(members, itemMembers...)
		nodes = append(nodes, itemNodes...)
		edges = append(edges, itemEdges...)
	}

	members = append(scopeVarMembers(stageScope, shell), members...)
	if len(members) > 0 {
		scope.Strata = []ir.Members{members}
	}
	return scope, nodes, edges, true
}

// lowerStageItem lowers one item of a stage body to its members, nodes, and edges.
func lowerStageItem(
	ctx acontext.Context[parser.IStageDeclarationContext],
	item parser.IBodyItemContext,
	kg *keyGenerator,
	shell *shellBuilder,
) (ir.Members, []ir.Node, []ir.Edge, bool) {
	var members ir.Members
	if decl := item.VariableDeclaration(); decl != nil {
		e, ok := lowerVarDecl(ctx.Child(decl), kg, shell)
		if !ok || e.init == nil {
			return nil, nil, nil, ok
		}
		return nil, []ir.Node{*e.init}, nil, true
	}
	if flowStmt := item.FlowStatement(); flowStmt != nil {
		nodes, edges, bodyMembers, _, ok := analyzeFlow(
			ctx.Child(flowStmt),
			kg,
			shell,
		)
		if !ok {
			return nil, nil, nil, false
		}
		for _, n := range nodes {
			members = append(members, ir.Member{NodeKey: new(n.Key)})
		}
		return append(members, bodyMembers...), nodes, edges, true
	}
	if assign := item.Assignment(); assign != nil {
		aNodes, aEdges, ok := lowerAssignment(ctx.Child(assign), kg, shell)
		if !ok {
			return nil, nil, nil, false
		}
		for _, n := range aNodes {
			members = append(members, ir.Member{NodeKey: new(n.Key)})
		}
		return members, aNodes, aEdges, true
	}
	if single := item.SingleInvocation(); single != nil {
		if use := useOf(single); flow.UsedBody(ctx, use) != nil {
			scope, ok := lowerUse(ctx.Child(use), flow.UsedBody(ctx, use), kg, shell)
			// A body with no trigger runs while the stage that holds it runs.
			scope.Liveness = ir.LivenessAlways
			return ir.Members{{Scope: &scope}}, nil, nil, ok
		}
		nodes, ok := analyzeSingleInvocation(ctx.Child(single), kg, shell)
		if !ok {
			return nil, nil, nil, false
		}
		for _, n := range nodes {
			members = append(members, ir.Member{NodeKey: new(n.Key)})
		}
		return members, nodes, nil, true
	}
	if decl := declarationOf(item); decl != nil {
		scope, nodes, edges, ok := lowerDeclaration(ctx.Child(decl), kg, shell)
		if !ok {
			return nil, nil, nil, false
		}
		return ir.Members{{Scope: &scope}}, nodes, edges, true
	}
	return nil, nil, nil, true
}

// useOf returns the name or the call of a bare item.
func useOf(single parser.ISingleInvocationContext) antlr.ParserRuleContext {
	if fn := single.Function(); fn != nil {
		return fn
	}
	return single.Expression()
}

// analyzeSingleInvocation lowers a bare call or expression step, with its node last. A
// call fires on the implicit triggers of a flow head.
func analyzeSingleInvocation(
	ctx acontext.Context[parser.ISingleInvocationContext],
	kg *keyGenerator,
	shell *shellBuilder,
) ([]ir.Node, bool) {
	if fn := ctx.AST.Function(); fn != nil {
		triggers, ok := implicitTriggers(ctx, kg, shell, fn)
		if !ok {
			return nil, false
		}
		result, ok := analyzeFunctionNode(ctx.Child(fn), kg, shell, len(triggers) > 0)
		if !ok {
			return nil, false
		}
		targets := []ir.Handle{result.input}
		for _, arg := range result.args {
			targets = append(targets, ir.Handle{
				Node:  arg.Key,
				Param: ir.DefaultInputParam,
			})
		}
		var nodes []ir.Node
		for _, t := range triggers {
			if t.node.Key != "" {
				nodes = append(nodes, t.node)
			}
			for _, target := range targets {
				shell.edges = append(shell.edges, ir.Edge{
					Source: t.output,
					Target: target,
					Kind:   ir.EdgeKindContinuous,
				})
			}
		}
		nodes = append(nodes, result.args...)
		return append(nodes, result.node), true
	}
	if expr := ctx.AST.Expression(); expr != nil {
		result, ok := analyzeExpression(ctx.Child(expr), kg, shell, false)
		if !ok {
			return nil, false
		}
		return []ir.Node{result.node}, true
	}
	return nil, false
}
