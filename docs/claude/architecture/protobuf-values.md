# Protobuf Value Serialization Standard

This document establishes the long-term standard for handling different types of data in
protobuf messages across the Synnax codebase.

## The Three Protobuf Types

### 1. `google.protobuf.Struct` - Loosely-Typed Configuration

**Use for:** User-provided, UI-driven configuration with unknown or flexible schema.

**Go representation:** `map[string]any`

**Examples:**

- `task.Task.config` - Hardware task configuration
- `graph.Node.config` - Graph editor node configuration
- `workspace.Workspace.layout` - UI layout configuration
- `device.Device.properties` - Device properties

**Characteristics:**

- Schema is not known at compile time
- Values are JSON-like primitives (string, number, bool, nested objects)
- No type safety beyond JSON types
- User-editable in UI contexts
- Oracle automatically generates `structpb.NewStruct()` for `map[string]any`

**Example proto:**

```protobuf
message Task {
  string name = 1;
  google.protobuf.Struct config = 3;  // Flexible, user-provided config
}
```

**Example usage:**

```go
// Go type
type Task struct {
    Name   string         `json:"name"`
    Config map[string]any `json:"config"`
}

// Values are JSON-like
config := map[string]any{
    "host": "localhost",
    "port": 9090,
    "timeout": 30.0,
    "enabled": true,
}
```

### 2. `google.protobuf.Any` - Strongly-Typed Protobuf Messages

**Use for:** Type-safe wrapping of concrete protobuf messages where the exact type may
vary.

**Go representation:** Specific protobuf message types

**Examples:**

- `status.Status.details` - Error/status details (various types)
- `control.State.resource` - Control resource (channel, device, etc.)

**Characteristics:**

- Strong type safety
- Contains type URL + serialized protobuf bytes
- Receiver can deserialize to concrete type
- Type information preserved across serialization
- Used for polymorphism with protobuf messages

**Example proto:**

```protobuf
message Status {
  string message = 1;
  google.protobuf.Any details = 2;  // Could be various error detail types
}
```

**Example usage:**

```go
// Pack a specific protobuf message
details, err := anypb.New(&ChannelNotFoundError{Key: 123})

// Unpack to specific type
var errDetails ChannelNotFoundError
if err := details.UnmarshalTo(&errDetails); err != nil {
    // handle error
}
```

### 3. Custom Type Systems - Type Metadata + Serialized Bytes

**Use for:** Values with custom type systems that include metadata beyond protobuf's
capabilities.

**Go representation:** Struct with separate Type and Value fields

**Examples:**

- `arc.types.Param` - Arc type system with units, dimensions, constraints
- Any domain-specific type system with rich metadata

**Characteristics:**

- Type information stored separately from value
- Values may be custom Go types (type aliases, structs)
- Type metadata includes domain-specific information (units, dimensions, etc.)
- Cannot use `google.protobuf.Value` because it only handles JSON primitives
- Cannot use `google.protobuf.Any` because values aren't protobuf messages

**Current problematic pattern:**

```protobuf
// ❌ PROBLEMATIC: Using Value for strongly-typed Arc values
message Param {
  string name = 1;
  Type type = 2;
  google.protobuf.Value value = 3;  // Breaks for telem.TimeSpan, etc.
}
```

## The Problem with Arc's Current Approach

Arc's `types.Param` has:

- Rich type metadata (`Type` field with units, dimensions, constraints)
- Go runtime values (`Value any` - could be `telem.TimeSpan`, `telem.TimeStamp`, etc.)

Currently using `google.protobuf.Value` for the value field, which **only accepts:**

- null, bool, number (float64), string
- map[string]any, []any

**It rejects:**

- Custom type aliases like `telem.TimeSpan` (even though it's just
  `type TimeSpan int64`)
- Custom structs
- Any non-JSON type

## Solutions for Custom Type Systems

### Option 1: Bytes with Documented Encoding (Recommended)

Store values as serialized bytes using msgpack or JSON.

**Advantages:**

- Handles any serializable Go type
- No conversion logic to maintain
- Already have msgpack tags on `types.Param`
- Type interpretation done using the Type field
- Future-proof for new custom types

**Disadvantages:**

- Not human-readable in protobuf dumps
- Slightly larger wire format than native protobuf

**Implementation:**

```protobuf
message Param {
  string name = 1;
  Type type = 2;
  bytes value = 3;  // msgpack-encoded value
}
```

```go
// Serialization
func ParamToPB(ctx context.Context, r types.Param) (*Param, error) {
    valueBytes, err := msgpack.Marshal(r.Value)
    if err != nil {
        return nil, err
    }
    pb := &Param{
        Name:  r.Name,
        Type:  typeVal,
        Value: valueBytes,
    }
    return pb, nil
}

// Deserialization
func ParamFromPB(ctx context.Context, pb *Param) (types.Param, error) {
    var value any
    if err := msgpack.Unmarshal(pb.Value, &value); err != nil {
        return types.Param{}, err
    }
    // Use Type field to interpret the value correctly
    return types.Param{
        Name:  pb.Name,
        Type:  typeFromPB,
        Value: value,
    }, nil
}
```

### Option 2: Type-Aware Conversion Layer

Add conversion logic that uses the Type field to unwrap custom types to primitives.

**Advantages:**

- Human-readable protobuf
- Uses existing `google.protobuf.Value`

**Disadvantages:**

- Requires maintaining conversion logic for every custom type
- Fragile - breaks when new custom types added
- Tight coupling between type system and serialization

**Implementation:**

```go
func valueToProto(v any, t Type) (*structpb.Value, error) {
    // Unwrap based on type metadata
    if t.Unit != nil && t.Unit.Dimensions.Equal(TimeSpan().Unit.Dimensions) {
        // Time-based types: unwrap to int64
        switch val := v.(type) {
        case telem.TimeSpan:
            return structpb.NewValue(int64(val))
        case telem.TimeStamp:
            return structpb.NewValue(int64(val))
        }
    }

    // Add more custom type conversions here...

    return structpb.NewValue(v)
}
```

### Option 3: Separate Proto Definitions for Each Type

Create protobuf definitions for each value type and use `google.protobuf.Any`.

**Advantages:**

- Type-safe serialization
- Well-supported protobuf pattern

**Disadvantages:**

- Massive overkill for primitive values
- Need proto definitions for every possible value type
- Complex for simple values like numbers with units

## Recommended Standard

### For New Code

| Data Category             | Go Type             | Protobuf Type              | Use Case                                 |
| ------------------------- | ------------------- | -------------------------- | ---------------------------------------- |
| **Flexible config**       | `map[string]any`    | `google.protobuf.Struct`   | User-provided, UI-driven, unknown schema |
| **Protobuf polymorphism** | Protobuf messages   | `google.protobuf.Any`      | Type-safe message wrapping               |
| **Custom type systems**   | Type + Value struct | Type field + `bytes` value | Rich type metadata beyond protobuf       |

### For Arc Specifically

**Change `arc.types.pb.Param` to use bytes:**

```protobuf
message Param {
  string name = 1;
  Type type = 2;
  bytes value = 3;  // msgpack-encoded, interpreted using Type field
}
```

**Rationale:**

- Arc has a custom type system with units, dimensions, constraints
- Values include custom Go types (`telem.TimeSpan`, etc.)
- Type field already contains all metadata needed to interpret the value
- Bytes + Type is semantically correct: "here's a typed value and its serialized form"
- Future-proof for Arc type system evolution

### Migration Path

1. **Change proto definition:** Update `arc/go/types/pb/types.proto`
2. **Update Oracle:** Modify oracle to generate msgpack serialization for `bytes` fields
   with custom type context
3. **Update translators:** Replace `structpb.NewValue()` with `msgpack.Marshal()`
4. **Update tests:** Fix any tests that depend on the proto format
5. **Document:** Add comments in proto explaining the encoding format

## Anti-Patterns to Avoid

❌ **Using `google.protobuf.Value` for custom types**

```protobuf
// Don't do this for custom type systems
message Param {
  Type type = 2;
  google.protobuf.Value value = 3;  // Breaks for non-JSON types
}
```

❌ **Using `map[string]any` for strongly-typed data**

```go
// Don't do this for Arc IR nodes
type Node struct {
    Config map[string]any  // Loses type information
}
```

❌ **Hardcoding type-specific conversions**

```go
// Don't do this - not scalable
if ts, ok := value.(telem.TimeSpan); ok {
    value = int64(ts)
}
if ts, ok := value.(telem.TimeStamp); ok {
    value = int64(ts)
}
// ... more hardcoded types
```

## Patterns to Follow

✅ **Use Struct for flexible, JSON-like config**

```protobuf
message Task {
  google.protobuf.Struct config = 1;  // User-provided, flexible
}
```

✅ **Use Any for protobuf message polymorphism**

```protobuf
message Status {
  google.protobuf.Any details = 1;  // Various protobuf error types
}
```

✅ **Use Type + bytes for custom type systems**

```protobuf
message Param {
  Type type = 1;       // Rich type metadata
  bytes value = 2;     // Serialized value (msgpack)
}
```

## Implementation Tasks

To fix the current Arc issue:

1. [ ] Update `arc/go/types/pb/types.proto` to use `bytes` for Param.value
2. [ ] Modify Oracle plugin to generate msgpack serialization
3. [ ] Update `arc/go/types/pb/translator.gen.go` (or custom translator)
4. [ ] Run tests: `bazel test //client/cpp/arc/...`
5. [ ] Update documentation in type system packages
6. [ ] Document msgpack as the standard encoding for custom type systems

## References

- Protocol Buffers Well-Known Types:
  https://protobuf.dev/reference/protobuf/google.protobuf/
- `google.protobuf.Struct`: For JSON-like structures
- `google.protobuf.Value`: For JSON primitives
- `google.protobuf.Any`: For type-safe message wrapping
