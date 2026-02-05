# 25 - Arc Control Authority

**Feature Name**: Arc Control Authority <br />
**Status**: Draft <br />
**Start Date**: 2025-02-05 <br />
**Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC proposes a design for implementing control authority mechanisms in the Arc
language. Currently, Arc sequences run with hardcoded absolute authority (255) on all
output channels, preventing sophisticated control scenarios like emergency overrides,
cooperative handoffs, operator intervention, and multi-sequence coordination.

The design introduces two complementary mechanisms:

1. **Static authority declarations** via a new `authority()` block at the top of Arc
   files
2. **Dynamic authority changes** via a `set_authority{}` built-in function callable
   within sequences

# 1 - Vocabulary

- **Authority** - An 8-bit unsigned integer (0-255) representing control precedence over
  a channel. Higher values take precedence.
- **Control Subject** - An entity (sequence, operator, task) identified by a unique key
  and name that can hold authority over channels.
- **Control Transfer** - A change in which subject holds authority over a resource.
- **Gate** - A subject's authority claim over a channel within a time region.
- **Silent Skip** - When a write is ignored due to insufficient authority, without
  raising an error.

# 2 - Motivation

## 2.0 - Current Limitations

Arc sequences currently acquire writers at startup with hardcoded `AuthorityAbsolute`
(255) for all output channels:

```go
// core/pkg/service/arc/runtime/task.go:188
ControlSubject: control.Subject{Name: t.p.Name, Key: t.p.Key.String()},
// Authority defaults to AuthorityAbsolute (255) in writer/service.go:134
```

This prevents several critical control scenarios:

1. **Emergency Override**: An abort sequence cannot forcibly take control from a nominal
   sequence
2. **Cooperative Handoff**: Sequences cannot voluntarily transfer control to each other
3. **Operator Intervention**: Human operators cannot take control from running Arc
   sequences via the Console
4. **Multi-Sequence Coordination**: Multiple Arc sequences cannot share control of
   overlapping channels with different priorities

## 2.1 - Existing Python Patterns

The Python client already supports these scenarios:

```python
# Static authority at acquire time
with client.control.acquire(
    write=[valve, vent],
    write_authorities=[200, 100],  # per-channel
) as ctrl:
    # Dynamic authority change
    ctrl.set_authority(254)  # escalate all channels
    ctrl.set_authority({valve: 255, vent: 50})  # per-channel
```

Arc needs equivalent capabilities to achieve feature parity and enable the same control
patterns in compiled sequences.

## 2.2 - Use Cases

The four primary use cases driving this feature:

### Emergency Override (Abort Pattern)

```
Nominal sequence (authority 200) controls pressure valve
  ↓
Abort condition detected (overpressure)
  ↓
Abort sequence escalates to authority 254
  ↓
Abort sequence takes control, closes valve
  ↓
Nominal sequence's writes silently fail
```

### Cooperative Handoff

```
Sequence A (authority 200) completes pressurization
  ↓
Sequence A lowers authority to 50
  ↓
Sequence B (authority 100) automatically gains control
  ↓
Sequence B proceeds with ignition
```

### Operator Intervention

```
Arc sequence (authority 200) running nominal control
  ↓
Operator sees anomaly in Console schematic
  ↓
Operator acquires control with authority 250
  ↓
Arc sequence's writes silently fail
  ↓
Operator manually controls actuator
  ↓
Operator releases control
  ↓
Arc sequence automatically resumes writing
```

### Multi-Sequence Coordination

```
Safety monitor (authority 255) always watching
Nominal sequence (authority 200) running
Diagnostic sequence (authority 50) logging
  ↓
All three can coexist, higher authority wins on conflicts
```

# 3 - Design Philosophy

## 3.0 - Code-First Configuration

Authority declarations should live in the Arc source code itself, not in external
configuration files. This ensures:

- Version control captures the complete program behavior
- Code review includes authority settings
- No hidden configuration affecting safety-critical behavior

## 3.1 - Explicit Over Implicit

While defaults exist for convenience, authors should be encouraged to explicitly declare
their authority requirements. Authority is a safety-critical concept that benefits from
visibility.

## 3.2 - Consistency with Existing Patterns

The design follows Arc's established patterns:

- Top-level declaration blocks (like the planned `import()`)
- Built-in functions using `func{}` instantiation syntax
- Channel references by global identifier

## 3.3 - Extensibility Without Breaking Changes

The syntax is designed to accommodate future extensions:

- Pattern matching on channel names (e.g., `*_valve`)
- Channel groups
- Named authority levels (e.g., `OPERATOR`, `SYSTEM`)
- Authority query functions

# 4 - Detailed Design

## 4.0 - Static Authority Declarations

### 4.0.0 - The `authority()` Block

A new top-level `authority()` block declares the initial authorities for channels when
the Arc program's writer is acquired at startup:

```arc
authority (
    * 200
    valve 100
    vent 150
)

func controller{...} (...) { ... }

sequence main {
    stage running { ... }
}
```

### 4.0.1 - Syntax

```
AuthorityBlock ::= 'authority' '(' AuthorityEntry* ')'

AuthorityEntry ::= '*' NumericLiteral
                 | Identifier NumericLiteral
```

- `*` specifies the default authority for all channels not explicitly listed
- `Identifier` specifies a channel name with its authority value
- `NumericLiteral` must be an integer in the range 0-255

### 4.0.2 - Placement Rules

The `authority()` block must appear **before** any:

- Function declarations (`func`)
- Flow statements (`->`, `=>`)
- Sequence declarations (`sequence`)

This ensures authority is established before any program logic is defined.

### 4.0.3 - Default Behavior

- If no `authority()` block is present, all channels use a system default (currently
  `AuthorityAbsolute`, 255)
- If `authority()` is present but no `*` entry exists, unlisted channels use
  `AuthorityAbsolute` (255)
- The `authority()` block is optional for backwards compatibility

### 4.0.4 - The `*` Wildcard

The `*` symbol is reserved for "all channels" semantics:

```arc
authority (
    * 200        // default for all channels
    valve 100    // override for specific channel
)
```

Currently, `*` only supports the exact "match all" meaning. Future versions may extend
this to pattern matching (e.g., `*_valve`, `press_*`), but v1 only supports:

- `*` for default authority
- Exact channel names for overrides

### 4.0.5 - Examples

**Simple default for all channels:**

```arc
authority (
    * 200
)
```

**Default with per-channel overrides:**

```arc
authority (
    * 200
    safety_valve 255
    diagnostic_output 50
)
```

**No default (uses implicit 255):**

```arc
authority (
    valve 100
    vent 150
)
// All other channels get authority 255
```

## 4.1 - Dynamic Authority Changes

### 4.1.0 - The `set_authority{}` Built-in

Within sequences, authority can be changed at runtime using the `set_authority{}`
built-in function:

```arc
sequence abort {
    stage escalate {
        set_authority{channel=valve, authority=254}
        set_authority{channel=vent, authority=254}
        valve = 0
        vent = 1
    }
}
```

### 4.1.1 - Syntax

```
SetAuthority ::= 'set_authority' '{' SetAuthorityParams '}'

SetAuthorityParams ::= 'channel' '=' ChannelRef ',' 'authority' '=' NumericLiteral

ChannelRef ::= Identifier    // specific channel
             | '*'           // all channels
```

### 4.1.2 - Function Signature

Following Arc's node instantiation pattern (like `wait{}`, `interval{}`):

```arc
set_authority{channel=<channel_or_*>, authority=<0-255>}
```

Parameters:

- `channel` - Either a channel identifier or `*` for all write channels
- `authority` - Integer literal in range 0-255

### 4.1.3 - Usage Patterns

**Single channel:**

```arc
set_authority{channel=valve, authority=254}
```

**All channels:**

```arc
set_authority{channel=*, authority=254}
```

**Multiple channels (requires multiple calls):**

```arc
set_authority{channel=valve, authority=254}
set_authority{channel=vent, authority=254}
set_authority{channel=pressure, authority=254}
```

### 4.1.4 - Return Value

The `set_authority{}` function returns a boolean (`u8`):

- `1` (true) if the authority change was acknowledged by the server
- `0` (false) if the change failed

The exact semantics of the return value and how it integrates with Arc's reactive model
are to be determined during implementation.

### 4.1.5 - Example: Abort Pattern

```arc
authority (
    * 100        // start with low authority
)

sequence main {
    stage nominal {
        sensor -> controller{output=valve}
        abort_condition => abort
    }
}

sequence abort {
    stage escalate {
        // Escalate to take control from nominal
        set_authority{channel=*, authority=254}
    }

    stage safed {
        valve = 0
        vent = 1
    }
}
```

## 4.2 - Behavior on Authority Loss

### 4.2.0 - Silent Skip (Default)

When an Arc sequence attempts to write to a channel but lacks sufficient authority, the
write is **silently skipped**:

- No error is raised
- The sequence continues executing
- The write simply has no effect

This matches the Python client's default behavior (`err_on_unauthorized=False`).

### 4.2.1 - Automatic Resume

When a higher-authority entity releases control, the Arc sequence automatically resumes
writing:

1. Arc sequence has authority 200, is writing to `valve`
2. Operator acquires `valve` with authority 250
3. Arc sequence's writes to `valve` silently fail
4. Operator releases control
5. Arc sequence's writes to `valve` automatically succeed again

The Arc sequence never "lost" its authority declaration - it was simply outranked
temporarily.

### 4.2.2 - No Detection Mechanisms (v1)

v1 does not include mechanisms for detecting authority loss:

- No `has_authority(channel)` query function
- No `lost_authority` / `gained_authority` reactive triggers
- No stage transitions on authority changes

These may be added in future versions.

## 4.3 - Compile-Time Validation

The Arc analyzer performs the following validations on authority declarations:

### 4.3.0 - Channel Existence

All channel identifiers in `authority()` and `set_authority{}` must exist in the bound
channel set:

```arc
authority (
    * 200
    nonexistent_channel 100  // ERROR: channel not found
)
```

### 4.3.1 - Authority Range

Authority values must be in the range 0-255:

```arc
authority (
    * 300  // ERROR: authority must be 0-255
)
```

### 4.3.2 - Duplicate Detection

The same channel cannot be declared twice in the `authority()` block:

```arc
authority (
    * 200
    valve 100
    valve 150  // ERROR: duplicate authority for 'valve'
)
```

## 4.4 - Interaction with Other Systems

### 4.4.0 - Lua Sequences

Arc sequences and Lua sequences are completely independent. They compete for control via
the normal Synnax authority system:

- Both are writers with authorities
- Cesium's control system resolves conflicts
- No special coordination needed

### 4.4.1 - Console Operators

Operators can take control via Console schematics with their own authority level.
The same rules apply:

- Higher authority wins
- Arc sequence's writes silently fail when outranked
- Arc sequence resumes when operator releases

### 4.4.2 - Multiple Arc Sequences

Multiple Arc sequences can run concurrently on overlapping channels:

- Each has its own authority settings
- Higher authority wins on conflicts
- Lower authority sequences' writes silently fail

# 5 - Implementation

## 5.0 - Parser Changes

Add the `authority` keyword and parsing rules for the `authority()` block:

```antlr
authorityBlock
    : 'authority' '(' authorityEntry* ')'
    ;

authorityEntry
    : ('*' | IDENTIFIER) INTEGER_LITERAL
    ;
```

Add `set_authority` as a recognized built-in function name.

## 5.1 - Analyzer Changes

### 5.1.0 - New Analyzer Pass

Add validation for authority declarations:

- Verify channel existence
- Verify authority range (0-255)
- Detect duplicate channel entries
- Ensure `authority()` block precedes other declarations

### 5.1.1 - Symbol Table Extension

Extend the symbol table to track:

- Default authority value
- Per-channel authority overrides
- Usage of `set_authority{}` in stages

## 5.2 - IR Extension

Extend the IR to include authority metadata:

```go
type Package struct {
    // ... existing fields
    Authority AuthorityConfig `json:"authority,omitempty"`
}

type AuthorityConfig struct {
    Default   *uint8            `json:"default,omitempty"`
    Channels  map[string]uint8  `json:"channels,omitempty"`
}
```

## 5.3 - Runtime Changes

### 5.3.0 - Writer Configuration

Modify the Arc runtime to use authority config when opening writers:

```go
// Before (hardcoded)
Authorities: []control.Authority{control.AuthorityAbsolute}

// After (from IR)
Authorities: buildAuthorities(program.Authority, channels)
```

### 5.3.1 - WASM Host Bindings

Add new host function for `set_authority`:

```go
// bindings.go
func (r *Runtime) SetAuthority(channelID int32, authority uint8) uint8 {
    err := r.writer.SetAuthority(control.Config{
        Keys:        []channel.Key{channel.Key(channelID)},
        Authorities: []control.Authority{control.Authority(authority)},
    })
    if err != nil {
        return 0
    }
    return 1
}
```

### 5.3.2 - "All Channels" Support

Implement `*` handling in `set_authority{}`:

```go
func (r *Runtime) SetAuthorityAll(authority uint8) uint8 {
    err := r.writer.SetAuthority(control.Config{
        Keys:        r.allWriteChannels,
        Authorities: []control.Authority{control.Authority(authority)},
    })
    // ...
}
```

## 5.4 - Compiler Changes

Generate WASM code that calls the `set_authority` host function when
`set_authority{}` nodes are executed.

# 6 - Future Extensions

The following are explicitly out of scope for v1 but the design accommodates them:

## 6.0 - Pattern Matching

```arc
authority (
    * 200
    *_valve 255        // all channels ending in _valve
    press_* 150        // all channels starting with press_
)
```

## 6.1 - Channel Groups

```arc
authority (
    * 200
    group safety_valves 255
    group telemetry 50
)
```

## 6.2 - Named Authority Levels

```arc
authority (
    * OPERATOR         // 100
    safety SYSTEM      // 255
)
```

## 6.3 - Authority Detection

```arc
sequence main {
    stage running {
        lost_authority(valve) => handle_loss
        gained_authority(valve) => resume_control
    }
}
```

## 6.4 - Authority Query

```arc
if has_authority(valve) {
    valve = 100.0
} else {
    set_status{message="Waiting for control"}
}
```

# 7 - Testing Strategy

## 7.0 - Unit Tests

- Parser tests for `authority()` block syntax
- Analyzer tests for validation rules
- Compiler tests for IR generation
- Runtime tests for authority configuration

## 7.1 - Integration Tests

- Arc sequence with static authority
- Arc sequence with dynamic `set_authority{}`
- Arc sequence vs operator control conflict
- Multiple Arc sequences on overlapping channels
- Abort pattern (authority escalation)
- Cooperative handoff (authority de-escalation)

## 7.2 - Edge Cases

- Empty `authority()` block
- `authority()` with only `*`
- `set_authority{}` with invalid channel
- `set_authority{}` with out-of-range authority
- Rapid authority changes

# 8 - Migration

Existing Arc programs continue to work unchanged:

- No `authority()` block = all channels at default authority (255)
- Behavior is backwards compatible
- Authors can incrementally add authority declarations

# 9 - Summary

This RFC introduces control authority mechanisms to Arc through:

1. **`authority()` block** - Top-level static declaration of initial authorities
2. **`set_authority{}`** - Runtime authority changes within sequences
3. **`*` wildcard** - Default authority for all channels
4. **Silent skip** - Graceful handling of authority conflicts

The design prioritizes simplicity for v1 while leaving room for future extensions like
pattern matching, groups, and authority detection.
