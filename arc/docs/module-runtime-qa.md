# Arc Module & Runtime Architecture - Q&A Design Document

This document captures design decisions for Arc's module system runtime infrastructure
through a structured Q&A process. Each section records a question, the answer, and the
resulting design understanding.

The previous document (`module-runtime-architecture.md`) contains the initial
investigation and architectural sketch. This document refines that into a precise,
implementable design.

---

## Q1: What is the actual scope of "modules" here?

**Question:** The word "module" is overloaded. There are at least three things it could
mean:

1. **User-facing language modules** - The `import ( math )` / `math.sqrt(16.0)` syntax
   from `modules.md`. This is about namespaced access to STL functions in Arc source
   code.

2. **Internal reorganization of host function binding** - Moving from the current
   brittle index-based `"env"` WASM import contract to name-based binding with logical
   WASM module namespaces (`"channel"`, `"math"`, `"series"`, etc.).

3. **Unification of the STL definition layer** - Creating a single source of truth for
   STL items (host functions, reactive nodes, symbols) instead of the current scattered
   approach across `compiler/bindings/`, `runtime/wasm/bindings/`, and individual
   `runtime/` packages.

Are all three of these in scope for this effort? Or is this specifically about the
runtime/binding infrastructure (2 and 3) that must exist _before_ the user-facing module
syntax (1) can be implemented?

**Answer:** #2 and #3 are the most important. #1 is not in scope, but the design must
not close off its future implementation.

**Understanding:** This effort is about fixing the internal plumbing - the host function
binding contract and STL definition layer - not about adding user-facing `import` syntax
yet. The deliverable is a clean, unified infrastructure where STL items are defined once
and consumed by the analyzer, compiler, Go runtime, and C++ runtime. The WASM import
contract moves from positional indices under `"env"` to name-based binding under logical
module namespaces. The design must be structured so that when `import ( math )` syntax
is added later, the underlying `math` module already exists as a first-class entity that
the parser/analyzer can reference.

---

## Q2: Should host functions and reactive nodes be structurally unified?

**Question:** There are two fundamentally different kinds of STL items:

- **WASM host functions** (channel reads, series ops, math.pow, now) - called via `call`
  instructions inside compiled WASM function bodies.
- **Reactive graph nodes** (set_authority, on, write, select, stable_for, constant,
  interval, wait) - Go-native nodes in the dataflow graph, never touch WASM.

These have completely different lifecycles. Should the unification be about co-location
(same package structure, shared registry) or structural merging (a single type
represents both)?

**Answer:** Co-location, not structural merging. They don't need a single type. The goal
is consistent resolution paths and living in the same package structure.

**Understanding:** `HostFunc` and reactive node definitions remain separate types - no
forced polymorphism. But they share a common organizational scheme (e.g., both live
under a module namespace like `math` or `control`) and both feed into symbol resolution
through the same pipeline. A `math` module might only have host functions. A `control`
module might only have reactive nodes. A future module could have both. The registry
provides a consistent way to discover and resolve symbols regardless of which kind they
are, but doesn't pretend they're the same thing at the implementation level.

---

## Q3: Where do compiler primitives live?

**Question:** There's a third category: compiler primitives - host functions users never
call directly. The compiler emits them implicitly for language constructs
(`channel_read_f64`, `state_store_i32`, `series_element_add_f64`,
`string_from_literal`). They have no user-visible symbol but still need to exist in the
WASM import contract and both runtimes need to provide them. Should they live in the
same module structure as user-facing STL items (just with no exported symbol), or be
treated separately?

**Answer:** Ideally fewer separate patterns is better. If they fit, keep them in the
same structure. If they really don't fit, they can be separate. Also need to think
carefully about how runtime state is stored (stateful variables, channel reads/writes,
etc.).

**Understanding:** The preference is to minimize the number of distinct organizational
patterns. Compiler primitives should try to fit into the same module structure as
user-facing items - just without a user-visible symbol. A `channel` module would contain
`read_f64`, `write_f64` etc. as host functions with no analyzer symbol, alongside
(potentially) future user-facing channel utilities. This keeps the WASM import contract
consistent: everything is `import("module_name", "function_name")` regardless of whether
users can call it directly.

The open question flagged here is about runtime state management. Compiler primitives
like `channel_read`, `state_load`, `state_store` are special because they interact with
the `state.State` struct - they're not pure functions. The host function implementations
need access to shared mutable state (the channel read/write buffers, the stateful
variable store, the series handle store). This is a design constraint on how these
functions are organized - they can't just be standalone function pointers, they need a
reference to the runtime state. This needs further exploration in subsequent questions.

---

## Q4: What's the relationship to the C++ runtime?

**Question:** The C++ runtime mirrors Go's host functions exactly with positional
binding. Is the C++ runtime updated as part of this work, or is the deliverable a new
WASM import contract that C++ migrates to later? Are there C++ constraints that
influence the design?

**Answer:** The C++ side doesn't need to be updated in the first step and doesn't need a
full design yet, but the contract must be sustainable for C++ best practices. C++ has
templates (analogous to Go generics for type variants). Need to research wasmtime vs
wazero API differences to ensure the contract works for both. Critical constraint: avoid
runtime type switching where possible.

**Understanding:** The Go side is the first deliverable. The C++ side consumes the WASM
binary's import section as its contract - if we get the names and signatures right, the
C++ side can independently adopt name-based binding via wasmtime's `Linker` API whenever
it's ready. The design constraint is that the WASM import contract (module names,
function names, signatures) must be clean enough that C++ can implement it idiomatically
with templates, not with giant switch statements or positional vectors.

The "avoid runtime type switching" constraint is important for both sides. Currently the
Go runtime has ~4,900 lines of generated code that's essentially a giant type switch
(one method per type per operation). The replacement should use Go generics to get
compile-time type specialization. Similarly, C++ should be able to use templates. The
WASM import naming convention (e.g., `read_f64`, `read_i32`) makes the type explicit in
the function name, which means each registered function is already type-specialized - no
runtime dispatch needed at the host function level.

---

## Q5: Is eliminating the code generator a hard requirement?

**Question:** Go generics could replace ~10,000 lines of generated code. But each
type-specialized host function still needs individual registration with the WASM
runtime. Is eliminating the code generator a hard requirement, or is it acceptable to
keep code generation if the generator becomes significantly simpler?

**Answer:** The code generator's complexity is not the problem. A sophisticated, complex
code generator is totally fine if it's abstracted from the developer experience. What
matters is that it's simple and clear to write a new STL module - the developer needs to
easily understand how their module is resolved in the analyzer, called in the compiler,
and executed in the runtime. If we need complex code generation under the hood to make
that authoring experience clean, that's completely acceptable.

**Understanding:** The optimization target is **STL module authoring ergonomics**, not
minimizing generated code. The developer writing a new `math` module or adding a
`math.sqrt` function should have a clear, simple mental model:

1. Define the function's signature and implementation in one place
2. Understand that the analyzer will make it available for type checking
3. Understand that the compiler will emit the right WASM import call
4. Understand that both Go and C++ runtimes will bind the implementation

If a code generator handles the mechanical expansion of that simple definition into the
hundreds of concrete type-specialized registrations, that's fine. The generator is
infrastructure, not interface. The interface is the module definition API.

---

## Q6: What is the real measure of success?

**Question:** What should the ideal authoring experience look like for adding a new STL
function like `math.sqrt`? Is defining that experience what this process should produce?

**Answer:** It's not only about how easy it is to add `math.sqrt`. It's about clarity in
understanding the interplay between runtime implementations, host function bindings, and
analyzer resolution. The goal is making it easy to reason about the role of the STL,
compiler, and WASM binding architecture in the toolchain as a whole.

**Understanding:** The success criteria is **architectural legibility**, not just
ergonomic convenience. A developer looking at the system should be able to answer
questions like:

- "Where does the analyzer get the type signature for `math.sqrt`?"
- "How does the compiler know what WASM import to emit for a call to `math.sqrt`?"
- "Where is the Go implementation that runs when WASM calls `math.sqrt`?"
- "What's the contract between the compiled WASM binary and the C++ runtime?"
- "How does a reactive node like `set_authority` differ from a WASM host function like
  `math.sqrt` in terms of resolution and execution?"

These questions should have obvious, traceable answers - not because there's a single
file, but because the architecture has clear boundaries and consistent patterns. The
current system fails this test because responsibilities are scattered: the import index
is in `compiler/bindings/`, the implementations are in `runtime/wasm/bindings/`, the
symbols are in individual `runtime/` packages, and the wiring is in generated glue code.
There's no clear mental model connecting them.

The new design should make the flow from "definition" to "analyzer resolution" to
"compiler emission" to "runtime execution" traceable and consistent, whether you're
looking at a compiler primitive, a user-callable host function, or a reactive graph
node.

---

## Q7: What happens to the `state.State` god object?

**Question:** Currently `state.State` holds everything: channel buffers, stateful
variable persistence, series handles, string handles, authority changes, node outputs.
Every host function reaches into it. When we reorganize into modules, does each module
get its own slice of state, or does the single `State` remain the shared backing store?
Is `State` part of the cleanup or an implementation detail that stays as-is?

**Answer:** Each module might get its own slice of state. Important consideration:
debuggability. When we implement a debugger for Arc, how easy is it to query program
state? Monolithic god objects are probably wrong, but an inability to track where state
exists is also bad.

**Understanding:** The `State` struct needs to be decomposed, but into a structure that
remains **discoverable**. The design tension is:

- **Against monolith:** A single struct with 10+ responsibilities makes it hard to
  reason about what state a particular module can touch. It's also a debugger
  nightmare - you can't ask "what state does the `channel` subsystem own?" because it's
  all mixed together.

- **Against full fragmentation:** If every module owns its own private state with no
  common structure, a debugger can't enumerate "all program state." You'd need to know
  every module's internal layout to inspect anything.

The likely shape is something like: each module owns a **typed slice of state** (channel
module owns channel buffers, state module owns variable persistence, series module owns
the handle store), but these slices are registered with a central **state registry**
that a debugger can enumerate. Think of it like a filing cabinet: each drawer (module)
owns its contents, but you can open the cabinet and see all the drawers.

This also affects host function construction: instead of every host function taking a
`*state.State` and reaching into arbitrary parts of it, a `channel.read_f64`
implementation would receive only the channel state slice. This makes the dependency
explicit - you can see exactly what state a module needs by looking at what it's
constructed with.

Open questions to resolve later:

- What's the interface between the scheduler/flush cycle and these state slices?
- How do cross-cutting concerns work (e.g., series handles are used by channel reads,
  series ops, AND state operations)?
- Does the debugger need a runtime query protocol, or is static analysis of the module
  registry sufficient?

---

## Q8: Where does the series/string handle store live?

**Question:** The series/string handle store is a cross-cutting resource used by channel
reads, series ops, state operations, and string operations. It doesn't belong to any one
module. Is it a shared service that modules depend on, or part of a "core" module that
other modules import?

**Answer:** Those are the same thing - what's the difference?

**Understanding:** Fair point. I was drawing a distinction that doesn't actually exist.
Whether you call it "a shared service injected into modules" or "a core module that
other modules reference," the result is the same: there's a handle store, modules that
need it get a reference to it, and it's owned by the runtime rather than by any
individual STL module.

The real design question is simpler than I made it: **the handle store is runtime
infrastructure, not an STL module concern.** It's analogous to WASM linear memory - it's
a resource the runtime provides that host functions use, not something defined at the
STL layer. When constructing a module's host function implementations, the runtime
passes in the handle store (along with whatever other state slices that module needs).
The module doesn't know or care how the handle store is organized internally.

So the dependency direction is: Runtime owns handle store -> Runtime constructs module
host functions with handle store reference -> Module implementations use it.

---

## Q9: Does the graph (visual editor) path need module awareness?

**Question:** Arc has two compilation paths: text (source code) and graph (visual
editor). Both produce the same IR + WASM output. When the module system is in place,
does the graph path need to be module-aware too? Or can it keep using flat symbol names?

**Answer:** Yes, absolutely the graph path needs to be module aware.

**Understanding:** Both paths must resolve symbols through the same module-aware
infrastructure. This means:

- The symbol resolver used by both paths needs to understand module namespaces. When the
  graph path encounters a node with type `"math.sqrt"`, it resolves through the same
  module registry as the text path would for `math.sqrt(x)`.

- The IR node `Type` field (currently flat strings like `"set_authority"`, `"ge"`,
  `"on"`) will need to carry module-qualified names, or the resolver needs to handle
  both qualified and unqualified lookups for backward compatibility.

- The visual editor's node palette in the Console will need to present STL functions
  organized by module. Instead of a flat list of all available nodes, users would see
  `math > sqrt, pow, abs` and `control > set_authority, stable_for, select`.

This reinforces that the module system is not just a text-syntax feature bolted on top -
it's a fundamental organizational unit of the STL that both compilation paths consume.
The symbol resolver, the IR, and the runtime factory lookup all need to speak the same
module-qualified language.

---

## Q10: How does module-qualified resolution coexist with unqualified names?

**Question:** Currently resolution is flat: bare name -> Symbol. With modules,
`math.sqrt` has a qualifier and a member. But unqualified names like `set_authority`,
`ge`, `on`, `write` still need to work in flow statements and graph nodes. Should all
STL symbols be module-qualified internally (with some "auto-imported"), or should there
be two tiers?

**Answer:** Not sure of the exact answer. The unfortunate reality is that we need
backwards compatibility. Maybe "auto-import" is the right approach, or maybe we bind an
alias to a module with an empty string name `""` to represent the global/unqualified
namespace.

**Understanding:** There's a backwards compatibility constraint that rules out
"everything must be qualified." Existing Arc programs and graph definitions use bare
names like `set_authority`, `ge`, `on`, `write`, and these must continue to work.

Two viable approaches:

1. **Auto-import / prelude:** Every STL symbol internally belongs to a module
   (`control.set_authority`, `op.ge`, `telem.on`), but certain modules are
   "auto-imported" into every program's scope. The resolver sees `set_authority`, checks
   the auto-imported modules, finds it in `control`. This is how Rust's prelude and Go's
   `builtin` package work.

2. **Empty-string module:** Symbols that should be globally available are registered
   under a module with name `""`. The resolver first checks the empty module, then
   checks qualified modules. This is mechanically simpler but semantically muddier.

Both need more thought. The key constraint is: internally, every symbol should have a
canonical module-qualified name (`control.set_authority`) for consistency in the IR,
factory lookup, and debugging. The question of whether users _write_ the qualified name
or get it auto-resolved is a separate, surface-level concern.

This is a decision we can defer slightly - the module registry can support both
qualified and unqualified lookup regardless of which user-facing policy we choose.

---

## Q11: Should host function calls and user-defined function calls be unified?

**Question:** Currently the compiler has two separate paths for function calls:
user-defined functions go through `ctx.FunctionIndices[funcName]`, host functions go
through `ctx.Imports.GetXxx()`. Should `math.sqrt` (host) and `myHelper()`
(user-defined) use the same lookup mechanism?

**Answer:** Ideally unified, but not sure if that's sustainable. Also need to consider
how this plays with importing from other compiled modules - i.e., user-defined modules.

**Understanding:** There are actually three kinds of function calls the compiler will
eventually need to handle:

1. **Host functions** - implemented in Go/C++, called via WASM imports (e.g.,
   `math.sqrt`, `channel_read_f64`)
2. **Local user-defined functions** - compiled to WASM in the same module, called via
   WASM function index (e.g., `myHelper()` defined in the same program)
3. **Imported user-defined functions** (future) - compiled to WASM in a _different_
   module, need some form of cross-module linking

All three ultimately become a WASM `call` instruction with a function index. The
difference is how that index is obtained:

- Host functions: registered as WASM imports, index comes from the import section
- Local functions: defined in the same module, index = import count + function offset
- Cross-module (future): could be WASM imports pointing to another WASM module, or could
  use WASM's `call_indirect` mechanism

A unified lookup would mean: the compiler sees a function call, resolves the symbol, and
gets back a function index regardless of where the function lives. The symbol carries
enough metadata to tell the compiler "this is an import" vs "this is local" vs "this is
from another module," and the index is resolved through a single `FunctionIndices` map
that's populated from all three sources.

This is worth pursuing but the cross-module case adds real complexity. For now, unifying
host functions and local functions into a single lookup seems tractable. The
cross-module case can extend the same pattern later (it's just another source of entries
in the function index map).

Key risk: host functions need lazy registration (the "only-imported" optimization from
the previous doc - don't import `series_element_add_f64` if the program never uses
series addition). User-defined functions are always present. A unified lookup needs to
handle this asymmetry.

### Q11 follow-up: Compiler function resolution (revised after Q21 discussion)

The compiler has two distinct resolution concerns that use **separate interfaces backed
by the same module data**:

1. **"What is this thing?"** → `ctx.Scope.Resolve()` (analyzer's scope). Returns a
   Symbol with type info and kind. Already exists, stays as-is.

2. **"What WASM function index do I call?"** → New `ctx.Resolve(name) -> uint32`.
   Replaces both `FunctionIndices` and `Imports`. Handles all cases uniformly:
   - User-defined local functions → local function index
   - Host functions → registers WASM import if needed, returns import index
   - Compiler primitives → same as host functions (e.g.,
     `ctx.Resolve("channel.read_f64")`)
   - Future cross-module user functions → either inlined (local) or imported

The WASM `call` instruction doesn't care what's behind the index. The compiler shouldn't
either. Modules populate both the analyzer scope and the compiler resolver from the same
definitions. Open question: whether user-defined module functions are inlined into the
WASM binary (becoming local functions) or cross-linked as WASM imports. Either way, the
compiler's `Resolve` returns an index and the call site doesn't care.

---

## Q12: Should symbol definitions and factory wiring be co-located?

**Question:** Right now, adding a new reactive node requires two separate steps in two
separate places:

1. Define the symbol + factory in its own package (e.g.,
   `runtime/authority/authority.go`)
2. Manually wire it into both the `MultiFactory` in
   `core/pkg/service/arc/runtime/task.go` AND the `CompoundResolver` in
   `core/pkg/service/arc/symbol/resolver.go`

The definition is co-located (good), but the wiring is in a different package and split
across two lists that must stay in sync (bad). Should each module package instead expose
a single struct containing both its symbols and its factories, so the assembly point
iterates one list of modules instead of independently maintaining two?

**Answer:** Yes, one list is good.

**Understanding:** Each STL module package exports a single entry point that provides
everything the system needs from that module: symbols (for the analyzer), factories (for
the runtime), and host function definitions (for the compiler/WASM binding). The
assembly point becomes a single list of modules:

```
modules := []Module{channel, series, state, math, control, time, op, ...}
```

From this one list, the system derives:

- The symbol resolver (iterate modules, collect all exported symbols)
- The node factory chain (iterate modules, collect all node factories)
- The WASM host function registry (iterate modules, collect all host functions)

One list, three consumers. Adding a new module means defining it in one package and
adding one line to the module list. No forgetting to wire up the symbol resolver
separately from the factory.

---

## Q24: Who owns state, and where does it live?

**Question:** If state is injected into modules, you can't ask "where does channel state
live?" If modules own state, `core/` needs access to populate/flush. All state is
meaningful (including series and strings) - a debugger needs to inspect everything.

**Proposal:** Each module defines its own state type, creates it internally, and returns
a handle to it alongside the module definition. Host function implementations close over
the module's state. Cross-cutting dependencies are explicit constructor parameters.

```go
seriesMod, seriesStore := series.NewModule()
stringMod, stringStore := string.NewModule()
channelMod, channelBufs := channel.NewModule(seriesStore, stringStore)
stateMod, stateVars := state.NewModule(seriesStore)
authMod, authLevels := authority.NewModule()
mathMod := math.NewModule()  // pure, no state
statusMod := status.NewModule(statusService)  // core/-only

modules := []Module{seriesMod, stringMod, channelMod, ...}
```

**Properties:**

- State ownership is unambiguous: each module creates and owns its state
- Dependencies are explicit in constructor signatures
- Construction order is enforced by Go's type system
- Cross-cutting access (channel → series store) is declared, not hidden
- Debugger iterates modules, each exposes inspection on its state handle
- `core/` uses returned state handles for I/O (populate reads, flush writes)

**Answer:** Right direction. Inter-module dependency is OK. Not all the way there yet -
needs further refinement.

### Q25 follow-up: Module as stateful object vs two return values

Prefer single return. The module IS the stateful object. It satisfies the Module
interface (symbols, host funcs, factories) AND exposes module-specific state access via
type assertion or module-specific methods. Split into two returns only if we really feel
the need later.

```go
channelMod := channel.NewModule(seriesStore, stringStore)
// channelMod satisfies Module interface
// channelMod.Reads() / channelMod.Writes() for core/ access
// Type assert when you need module-specific access
```

---

## Q13: Should graph nodes and host functions for the same operation share a module?

**Question:** The `op` package provides reactive graph nodes for element-wise series
operations (`ge`, `add`, etc.) - Go structs called by the scheduler. Separately, there
are series arithmetic/comparison host functions (`series_element_add_f64`,
`series_compare_gt_scalar_f64`) called from inside WASM when operating on series-typed
variables. Both do the same math on series, but with completely different interfaces
(scheduler ports vs handle-based WASM calls). Should they live in the same module?

**Answer:** Yes, they should be in the same module. Both are doing comparisons on series
or adds on series.

**Understanding:** A module is organized around **what it does semantically**, not
around which execution layer it runs in. An `op` (or `series` or `math`) module would
contain:

- The reactive graph node factory for `add` (Go struct, series in/series out)
- The WASM host function definitions for `series_element_add_f64`,
  `series_series_add_f64` etc. (handle-based, called from compiled code)
- The symbol definition that the analyzer uses for type checking

This means a single module package can provide all three kinds of artifacts from Q2:
graph node factories, host function definitions, and analyzer symbols. They're
co-located because they're the same logical operation, even though they have different
runtime interfaces.

This also means the module struct from Q12 needs to accommodate all three:

```
Module {
    Symbols       -> for analyzer
    NodeFactories -> for scheduler/reactive layer
    HostFunctions -> for WASM binding/compiler
}
```

---

## Q14: Should the compiler's type-based dispatch be driven by module definitions?

**Question:** When the compiler sees `a + b`, it checks types: both scalars → native
WASM instruction, one/both series → host function call, both strings → `string_concat`.
This dispatch is hardcoded in the compiler's expression codegen with knowledge of host
function naming conventions. Should the module definition describe this dispatch logic,
or should it stay hardcoded in the compiler?

**Answer:** It's fine to hardcode these in the compiler. It's fine for the compiler to
know these names hardcoded.

**Understanding:** The compiler is allowed to have intimate knowledge of compiler
primitives. The module definition doesn't need to be a fully generic "teach the compiler
how to emit code" system. The compiler knows that `+` on two series means
`series_series_add_{type}`, and it constructs that name directly. The module definition
ensures the host function _exists_ under that name and provides the implementation, but
the compiler's codegen for operators remains specialized.

This keeps the module definition simple: it declares "here is a host function called
`series_add_f64` with this signature and this implementation." The compiler
independently knows "when I see series + series of type f64, I emit a call to
`series_add_f64` in the `op` module." The contract is the name and signature, not a
declarative dispatch table.

---

## Q15: Is the "only-imported" optimization important?

**Question:** Currently all ~390 host functions are imported into every WASM binary. A
lazy tracker could import only the ones actually used. Is this optimization important,
or is the overhead of importing everything negligible?

**Answer:** Not the critical problem to solve, but if it's easy to support, it's nice.

**Understanding:** Don't design around this optimization, but don't prevent it either.
With name-based binding, both wazero and wasmtime resolve imports by name - the runtime
registers all available host functions, and only the ones the WASM binary actually
imports get resolved. Unused functions in the runtime are harmless. So the
"only-imported" optimization is about WASM binary size (fewer import entries), not
runtime cost.

The practical approach: start by importing everything (simpler compiler), and if binary
size becomes a concern later, add lazy tracking as a compiler optimization. Name-based
binding makes this possible without architectural changes.

---

## Q16: Where do modules that depend on server services live?

**Question:** Several reactive nodes depend on Synnax server infrastructure outside
`arc/`: `on`/`write` (I/O buffers from `dataRuntime`), `set_authority` (flushed by
`core/`), `set_status` (lives entirely in `core/`, depends on `status.Service`). Should
modules that need server services live in `core/` as "server-provided modules," or
should all module definitions live in `arc/` with dependencies injected?

**Answer:** They should live in `core/`. The core is the controlling process - it
already does the job of setting up the symbol resolver, the runtime factory, etc. It
gets to dependency inject all of the modules it supports. This allows for composability
and extension instead of a hardcoded STL in `arc/`.

**Understanding:** This is a fundamental architectural decision. `arc/` is a **language
library** - it provides the parser, analyzer, compiler, WASM runtime, scheduler, and the
module interface/contract. It does NOT own the STL.

`core/` (or any other host process) is what **composes modules** and injects them into
the `arc/` pipeline. The core assembles the list of modules, each providing its symbols,
factories, and host functions. Some modules might be pure (like `math` - no server
dependencies, could live in `arc/`). Others need server services (like `set_status`,
`on`/`write`). The core can provide both.

This means:

- `arc/` defines the `Module` interface/types and the machinery that consumes them
- `core/` (or any host) instantiates concrete modules with their dependencies and passes
  them to the `arc/` pipeline
- There's no hardcoded STL list inside `arc/` - the set of available modules is entirely
  determined by the host process
- The C++ driver could provide a different set of modules than the Go server (e.g.,
  hardware-specific modules)
- Third parties could extend Arc with custom modules without forking `arc/`

This is dependency injection at the module system level. `arc/` provides the socket, the
host plugs in modules.

---

## Q22: Where does the split between arc/ and core/ modules land?

**Question:** Should `arc/` ship with any modules, or be a pure framework?

**Answer:** Nodes and STL definitions live at the lowest level possible. If they don't
need server services, they should live in `arc/`. The MAJORITY of the STL should live in
`arc/`.

**Understanding:** `arc/` ships with every module that has no external dependencies:

**In `arc/` (language-intrinsic, no server deps):**

- Operators (`add`, `ge`, `gt`, etc.) + series arithmetic/comparison host functions
- `constant`, `select`, `stable_for`, `stage_entry`
- `interval`, `wait`
- `math` (pow, future sqrt/sin/cos/abs)
- `time` (now)
- `string` operations
- `series` operations (create, index, slice, len)
- `channel` read/write host functions (compiler primitives - the host function
  implementations operate on state that's injected, not on server services)
- `state` load/store host functions (same - operate on injected state)
- `stat` (avg, min, max)
- Error/panic

**In `core/` (needs Synnax server services):**

- `set_status` (needs `status.Service`)

**Correction from Q23 discussion:** `on`/`write` and `set_authority` were initially
assumed to need `core/`, but they don't. They live in `arc/` and only depend on
`state.State`. The server-specific work (populating channel buffers from Synnax
streamers, flushing writes/authority changes to Synnax writers) happens in `core/`'s
`dataRuntime` OUTSIDE the nodes. The nodes themselves are pure `arc/`.

`core/` assembles the full module list: all of `arc/`'s built-in modules plus its own
server-specific modules (currently just `set_status`). The C++ driver would do the same
with its own set.

---

## Q17: What should the Module interface look like?

**Question:** A Module needs to provide symbols (for analysis), WASM import
names/signatures (for compilation), and Go implementations + node factories (for
runtime). Should this be a single interface with methods, a flat struct, or phased?

**Answer:** Don't know yet. It depends on what the actual needs are at the end. If you
need runtime dependencies then you can't statically define everything. Every single
field, method, and interface that a module needs to implement should be painstakingly
thought through and justified.

**Understanding:** This is the central design artifact of the whole effort, and it can't
be designed in the abstract. The Module contract must emerge from concrete requirements,
not from speculation. Key tension identified:

- Some module data is **static** (symbol names, types, WASM import signatures) - known
  at package init time, doesn't depend on runtime state.
- Some module data is **dynamic** (host function implementations that close over
  `state.State`, node factories that need `status.Service`) - can only be constructed
  when the runtime is being set up.

This means the Module contract probably can't be a single static struct. But it also
shouldn't be an over-abstracted interface with 10 methods. The exact shape needs to be
derived bottom-up from the actual consumers:

1. What exactly does the analyzer need from a module? (Just symbols? Or also type
   checking hooks?)
2. What exactly does the compiler need? (Just import names + signatures? Or also codegen
   hints?)
3. What exactly does the Go runtime need? (Host function closures over state? Node
   factories with injected services?)
4. What exactly does the C++ runtime need? (Nothing directly - it reads the WASM
   binary's import section.)

Each field/method must be justified by a concrete consumer. This question stays open and
will be answered by the design process itself.

---

## Q18: Can a module member be just a host function, just a graph node, or both?

**Question:** `now()` is currently WASM-only (no graph node). Under the module system,
should a module member like `time.now` be allowed to provide only a host function, only
a graph node, or both? Who decides?

**Answer:** It should be the module author's choice. A member might be just a host
function, or it might be both a host function and a graph node. The designer of the
module should be able to choose.

**Understanding:** The module contract must support **optional provision** of each
artifact type. A module member doesn't have to provide all three (symbol, host function,
graph node). Concretely:

- `time.now` might provide: symbol + host function only (WASM-callable, no graph node)
- `time.now` might provide: symbol + host function + graph node (works in both layers)
- `set_authority` provides: symbol + graph node only (no WASM host function)
- `series_element_add_f64` provides: internal symbol + host function (no graph node -
  compiler primitive; symbol has Internal flag, not user-visible)

The module contract must not force all three. Each is independently optional. This is
another argument against a single unified type for "module member" - the combinations
are too varied. The module provides collections of each artifact type, and a given
logical operation might appear in one, two, or all three collections.

---

## Q19: Should WASM module names match user-facing module names?

**Question:** WASM import module names serve as the contract between compiler and
runtime. User-facing module names are what appears after `import`. Should they be the
same string, or independent?

**Recommendation:** Independent, because:

1. Compiler primitives (`channel_read_f64`) have no user-facing module - there's no
   `import ( channel )`. Forcing a match creates phantom modules.
2. Granularity can differ - user sees `math` as one thing, WASM level might organize
   differently.
3. C++ runtime only sees WASM names, doesn't care about user-facing names.
4. Future user-defined modules need their own WASM namespace without colliding with STL.

**Answer:** Overall fine with independence. The WASM namespace should default to `Name`
with an optional `*string` override, not a separate required field.

**Understanding:** The module definition has a `Name` field used for user-facing symbol
resolution. WASM namespace defaults to `Name`. When they need to differ (compiler
primitives, or any case where the WASM contract should be different from the user-facing
name), an optional override is provided. This means:

- `math` module: `Name: "math"`, WASM namespace defaults to `"math"`, no override needed
- `channel` primitives: `Name: ""` (not user-facing), WASM override: `"channel"`
- Most modules: one field, not two

This avoids both the coupling problem (forced matching) and the redundancy problem
(always specifying two names that are usually identical).

---

## Q20: How do host functions work as graph nodes?

**Question:** When `math.sqrt{}` is used in a flow statement, the analyzer creates an IR
node with `Type: "math.sqrt"`. At runtime, the `MultiFactory` tries to create a node for
it. The WASM factory only handles compiled user-defined functions (it looks up
`cfg.Module.Functions.Find(cfg.Node.Type)`). `math.sqrt` is a host function with no
compiled WASM body, so the WASM factory returns `ErrNotFound`.

For `math.sqrt` to work in a flow, it needs either a dedicated Go graph node factory
(like `set_authority` has) or a compiler-generated thin WASM wrapper. Which approach?

**Answer:** Dedicated Go graph node factory. Thin WASM wrapper is stupid.

**Understanding:** If a module author wants a host function to also be usable as a graph
node, they provide a Go graph node factory for it. This is the same pattern as
`set_authority`, `select`, `stable_for` - all Go-native graph nodes.

This means for `math.sqrt` to work in both contexts:

- The module provides a host function definition (for WASM calls in function bodies)
- The module provides a graph node factory (for use in flow statements)
- Both implementations do the same math but through different interfaces

This is not duplication in a bad sense - the graph node operates on series via
`state.Node` ports, the host function operates on scalar/handle values via WASM params.
They're genuinely different code paths that happen to compute the same thing.

It also means not every host function automatically becomes a graph node. The module
author explicitly opts in by providing a factory. This aligns with Q18 - it's the module
author's choice.

---

## Q26: The Module interface - evolving design

### First attempt (rejected)

```go
type Module interface {
    Name() string
    WASMNamespace() string
    Symbols() []symbol.Symbol
    HostFuncs() []HostFunc  // HostFunc had wasm.FunctionType
    NodeFactory() node.Factory
}
```

Problems identified:

- Module should BE the factory, not contain one
- Module should BE a resolver, not return a list of symbols
- `wasm.FunctionType` is redundant - derivable from Arc types
- Bunch of getter methods returning strings is unjustified

### Second attempt

```go
type Module interface {
    symbol.Resolver        // Resolve(ctx, name) -> (Symbol, error)
    node.Factory           // Create(ctx, cfg) -> (Node, error)
    HostFuncs() []HostFunc // compiler + runtime iterate
}
```

Problems identified:

- HostFuncs should also be a resolver pattern (compiler does name-based lookup)
- wasm.FunctionType still present - redundant since derivable from Arc types

### Key insight: compiler primitives should have Arc-level symbols

Instead of having two categories (functions with symbols vs. functions without), ALL
host functions should have an Arc-level symbol. Compiler primitives get a symbol with an
`Internal` flag. This means:

- ALL functions go through the same symbol resolution path
- The Arc type on the symbol is the single source of truth for signatures
- WASM types are derived by the compiler via `wasm.ConvertType()`
- wazero infers WASM types from Go function types via reflection
- No `wasm.FunctionType` anywhere in the module definition

### Compiler flow with unified resolution (from Q21, updated Q31)

```
1. ctx.Scope.Resolve("math.sqrt") → Symbol{Type: func(f64)->f64}  // type checking
2. Compile argument expressions
3. ctx.Resolve("math.sqrt", scope.Type) → uint32 handle           // code emission
4. ctx.Writer.WriteCall(handle)
```

Step 3 passes the **concrete function type** (from the scope, after constraint solving).
The resolver uses this to determine polymorphic specialization (see Q31). Phase 1
returns a temporary handle and records usage. Phase 2 partitions into imports vs locals
(see Q34), derives WASM coordinates, registers imports, and patches handles with real
indices.

### Current working direction (needs further refinement)

```go
type Module interface {
    symbol.Resolver  // analyzer + compiler symbol lookup
    node.Factory     // scheduler node creation
    BindTo(ctx context.Context, rt HostRuntime) error  // runtime host func registration
}
```

Where `HostRuntime` abstracts wazero so we don't leak the dependency.

Open questions:

- For compiler primitives: the compiler constructs names like "channel.read_f64" and
  resolves them through the same path as any other function call.
- See Q27 for resolution of flat vs structured keys and `Name()`.
- See Q28 for how the compiler resolver bridges symbols to WASM indices.

---

## Q27: How should module symbol resolution work?

**Question:** Should the math module's MapResolver have `"math.sqrt"` as the key (flat
qualified), or `"sqrt"` as the key with some mechanism to handle the `"math."` prefix?
Does the Module interface need a `Name()` method?

**Initial (wrong) analysis:** I argued flat keys break with nested modules because
`"math.fft.transform"` is ambiguous - could be module `"math"`, member `"fft.transform"`
vs module `"math.fft"`, member `"transform"`. **This was wrong.** Identifiers in Arc
can't contain dots. The dot is a syntactic operator. So `"math.fft.transform"` can only
be three levels of identifier. There is no ambiguity.

**Other initially claimed downsides of flat keys, corrected:**

1. ~~Import aliases break~~ **No.** The analyzer is aware of imports. It translates
   `m.sqrt` → `math.sqrt` before hitting any resolver. Or an `AliasAwareResolver`
   wrapper handles it. The module resolver never sees aliases. Non-problem.

2. ~~Visual editor grouping requires string parsing~~ **No.** Lexical sort on qualified
   names naturally groups `math.pow`, `math.sqrt` adjacently. Free.

3. **Rename fragility** - real but mild. Flat = change N keys, structured = change one
   field.

4. **Silent collisions** - real but mild. Flat = first-match-wins in CompoundResolver,
   structured = can detect duplicate names at registration.

**Proposed solution: `ModuleResolver` as a concrete type, not an interface change.**

A `ModuleResolver` is a struct that satisfies the existing `symbol.Resolver` interface:

```go
type ModuleResolver struct {
    Name    string          // implementation detail, not on the interface
    Members MapResolver     // unqualified keys: "sqrt", "pow"
}

func (m *ModuleResolver) Resolve(ctx context.Context, name string) (Symbol, error) {
    if !strings.HasPrefix(name, m.Name + ".") {
        return Symbol{}, ErrNotFound
    }
    return m.Members.Resolve(ctx, strings.TrimPrefix(name, m.Name + "."))
}
```

Properties:

- The `Resolver` interface is unchanged: `Resolve(ctx, name) -> (Symbol, error)`
- `Name` is a field on the concrete struct, NOT a method on the interface
- The caller doesn't know or care how the resolver works internally
- `CompoundResolver` chains these, first match wins - same pattern as today
- Nested modules: stripped remainder `"fft.transform"` forwards to a child
  `ModuleResolver` with `Name: "fft"`
- Auto-imported/unqualified symbols: use a plain `MapResolver` (no prefix check) in the
  chain - catches bare names like `"set_authority"`

**Impact on Module interface:** `Name()` is NOT needed on the Module interface. The
module satisfies `symbol.Resolver` however it wants internally. If it uses a
`ModuleResolver` with a name field, that's its choice. The interface contract is just
`Resolve`.

**Updated Module interface direction:**

```go
type Module interface {
    symbol.Resolver  // how it resolves is its own business
    node.Factory
    BindTo(ctx context.Context, rt HostRuntime) error
}
```

---

## Q28: How does the compiler bridge from symbol names to WASM function indices?

**Problem:** The current compiler has three separate mechanisms that all produce a
uint32 for `WriteCall`:

1. **`ImportIndex`** - 400-line struct with ~40 typed fields, manually maintained in
   `bindings/imports.go`. All ~390 host functions registered upfront under `"env"`.
   Adding a new host function means touching this struct, the setup function, and the
   generated bindings code.

2. **`FunctionIndices`** - separate `map[string]uint32` for locally-defined functions
   (stage bodies, user-written helpers), populated in `compiler.go` lines 74-76.

3. **`compileBuiltinCall`** - third path that special-cases `len`, `now`, `panic` before
   either of the above is checked.

Three paths for the same thing. The module system eliminates the need for all three.

**Design: two-phase compile-then-link**

WASM function indices are: imports first (0 to N-1), then local functions (N to N+M-1).
If the compiler registers imports lazily during compilation, adding more imports shifts
local function indices - breaking already-emitted `call` instructions. Registering every
STL import upfront avoids this but doesn't generalize to user-defined modules.

Solution: separate compilation from index assignment.

**Phase 1 (compile):** The compiler compiles all code. `ctx.Resolve(name, concreteType)`
returns a temporary handle (not a real index) and records that the function was
referenced. The Writer emits call instructions with placeholder bytes (fixed-width
5-byte padded LEB128). The `concreteType` is the fully-resolved function type from the
scope (after constraint solving) — used for polymorphic specialization (see Q31).

```go
handle := ctx.Resolve("math.sqrt", scope.Type)  // returns temp handle, records usage
ctx.Writer.WriteCall(handle)                      // writes 5-byte placeholder
```

**Phase 2 (link):** Compilation is done. The resolver knows exactly which host functions
were referenced. It registers only those as WASM imports, assigns them indices 0..N-1.
Local functions get indices N..N+M-1. A fixup pass walks the recorded placeholder
locations and patches them with real indices.

```go
resolver.Finalize(wasmModule)       // register used imports, assign all indices
ctx.Writer.PatchCalls(resolver)     // overwrite placeholders with real uint32s
```

**Properties:**

- Only referenced host functions are imported (smallest possible WASM binary)
- Local function indices are stable (assigned after all imports are known)
- User-defined module functions work naturally: compiled as local functions in Phase 1,
  assigned indices in Phase 2
- `ImportIndex` goes away entirely - replaced by the resolver's usage tracking
- `FunctionIndices` map goes away - local functions tracked by the same resolver
- `compileBuiltinCall` goes away - `len`, `now`, `panic` become regular module symbols
  resolved through the same path
- WASM name derivation: split qualified symbol name on last dot. `"math.sqrt"` → import
  module `"math"`, function `"sqrt"`. `"math.fft.transform"` → import module
  `"math.fft"`, function `"transform"`.

**Answer:** Yes, two-phase compile-then-link is the right approach.

**How it connects to modules:** Each module satisfies `symbol.Resolver`. The compiler's
resolver wraps the compound symbol resolver from all modules. During Phase 1, it
resolves symbols to get Arc types (for type derivation) and records usage. During Phase
2, it derives WASM coordinates from the qualified names, converts Arc types to WASM
types via `wasm.ConvertFuncType`, and registers imports. The module's `BindTo` registers
Go implementations under the same (module, function) pairs - both sides derive names
from the same qualified symbol name.

---

## Q29: What does `BindTo` / `HostRuntime` look like?

**Context:** The current binding code is ~3000 lines of generated Go: a `Bindings`
struct with ~100+ function pointer fields, a `Runtime` struct with implementations
closing over `*state.State`, `BindRuntime` connecting them, `Bind` registering
everything under `"env"` with wazero, and 234 type conversion wrapper functions.

**Key constraint discovered:** wazero only accepts four Go types for host function
parameters/returns: `uint32`, `uint64`, `float32`, `float64` (plus `context.Context`).
Sub-32-bit types like `uint8`, `int16` must be wrapped to `uint32`. This is why the
current code has 234 wrapper functions.

**Design:**

`HostRuntime` is a single-method interface:

```go
type HostRuntime interface {
    Export(module, name string, impl any) error
}
```

The wazero implementation of `HostRuntime` groups exports by WASM module name internally
and uses `NewHostModuleBuilder`/`NewFunctionBuilder`/`WithFunc`/`Export`.

Module authors write functions with wazero-compatible types directly:

```go
func (m *MathModule) BindTo(ctx context.Context, rt HostRuntime) error {
    rt.Export("math", "sqrt", func(ctx context.Context, x float64) float64 {
        return gomath.Sqrt(x)
    })
    return nil
}
```

For type-heavy modules (channel, state, series), Go generics can share implementations
within WASM type categories:

```go
func exportReadI32[T ~uint8 | ~uint16 | ~uint32 | ~int8 | ~int16 | ~int32](
    rt HostRuntime, s *ChannelState, name string,
) {
    rt.Export("channel", name, func(ctx context.Context, chID uint32) uint32 {
        series, ok := s.ReadChannelValue(chID)
        if !ok || series.Len() == 0 { return 0 }
        return uint32(telem.ValueAt[T](series, -1))
    })
}
```

Generics reduce implementation duplication (one body for 6 i32-compatible types) but
call sites still enumerate each type. Code generation remains an option for the most
repetitive modules. Translation utilities can be added as needed.

**Answer:** WASM leakage into module implementations is acceptable. Add a translation
layer and/or utilities if the verbosity becomes a problem. The `HostRuntime` interface
is simple: `Export(module, name, impl)`.

---

## Q30: Auto-import / prelude for unqualified names

**Non-question.** How unqualified names like `set_authority` resolve through
module-based resolvers is a localized implementation detail inside a concrete resolver
type. The `symbol.Resolver` interface is just `Resolve(ctx, name)`. Whether a
`ModuleResolver` has an `AutoImport` flag, or there's a separate prelude `MapResolver`,
or some other custom resolver — nothing outside the resolver cares. No cross-package
boundary is affected. Decide during implementation, not architecture.

---

## Q31: How does polymorphism work in the module system?

**Context:** `math.sqrt` needs to work on both `f32` and `f64`. The `^` operator already
dispatches to 10 different host functions based on operand types. The question is how
the module system handles polymorphic functions.

**Research findings — Arc already has the analyzer-side machinery.** Type variables with
constraints (`types.Variable("T", &floatConstraint)`), constraint-based unification in
`analyzer/constraints/unify.go`, and freshening in `types/fresh.go`. By the time the
compiler runs, all type variables are resolved to concrete types on `scope.Type`. The
compiler only ever sees concrete types.

**The compiler resolver interface:**

```go
ctx.Resolve(name string, concreteType types.Type) → (uint32, error)
```

The compiler always has `scope.Type` (fully resolved after constraint solving). It
passes this to the resolver along with the qualified name. For monomorphic functions,
the concrete type is the same every time. For polymorphic functions, it varies per call
site.

**How it works end-to-end (example: `math.sqrt(x)` where x is f64):**

Module author defines:

- Symbol resolver: `sqrt` with type `func(T: float) -> T` (polymorphic)
- BindTo: registers `("math", "sqrt_f32", goImpl)` and `("math", "sqrt_f64", goImpl)`

1. **Analyzer:** resolves `math.sqrt`, constraint system resolves T = f64, scope.Type
   becomes `func(f64) -> f64`
2. **Compiler Phase 1:** calls `ctx.Resolve("math.sqrt", func(f64)->f64)`
3. **Compiler resolver internally:**
   - Splits "math.sqrt" → WASM module "math", base name "sqrt"
   - Looks at concrete type → input is f64 → constructs name "sqrt_f64"
   - Stores pending import: `{module: "math", func: "sqrt_f64"}`
   - Returns temporary handle
4. **Compiler Phase 2 (link):** walks pending imports, registers WASM imports, patches
   handles with real indices
5. **Runtime:** module's `BindTo` registered Go implementations under
   `("math", "sqrt_f64")`. wazero matches WASM imports to Go implementations by (module,
   name).

**The naming convention is the contract.** The module registers `"sqrt_f64"` in
`BindTo`. The compiler resolver constructs `"sqrt_f64"` from the base name + type
suffix. They agree on the convention `{name}_{type_suffix}`. Utilities can enforce this
for module authors.

**Answer:** Accepted. The compiler resolver takes `(name, concreteType)`, derives the
concrete host function name via naming convention, and records a pending import. The
Module interface does not need a `Specialize` method — the resolver handles this
internally. The naming convention is the implicit contract between the resolver and
`BindTo`.

**Resolved follow-up threads:**

- **`len()` return type:** Standardize to `i64` for both series and strings. A length is
  a length. The current `i32` for strings is incidental. With `func(T) -> i64`, there's
  no polymorphism in the return type — fits the model cleanly.

- **Scalar arithmetic operators (`+`, `-`, etc.):** Stay as compiler-internal dispatch.
  Compiler emits native WASM opcodes (`OpF64Add`, etc.) directly. No resolver involved.
  BUT series+series, series+scalar, scalar+series operations DO go through the resolver
  as host function calls (e.g., `"op.series_element_add_f64"`).

- **`channel_read` type:** The compiler knows the channel's declared type from the
  symbol (e.g., `chan<f64>`). It constructs the concrete type on the fly —
  `func(i32) -> f64` — and passes it to `ctx.Resolve("channel.read", func(i32)->f64)`.
  The resolver derives `"read_f64"` from the return type. Straightforward.

---

## Q32: How does the resolver distinguish polymorphic vs monomorphic functions?

**Problem:** When the compiler resolver gets `Resolve("math.sqrt", func(f64)->f64)`, it
needs to append `_f64` to construct the concrete host function name `"sqrt_f64"`. But
when it gets `Resolve("time.now", func()->timestamp)`, it should NOT append `_timestamp`
— the host function is just `"now"`. How does the resolver know which case it's in?

**Solution:** The resolver has access to the compound `symbol.Resolver` (it wraps the
modules). It resolves the **original** polymorphic symbol — not the scope's substituted
version — by calling the compound resolver:

1. `ctx.Resolve("math.sqrt", func(f64)->f64)` called
2. Resolver calls `compoundResolver.Resolve(ctx, "math.sqrt")` → gets original Symbol
   with `func(T: float) -> T` (type variables intact)
3. Checks: does the original symbol's type have type variables? Yes → polymorphic
4. Constructs suffix from concrete type at positions that were type variables → `"_f64"`
5. Splits qualified name on last dot → WASM module `"math"`, function `"sqrt_f64"`
6. Records pending reference

For monomorphic:

1. `ctx.Resolve("time.now", func()->timestamp)` called
2. `compoundResolver.Resolve(ctx, "time.now")` → Symbol with `func() -> timestamp`
3. No type variables → no suffix
4. WASM module `"time"`, function `"now"`

**Key point:** The resolver does NOT need to know which specific module owns the symbol.
It calls the compound resolver (which chains all modules via `CompoundResolver`, first
match wins). The prefix routing is internal to each `ModuleResolver`. The resolver only
sees a Symbol come back.

The suffix is derived from **type variable positions** in the original symbol, not from
all parameter types. So `string_concat(string, string) -> string` (no type variables)
stays `"string_concat"`. `pow(T, T) -> T` where T=f64 becomes `"pow_f64"` — one suffix
because there's one type variable.

**Answer:** Accepted. The extra lookup (compound resolver to get original symbol) is a
map access — cheap. The resolver already wraps the modules.

---

## Q33: What about missing specialization errors?

**Problem:** Module declares `sqrt(T: float) -> T` but `BindTo` only registers
`sqrt_f64`, forgetting `sqrt_f32`. The analyzer accepts `math.sqrt(f32_val)`. The
compiler constructs `"sqrt_f32"` and records it. The error surfaces when?

**Answer:** At WASM instantiation time — when the runtime (wazero or wasmtime) tries to
match imports to host function implementations. This happens **before user code runs**
but after compilation.

The compiler cannot catch this because it's runtime-independent. The WASM binary is the
contract. The compiler doesn't know what `BindTo` will register — and it shouldn't,
because the C++ runtime provides implementations independently via wasmtime's Linker
API. Coupling the compiler to a specific runtime's host function registry would break
the Go/C++ portability.

This is the module author's responsibility. Utilities and codegen for template
specializations (e.g., a helper that generates all numeric type variants) prevent this
bug in practice. The error at instantiation time is clear: "unresolved import:
math.sqrt_f32."

---

## Q34: How does Phase 2 distinguish host functions from local functions?

**Problem:** Phase 1 records every function reference uniformly — just name, concrete
type, and a temporary handle. It doesn't distinguish host functions from local
user-defined functions. How does Phase 2 know which are imports and which are locals?

**Answer:** Phase 1 is "what do we need." Phase 2 is "how do we get it."

After Phase 1, the resolver has:

- **Pending references:** `[{name, concreteType, handle, callSites}]` — every function
  that was called
- **Compiled function bodies:** the local WASM functions that were actually compiled
  (stage bodies, user-defined helpers)

Phase 2 partitions:

- If a pending reference's name matches a compiled function body → **local**. Assign a
  local index (after all imports).
- If no matching body exists → **import**. Derive WASM coordinates, register import.

```
Example:
  Pending refs:
    handle 0: "math.sqrt"     (concreteType: func(f64)->f64)
    handle 1: "myHelper"      (concreteType: func(i32)->i32)
    handle 2: "channel.read"  (concreteType: func(i32)->f64)

  Compiled bodies: ["myHelper"]

  Phase 2 result:
    Imports: math.sqrt_f64 → index 0, channel.read_f64 → index 1
    Locals:  myHelper → index 2
    Patch:   handle 0→0, handle 1→2, handle 2→1
```

This naturally supports future user-defined modules:

- **Inlined modules:** their functions are compiled in Phase 1 alongside the main
  program. Phase 2 sees them as locals. No architectural change needed. The imported
  module's source must be parsed and analyzed before Phase 1 (a "Phase 0" concern —
  dependency resolution).
- **Separately compiled modules:** compiled in an independent Phase 1 + Phase 2,
  producing their own WASM binary. The importing program never compiles them. Phase 2
  treats references as imports. Runtime links the two binaries.

The decision of inline vs separate linking is a future implementation choice that
doesn't affect the resolver architecture.

---

## Q35: Is there a graph node polymorphism gap?

**Context:** The WASM path dispatches polymorphic functions at compile time via naming
conventions (`sqrt_f64`). The graph path dispatches at node creation time via
`cfg.State.Input(0).DataType`. Are these two independent dispatch mechanisms a problem?

**Answer:** No gap. The naming convention (`sqrt_f64`) exists because WASM's import
section requires string names — it's a serialization constraint, not an architectural
choice. The canonical dispatch in both paths is the same: **concrete type →
implementation**. The WASM path encodes that into a string. The graph path does it
in-process with whatever mechanism the module author prefers.

The module system doesn't need shared dispatch infrastructure. The chain-of-command
pattern (`MultiFactory` for node creation, `CompoundResolver` for symbols) handles
cross-module routing. Within a module, `Create` and `BindTo` live in the same package
and naturally share implementations. Keeping them in sync is the module author's
responsibility — same as any other internal code consistency.

The existing operator factory (`runtime/op/op.go`) demonstrates the pattern: it uses
`map[DataType]op.Binary` internally, but that's an implementation detail, not a
system-level concern. Other modules can use switch statements, typed maps, or whatever
fits. The module interface doesn't prescribe internal dispatch.

---

## Q36: WASM binary migration strategy

**Non-issue.** There are no stored compiled WASM binaries. Arc always recompiles from
source (text or graph). The WASM binary is ephemeral. Changing the import namespace from
`"env"` to module-scoped names is just a compiler change with no migration concern.

---

## Q37-Q38: Symbol metadata and type suffix convention

**Out of scope / deferred.** LSP symbol metadata (documentation, module origin, internal
flags) is a consumer-side concern that adapts to whatever the module system looks like.
The type suffix convention (`{name}_{type_suffix}`) is an implementation detail of the
compiler resolver — will be formalized when writing the code.

---

## Q39: Flush cycle orchestration across module-owned state

**Non-issue.** The runtime owns the flush cycle, not modules. Modules expose their state
(Q24/Q25), and the runtime calls flush methods on each module's state in the correct
order (authority before channel writes). The ordering is hardcoded in the runtime — it's
a small, fixed set of steps, not a dynamic plugin system.

---

## Q40: How does the current node key reach module host function implementations?

**Problem:** Stateful variables are keyed per-node — two instances of the same function
in a graph have independent state. Host function closures registered in `BindTo` need to
know which node is currently executing so they can index into the right state slot.

**Answer:** Context-based injection. wazero passes `context.Context` through to every
host function. The runtime attaches the node key to the context before each WASM call:

```
Runtime (before call):
    ctx = context.WithValue(ctx, nodeKeyCtxKey, nodeKey)
    fn.Call(ctx, params...)

Host function (when stateful access needed):
    key := ctx.Value(nodeKeyCtxKey).(string)
    value := moduleState[key][varID]
```

**Why context, not a mutable field:**

- No mutable shared state between runtime and module
- Non-stateful modules (like `math`) ignore it entirely
- The context is already per-call — `nodeImpl.Next(ctx)` passes context to `Call`
- Performance cost (~10-20ns per `Value()` lookup) is accepted; WASM FFI overhead dwarfs
  it

**C++ equivalent:** wasmtime also passes context/caller data to host functions. The same
pattern applies — attach node key to the caller context before each call.

---

## Q41: Gap analysis — architectural completeness review

After deep research across six areas (C++ interop, error handling, WASM memory model,
LSP integration, migration concerns, type system edge cases), the following gaps were
identified and resolved:

**Resolved as non-issues:**

- WASM binary migration (Q36) — no stored binaries, always recompile
- Flush cycle orchestration (Q39) — runtime owns it, not modules
- Error reporting contract — out of scope, panic-and-recover is sufficient

**Resolved with design decisions:**

- Node key injection (Q40) — context-based, via `context.WithValue`

**Deferred to implementation:**

- Type suffix convention formalization (Q38) — internal to compiler resolver
- LSP/Symbol metadata (Q37) — consumer-side concern

**Confirmed compatible (no design change needed):**

- C++ wasmtime Linker API supports name-based binding; current positional binding
  migrates to `Linker::define()` with no architectural impact
- Handle store lifecycle (transient series/string handles, persistent stateful
  variables) is runtime infrastructure per Q8; module decomposition doesn't affect it
- Generated code (~4000 lines) gets replaced by module `BindTo` implementations;
  migration is incremental
