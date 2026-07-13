// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package symbol implements symbol table management for the Arc programming language.
//
// A Symbol is the unit of named entity. Variables, functions, modules, channels,
// blocks, loops, sequences, and stages are all Symbols distinguished by Kind. A
// Symbol can be a leaf (no Children) or a container (Children populated). The
// language's lexical scope is exactly the tree of Symbols rooted at the program's
// root Symbol.
//
// Resolution is uniform across all Symbols. Calling Resolve on a container Symbol
// asks "from inside this Symbol, what does this name mean?". The search proceeds:
// children → DynamicResolver (when set) → parent. Dotted names recurse — Resolve
// splits at the first dot, resolves the head, then resolves the tail against the
// head's children.
//
// Sealed namespaces (modules, aliases) have Parent == nil. The walk-up cannot
// escape the seal, which gives modules their member-access semantics for free.
// Transparent containers (functions, blocks, loops) keep Parent set; lexical
// resolution from inside them walks outward as usual.
package symbol

import (
	"context"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// Argument is one call argument: a value expression bound to a param by Name,
// or by position (Index) when Name is empty.
type Argument struct {
	// Index is the argument's position in the call's argument list.
	Index int
	// Name is the param this argument binds to, or empty if positional.
	Name string
	// Expr is the argument's value expression.
	Expr parser.IExpressionContext
	// AST is the argument's parser node, for diagnostic source locations.
	AST antlr.ParserRuleContext
}

// ArgumentsHook runs optional symbol-specific argument validation, reporting any
// findings to the analyzer's diagnostics sink.
type ArgumentsHook func(diags *diagnostics.Diagnostics, args []Argument)

// TriggerBinding declares what an upstream wire does to a symbol in flow context.
type TriggerBinding struct {
	// Target names the param the wire binds its value to, or empty (TriggerOnly)
	// for pure activation.
	Target string
}

// TriggerOnly is the zero TriggerBinding: the wire activates without binding a value.
var TriggerOnly = TriggerBinding{}

// TriggerInput declares that an upstream wire binds its value to the named param.
func TriggerInput(name string) TriggerBinding { return TriggerBinding{Target: name} }

// ExecContext indicates which execution context a symbol is valid in.
type ExecContext int

const (
	// ExecWASM marks a symbol as only usable in func blocks (WASM compilation).
	ExecWASM ExecContext = 1 << iota
	// ExecFlow marks a symbol as only usable in flow statements (graph nodes).
	ExecFlow
	// ExecBoth marks a symbol as usable in both contexts. WASM-form inputs must
	// mirror flow-form inputs one-for-one (N=0 allowed); upstream edges in flow
	// form are triggers, not typed inputs. Invariant enforced in stl_test.go.
	ExecBoth = ExecWASM | ExecFlow
)

// Compatible reports whether a symbol with exec context e is visible in a
// context that filters for target. An untagged symbol (e == 0) is never
// visible. An unset filter (target == 0) shows all tagged symbols.
func (e ExecContext) Compatible(target ExecContext) bool {
	if e == 0 {
		return false
	}
	if target == 0 {
		return true
	}
	return e&target != 0
}

// Kind categorizes symbols by their role in the Arc program.
type Kind int

//go:generate stringer -type=Kind
const (
	// KindVariable represents a regular variable within a function or block scope.
	KindVariable Kind = iota
	// KindStatefulVariable represents a variable that persists across reactive stage executions.
	KindStatefulVariable
	// KindChannel represents a channel type for inter-task communication via unbounded FIFO queues.
	KindChannel
	// KindFunction represents a function declaration.
	KindFunction
	// KindBlock represents a block scope such as a task or stage.
	KindBlock
	// KindInput represents an input parameter to a function or task.
	KindInput
	// KindOutput represents an output parameter from a function or task.
	KindOutput
	// KindSequence represents a sequence (state machine) declaration.
	KindSequence
	// KindStage represents a stage within a sequence.
	KindStage
	// KindConstant represents a pure literal value in a flow statement.
	KindConstant
	// KindLoop represents a loop scope (for break/continue validation).
	KindLoop
	// KindLoopVariable represents an immutable loop iteration variable.
	KindLoopVariable
	// KindModule represents a namespace container (e.g., math, time). Module
	// symbols are sealed: lookup of a member inside a module is restricted
	// to that module's children — the Resolve walk does not continue out
	// through Parent. This is what gives module access its member-only
	// semantics: `math.foo` cannot accidentally find an unrelated `foo` in
	// an enclosing scope.
	KindModule
	// KindModuleAlias represents an import alias bound in a file's root. The
	// Target field points at the underlying module symbol; member lookups on
	// the alias indirect through Target.
	KindModuleAlias
	// KindAmbient represents the prelude scope that sits as Parent of the
	// program root. It holds STL bare globals (deprecated aliases,
	// operators, len, select, ...) and the program root itself. Root()
	// recognizes KindAmbient as the boundary above the user program — it
	// walks to the topmost user-program scope and stops there.
	KindAmbient
)

// VarKind categorizes a value variable by how it is declared, fixing its
// read/write/reassignment semantics. VarKindNone marks a non-value variable.
type VarKind int

const (
	// VarKindNone marks a symbol that is not a value variable.
	VarKindNone VarKind = iota
	// VarKindChannelReadWrite is a variable bound to a bare channel; it behaves
	// exactly as that channel for both reads and writes.
	VarKindChannelReadWrite
	// VarKindChannelRead is a variable derived from an expression reading one or
	// more channels; it is a read-only channel-backed stream.
	VarKindChannelRead
	// VarKindLiteral is a value variable backed by a program-local channel; `:=`
	// literals and `$=` statefuls share it, resetting vs persisting keyed on Kind.
	VarKindLiteral
)

// String returns the user-facing name of the kind.
func (k VarKind) String() string {
	switch k {
	case VarKindChannelReadWrite:
		return "channel read/write"
	case VarKindChannelRead:
		return "channel read"
	case VarKindLiteral:
		return "literal"
	default:
		return "none"
	}
}

// Symbol is a named entity in an Arc program and, when it has Children, a
// container that holds other Symbols. Variables, functions, modules, channels,
// blocks, loops, sequences, stages, and aliases are all Symbols distinguished
// by Kind.
//
// ID Assignment: Symbols whose Kind allocates a runtime slot (KindVariable,
// KindStatefulVariable, KindChannel, KindInput, KindOutput,
// KindLoopVariable) receive a unique ID from the nearest ancestor that owns an
// ID counter (the root and each function or sequence).
type Symbol struct {
	// Type is the symbol's type from Arc's type system.
	Type types.Type
	// AST is the parser node for source location information. Built-in symbols
	// from a resolver or pre-populated module members have AST == nil.
	AST antlr.ParserRuleContext
	// DefaultValue holds the symbol's default or seed value, or nil if it has none:
	// an optional parameter's default, or a literal variable's initial value.
	DefaultValue any
	// Name is the symbol's identifier. Container symbols of anonymous kinds
	// (KindBlock, KindLoop, top-level stages) may have an empty Name.
	Name string
	// Kind categorizes the symbol.
	Kind Kind
	// VarKind categorizes a value variable (Kind KindVariable or
	// KindStatefulVariable) for its read/write semantics; VarKindNone otherwise.
	VarKind VarKind
	// ID is a unique identifier within the containing function scope. Only
	// assigned to symbols whose Kind allocates a runtime slot.
	ID int
	// SourceID tracks the ID of the source symbol for channel type propagation.
	// When a variable or input param holds a channel reference, this field
	// stores the ID of the original source (input param or global channel) so
	// that Channels.Read/Write can be correctly resolved at instantiation time.
	// A nil value means this symbol is the original source (e.g., an input param).
	SourceID *int
	// Internal marks symbols that are only accessible to the compiler (e.g.,
	// WASM host function signatures used for type suffix derivation). Internal
	// symbols are skipped during user-facing Resolve so user code cannot
	// reference them, but remain visible to the compiler's resolve.Resolver
	// which passes IncludeInternal to bypass the filter.
	Internal bool
	// Renameable marks symbols whose backing resource the host can rename via an
	// out-of-band side effect. Source-defined symbols (AST != nil) are renameable
	// by source-text edits alone and do not need this flag; resolver-supplied
	// symbols (e.g., Synnax channels) set it to opt into the LSP rename flow,
	// which dispatches to the host via OnRename. Independent of Internal — the
	// resolver decides the policy for its kind.
	Renameable bool
	// Exec indicates which execution context this symbol is valid in (WASM,
	// Flow, or Both). A zero value is invalid and will cause resolution to
	// fail, forcing every symbol to be explicitly tagged.
	Exec ExecContext
	// AnalyzeArguments runs optional symbol-specific argument validation.
	AnalyzeArguments ArgumentsHook
	// Trigger names the param an upstream wire binds to in flow context; the
	// zero value (TriggerOnly) means pure activation.
	Trigger TriggerBinding
	// Deprecated points at the canonical replacement Symbol for a deprecated
	// reference. Nil means not deprecated. When non-nil, analysis helpers
	// emit a deprecation warning naming Deprecated.QualifiedName(), and the
	// compiler routes the emitted call to Deprecated rather than s.
	Deprecated *Symbol
	// Doc is the user-facing documentation body for this symbol. The LSP
	// hover renderer appends it after the auto-generated title and
	// signature. Contains only the description and examples — the title,
	// kind annotation, and signature are derived from Name / Kind / Type
	// and must not be duplicated here. Empty Doc is rendered as nothing.
	Doc doc.Doc

	// --- Tree structure (zero on leaf symbols, populated on containers) ---

	// Parent is the lexically enclosing symbol. Nil for sealed namespaces
	// (KindModule, KindModuleAlias) and for the topmost scope in a tree.
	// For a program with an ambient prelude, the program root's Parent is
	// the prelude; resolution walks through the prelude via the standard
	// parent walk.
	Parent *Symbol
	// children are the symbols directly contained by this scope. Access
	// goes through the Children() method so KindModuleAlias can
	// transparently delegate to its Target — callers iterating a scope
	// don't need to special-case aliases. The one consumer that needs
	// the alias's own (empty) child list — LSP semantic-token tagging —
	// uses FindChild to obtain the alias and works with its
	// fields directly.
	children []*Symbol
	// counter assigns unique IDs to slot-allocating descendants. Non-nil on
	// the root and on each function and sequence; nil elsewhere.
	counter *int
	// Channels tracks which Synnax channels this symbol's AST node reads
	// from and writes to. Populated for function symbols.
	Channels types.Channels
	// Target is the aliased symbol for KindModuleAlias. Member resolution on
	// an alias indirects through Target.
	Target *Symbol
	// Used reports whether this alias has been consulted by a Resolve call.
	// Drives unused-import diagnostics. Only set on KindModuleAlias symbols;
	// other Kinds leave it false because nothing reads it and writing to
	// shared STL members would race across concurrent NewRoot calls.
	Used bool
	// globalResolver provides on-demand lookups for symbols not pre-materialized
	// in the symbol tree (e.g., global channels). Consulted by Resolve
	// when a name is not found among Children. Set on the root only via NewRoot.
	globalResolver Resolver
}

// NewRoot creates a user-program root attached to an empty ambient
// prelude with ambientGlobals already attached. The optional
// dynamicResolver is consulted for external symbols (e.g., cluster
// channels) when local lookup misses. The ambientGlobals are added
// directly to the ambient prelude as siblings of the root — this is
// where STL symbols, test channels, and custom modules belong.
//
// Each ambient global is shallow-copied before being attached so the
// per-root Parent assignment does not mutate the caller's symbol. This
// matters when a caller reuses the same slice across multiple roots:
// without the copy, the second NewRoot call would re-parent symbols
// away from the first root's ambient.
func NewRoot(dynamicResolver Resolver, ambientGlobals []*Symbol) *Symbol {
	ambient := &Symbol{Kind: KindAmbient}
	root := &Symbol{
		Kind:           KindBlock,
		counter:        new(int),
		globalResolver: dynamicResolver,
	}
	ambient.AddChild(root)
	for _, sym := range ambientGlobals {
		clone := *sym
		ambient.AddChild(&clone)
	}
	return root
}

// GetChildByParserRule finds a direct child with the given AST parser rule.
// Returns an error if no matching child is found.
func (s *Symbol) GetChildByParserRule(rule antlr.ParserRuleContext) (*Symbol, error) {
	res := s.findChild(func(child *Symbol) bool { return child.AST == rule })
	if res == nil {
		return nil, errors.New("could not find symbol matching parser rule")
	}
	return res, nil
}

// Children returns the symbols directly contained by s. A KindModuleAlias
// transparently returns its Target's children, so callers iterating a
// scope's members do not need to special-case aliases. Callers that need
// the alias's own (empty) child slice — LSP semantic-token tagging,
// rename refactors — obtain the alias via FindChild and access its
// fields directly.
func (s *Symbol) Children() []*Symbol {
	if s.Kind == KindModuleAlias && s.Target != nil {
		return s.Target.Children()
	}
	return s.children
}

// FindChild searches for a direct child with the given name. Returns nil
// if no matching child is found.
func (s *Symbol) FindChild(name string) *Symbol {
	return s.findChild(func(child *Symbol) bool { return child.Name == name })
}

func (s *Symbol) findChild(predicate func(*Symbol) bool) *Symbol {
	res, _ := lo.Find(s.children, predicate)
	return res
}

// FilterChildrenByKind returns all direct children of the given kind.
func (s *Symbol) FilterChildrenByKind(kind Kind) []*Symbol {
	return lo.Filter(s.children, func(item *Symbol, _ int) bool {
		return item.Kind == kind
	})
}

// AutoName assigns a unique name by appending a numeric ID from the parent's
// counter. Returns s for method chaining.
func (s *Symbol) AutoName(prefix string) *Symbol {
	idx := s.Parent.addIndex()
	s.Name = prefix + strconv.Itoa(idx)
	return s
}

// maxProgramLocalChannelKey is the exclusive upper bound on top-level variable
// channel keys.
const maxProgramLocalChannelKey = 1 << 20

// Add creates a new child Symbol from sym and appends it to s.children.
//
// If sym has a non-empty Name, Add checks for naming conflicts in the lexical
// chain. Built-in symbols (AST == nil) can be shadowed. Returns an error if a
// locally-defined symbol with the same name already exists.
//
// Functions and sequences receive a new ID counter so their slot IDs are
// independent. Slot-allocating Kinds (variables, channels, params) receive an
// ID from the nearest ancestor counter.
func (s *Symbol) Add(ctx context.Context, sym Symbol) (*Symbol, error) {
	if sym.Name != "" {
		existing, err := s.Resolve(ctx, sym.Name)
		if err == nil && existing.AST != nil {
			tok := existing.AST.GetStart()
			return nil, errors.Newf(
				"name %s conflicts with existing %s at line %d, col %d",
				sym.Name,
				nounForKind(existing.Kind),
				tok.GetLine(),
				tok.GetColumn(),
			)
		}
	}
	child := &Symbol{}
	*child = sym
	child.Parent = s
	if sym.Kind == KindFunction || sym.Kind == KindSequence {
		child.counter = new(int)
	}
	if sym.Kind == KindFunction {
		child.Channels = types.NewChannels()
	}
	switch sym.Kind {
	case KindVariable, KindStatefulVariable:
		if _, err := s.ClosestAncestorOfKind(KindFunction); err != nil {
			child.ID = s.Root().addIndex()
			if child.ID >= maxProgramLocalChannelKey {
				return nil, errors.Newf(
					"program exceeds the maximum of %d program-local channels",
					maxProgramLocalChannelKey,
				)
			}
		} else {
			child.ID = s.addIndex()
		}
	case KindInput, KindOutput, KindLoopVariable:
		child.ID = s.addIndex()
	}
	s.children = append(s.children, child)
	return child, nil
}

// nounForKind returns a human-readable noun for k, used in diagnostics so a
// name conflict names what it collides with (variable, channel, function, ...).
func nounForKind(k Kind) string {
	switch k {
	case KindVariable, KindStatefulVariable, KindLoopVariable:
		return "variable"
	case KindChannel:
		return "channel"
	case KindFunction:
		return "function"
	case KindInput:
		return "input parameter"
	case KindOutput:
		return "output parameter"
	case KindSequence:
		return "sequence"
	case KindStage:
		return "stage"
	case KindConstant:
		return "constant"
	case KindModule, KindModuleAlias:
		return "import"
	default:
		return "symbol"
	}
}

// AddChild appends each already-constructed child to s.children and sets
// its Parent to s. Used by builders (e.g., STL module construction) that
// prepare children before attaching them. Does not check for naming
// conflicts and does not assign IDs; callers are responsible for these
// when constructing tree fragments directly. Returns s so calls can be
// chained.
func (s *Symbol) AddChild(children ...*Symbol) *Symbol {
	for _, child := range children {
		child.Parent = s
		s.children = append(s.children, child)
	}
	return s
}

// AutoImportModules aliases each non-Internal KindModule in root's
// ambient as a KindModuleAlias child of root. Used by graph mode and
// other entry points that reference module-qualified names
// (`control.set_authority`, `time.interval`) without producing `import`
// statements. Internal modules are skipped — they are compiler-only and
// must not become reachable by name from user code. Text-mode callers
// do not call this — the analyzer installs aliases from source-level
// import declarations.
func AutoImportModules(root *Symbol) {
	if root.Parent == nil {
		return
	}
	for _, child := range root.Parent.children {
		if child.Kind != KindModule || child.Internal {
			continue
		}
		alias := &Symbol{
			Name:   child.Name,
			Kind:   KindModuleAlias,
			Target: child,
			Parent: root,
		}
		root.children = append(root.children, alias)
	}
}

// InternalHostFunc builds a KindFunction Symbol for a WASM host shim that
// is hidden from user code. Used by STL packages to declare host bindings
// (channels.read, state.load, etc.) with the common shape factored out:
// every WASM host shim is Kind=KindFunction, Exec=ExecWASM, Internal=true.
func InternalHostFunc(name string, inputs, outputs types.Params) *Symbol {
	return &Symbol{
		Name:     name,
		Kind:     KindFunction,
		Exec:     ExecWASM,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs:  inputs,
			Outputs: outputs,
		}),
	}
}

func (s *Symbol) addIndex() int {
	if s.counter != nil {
		v := *s.counter
		*s.counter++
		return v
	}
	return s.Parent.addIndex()
}

// Root returns the user program's root scope. The walk stops at the
// topmost scope before crossing into a KindAmbient prelude (or at the
// topmost scope if there is no ambient). This is the scope that holds
// the user's imports and the slot-counter for their variables.
func (s *Symbol) Root() *Symbol {
	for {
		if s.Parent == nil || s.Parent.Kind == KindAmbient {
			return s
		}
		s = s.Parent
	}
}

// QualifiedName returns s's canonical dotted name. For a module member
// (s.Parent.Kind == KindModule), returns "<module>.<name>". For a top-level
// symbol, returns s.Name. This is the canonical string identifier used at
// IR/emission boundaries that need a single name token; it never produces
// an alias-prefixed form because the symbol tree itself has already
// resolved aliases via Target indirection.
func (s *Symbol) QualifiedName() string {
	if s.Parent != nil && s.Parent.Kind == KindModule {
		return s.Parent.Name + "." + s.Name
	}
	return s.Name
}

// ResolveOption tunes Resolve's behavior. The zero set of options is the
// user-code view: Internal symbols are hidden, bare module access is
// blocked, and successful lookups mark the alias chain as Used. Options
// flip those filters for callers that need a different view of the tree.
type ResolveOption func(*resolveOpts)

type resolveOpts struct {
	// includeInternal disables the Internal-skipping filter and the bare
	// KindModule filter. Used by the compiler to reach host-function
	// shims and module members that user code is not allowed to see.
	includeInternal bool
	// suppressUsage stops Resolve from recording that a module alias was
	// consulted. Read-only callers operating on an already-analyzed tree
	// set this so the lookup leaves no trace; see WithoutUsageTracking.
	suppressUsage bool
}

// IncludeInternal causes Resolve to walk past the user-visibility filters:
// Internal symbols are returned, and bare module names resolve to their
// module symbol. Aliases are still followed and Used is still recorded.
// Use this from the compiler when resolving host-function shims; do not
// use it from analysis paths that surface symbols to user code.
func IncludeInternal(o *resolveOpts) { o.includeInternal = true }

// WithoutUsageTracking causes Resolve to leave the alias Used flag untouched.
// Marking an alias Used is a write into the symbol tree, which is unsafe when
// the tree is a shared, already-analyzed snapshot read concurrently by other
// callers. Feature queries (hover, rename, definition, completion) run against
// such snapshots and must pass this; the analysis pass that builds the tree and
// reports unused imports must not, since it relies on the usage record.
func WithoutUsageTracking(o *resolveOpts) { o.suppressUsage = true }

// Resolve looks up a single name using lexical scoping rules: children →
// global resolver → parent. The walk stops at a KindModule scope — inside
// a module, lookup cannot escape outward. That seal is what gives module
// members their member-only semantics. By default the walk also skips
// KindModule entries encountered along the way: bare module names are
// not accessible from user code; modules are reached only through
// KindModuleAlias children installed by `import`. Pass IncludeInternal
// to disable both filters.
//
// Resolve does not split dotted names. Member access (`math.abs`) is the
// caller's job: resolve the head against the current scope, follow
// alias.Target if the result is an alias, then resolve the tail against
// the resulting symbol. The grammar already produces the head and tail
// as separate tokens; reconstructing them as `"math.abs"` and splitting
// here would throw that structure away.
func (s *Symbol) Resolve(
	ctx context.Context,
	name string,
	options ...ResolveOption,
) (*Symbol, error) {
	var opts resolveOpts
	for _, o := range options {
		o(&opts)
	}
	return s.resolve(ctx, name, opts, s)
}

// resolve does the actual lexical walk. origin is the scope the caller
// invoked Resolve on; it is propagated unchanged through the parent walk
// so that UndefinedSymbolError reports the original scope (where "did
// you mean" suggestions should be drawn from), not the topmost ancestor.
//
// When name parses as a uint32, the walk matches symbols by ID instead
// of by Name. The Arc lexer forbids identifiers starting with a digit,
// so an all-numeric input is unambiguously an ID reference (graph
// inputs, channel-key literals in source). The branches share the walk
// shape (children → global resolver → parent) so a caller passing either
// kind of key gets the same scoping semantics.
//
// Resolving on a KindModuleAlias transparently dispatches to its
// Target — that is, member lookups (`alias.Resolve("foo")`) walk into
// the module the alias points at. Direct lookups that *find* an alias
// return the alias itself; this keeps duplicate-name detection honest
// (the alias is the thing that conflicts with a second import) and
// lets LSP refactor operations work on alias declaration sites.
func (s *Symbol) resolve(
	ctx context.Context,
	name string,
	opts resolveOpts,
	origin *Symbol,
) (*Symbol, error) {
	if s.Kind == KindModuleAlias && s.Target != nil {
		return s.Target.resolve(ctx, name, opts, origin)
	}
	matches := func(child *Symbol) bool { return child.Name == name }
	if id, err := strconv.ParseUint(name, 10, 32); err == nil {
		matches = func(child *Symbol) bool { return child.ID == int(id) }
	}
	if child := s.findChild(matches); child != nil {
		if opts.includeInternal || (!child.Internal && child.Kind != KindModule) {
			if child.Kind == KindModuleAlias && !opts.suppressUsage {
				child.Used = true
			}
			return child, nil
		}
	}
	if s.globalResolver != nil {
		if sym, err := s.globalResolver.Resolve(ctx, name); err == nil {
			if opts.includeInternal || !sym.Internal {
				return sym, nil
			}
		}
	}
	if s.Kind == KindModule {
		return nil, &UndefinedSymbolError{ctx: ctx, Name: name, scope: s}
	}
	if s.Parent != nil {
		return s.Parent.resolve(ctx, name, opts, origin)
	}
	return nil, &UndefinedSymbolError{ctx: ctx, Name: name, scope: origin}
}

// Search returns symbols matching the given prefix or fuzzy term using the
// same traversal order as Resolve.
func (s *Symbol) Search(ctx context.Context, term string) ([]*Symbol, error) {
	seen := make(set.Set[string])
	var results []*Symbol
	for _, child := range s.children {
		if child.Internal || child.Name == "" {
			continue
		}
		if matchesSearch(child.Name, term) && !seen.Contains(child.Name) {
			results = append(results, child)
			seen.Add(child.Name)
		}
	}
	if s.globalResolver != nil {
		matches, err := s.globalResolver.Search(ctx, term)
		if err == nil {
			for _, sym := range matches {
				if sym.Internal {
					continue
				}
				if !seen.Contains(sym.Name) {
					results = append(results, sym)
					seen.Add(sym.Name)
				}
			}
		}
	}
	if s.Parent != nil {
		parentResults, err := s.Parent.Search(ctx, term)
		if err == nil {
			for _, sym := range parentResults {
				if !seen.Contains(sym.Name) {
					results = append(results, sym)
					seen.Add(sym.Name)
				}
			}
		}
	}
	return results, nil
}

func matchesSearch(name, term string) bool {
	if strings.HasPrefix(name, term) {
		return true
	}
	return len(term) > 2 && compare.LevenshteinDistance(name, term) <= 2
}

// ClosestAncestorOfKind walks up the parent chain (including s itself) and
// returns the nearest symbol with the given Kind. Returns query.ErrNotFound
// when no such ancestor exists.
func (s *Symbol) ClosestAncestorOfKind(kind Kind) (*Symbol, error) {
	if s.Kind == kind {
		return s, nil
	}
	if s.Parent == nil {
		return nil, errors.Wrap(query.ErrNotFound, "undefined symbol")
	}
	return s.Parent.ClosestAncestorOfKind(kind)
}

// ResolveInputChannel replaces an internal input param ID with the actual
// channel ID. For user-defined functions, the analyzer populates fnSym.Channels
// with internal param IDs when processing the function body, so we replace
// those with actual channel IDs. For built-in functions, fnSym has no children
// or channels, so we fall back to adding the channel to Read.
func ResolveInputChannel(
	c *types.Channels,
	fnSym *Symbol,
	paramName string,
	channelKey uint32,
	channelName string,
) {
	replaced := false
	if inputParamSym := fnSym.FindChild(paramName); inputParamSym != nil {
		inputParamID := uint32(inputParamSym.ID)
		if _, ok := fnSym.Channels.Write[inputParamID]; ok {
			delete(c.Write, inputParamID)
			c.Write[channelKey] = channelName
			replaced = true
		}
		if _, ok := fnSym.Channels.Read[inputParamID]; ok {
			delete(c.Read, inputParamID)
			c.Read[channelKey] = channelName
			replaced = true
		}
	}
	if !replaced {
		dir := types.ChanDirectionRead
		if param, ok := fnSym.Type.Inputs.Get(paramName); ok && param.Type.ChanDirection.IsSet() {
			dir = param.Type.ChanDirection
		}
		if dir.IsRead() {
			c.Read[channelKey] = channelName
		}
		if dir.IsWrite() {
			c.Write[channelKey] = channelName
		}
	}
}

// String returns a human-readable string representation of the symbol tree.
func (s *Symbol) String() string { return s.stringWithIndent("") }

func (s *Symbol) stringWithIndent(indent string) string {
	builder := strings.Builder{}
	if s.Name != "" {
		builder.WriteString(indent)
		builder.WriteString("name: ")
		builder.WriteString(s.Name)
		builder.WriteString("\n")
	}
	builder.WriteString(indent)
	builder.WriteString("kind: ")
	builder.WriteString(s.Kind.String())
	builder.WriteString("\n")
	if s.Type.Kind != types.KindInvalid {
		builder.WriteString(indent)
		builder.WriteString("type: ")
		builder.WriteString(s.Type.String())
		builder.WriteString("\n")
	}
	if len(s.children) > 0 {
		builder.WriteString(indent)
		builder.WriteString("children: \n")
		childIndent := indent + "  "
		for _, child := range s.children {
			builder.WriteString(child.stringWithIndent(childIndent))
			builder.WriteString(childIndent)
			builder.WriteString("---\n")
		}
	}
	return builder.String()
}
