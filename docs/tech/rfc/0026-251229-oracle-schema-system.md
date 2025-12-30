# 0026 - Oracle Schema System

**Feature Name:** Oracle - Schema-First Code Generation for Meta-Data Structures

**Status:** Draft

**Related:** [RFC 0025 - Meta Data Structures](./0025-251214-meta-data.md)

---

# 0 - Summary

Oracle is a schema-first code generation system that addresses the meta-data structure
problems identified in RFC 0025. It provides a single source of truth for resource
definitions, generating type-safe code across Go, TypeScript, and Python, along with
query builders, indexes, validation, and migrations.

```
                              ┌─────────────────────┐
                              │   .oracle schemas   │
                              └──────────┬──────────┘
                                         │
          ┌──────────────────────────────┼──────────────────────────────┐
          │                              │                              │
          ▼                              ▼                              ▼
   ┌─────────────┐                ┌─────────────┐                ┌─────────────┐
   │ Go Plugins  │                │ TS Plugins  │                │ Py Plugins  │
   ├─────────────┤                ├─────────────┤                ├─────────────┤
   │ go/types    │                │ ts/types    │                │ py/types    │
   │ go/query    │                │ zod         │                │ py/pydantic │
   │ go/index    │                │ ts/query    │                │ py/query    │
   │ go/validate │                │             │                │             │
   └─────────────┘                └─────────────┘                └─────────────┘
          │                              │                              │
          ▼                              ▼                              ▼
    types.gen.go                  types.gen.ts                  types.gen.py
    query.gen.go                  schema.gen.ts                 query.gen.py
    index.gen.go                  query.gen.ts                  pydantic...
```

---

# 1 - Motivation

RFC 0025 identifies several problems with the current meta-data toolchain:

1. **Boilerplate** - Similar code replicated across Go services, TypeScript client,
   Python client (~4,500-7,500 lines of repeated patterns)

2. **Inconsistent Query Capabilities** - Each service implements its own query patterns
   with no standardization for filtering, sorting, or pagination

3. **Client-Side Migrations** - Visualizations stored on server but migrated in Console,
   causing SDK incompatibility

4. **Multiple Sources of Truth** - Types defined separately in Go, TypeScript, Python,
   leading to drift and bugs

5. **Inefficient Queries** - Gorp uses O(n) full scans for any non-key lookup; no
   secondary indexes

Oracle solves these by making the schema the single source of truth from which all
code is derived.

---

# 2 - Design Principles

## 2.0 - Schema is Source of Truth

All type definitions, query capabilities, indexes, and constraints are declared in
`.oracle` schema files. Generated code is derived, never hand-edited.

## 2.1 - Semantic APIs, Not Generic Query Language

Unlike GraphQL or a generic query DSL, Oracle generates **semantic, type-safe APIs**
specific to each resource:

```go
// Generated - semantic and type-safe
ranger.NewRetrieve().
    WhereLabels(labelA, labelB).HasAny().
    OrderByCreatedAt(query.Desc).
    Page("", 25).
    Exec(ctx)

// NOT this - generic and stringly-typed
query.New("range").
    Where("labels", "hasAny", []string{...}).
    OrderBy("created_at", "desc").
    Exec(ctx)
```

## 2.2 - Plugin Architecture

Core parses and validates schemas. Plugins handle language-specific code generation.
This separation allows:

- Adding new target languages without changing core
- Independent plugin versioning
- Custom internal plugins for specific needs

## 2.3 - Domain-Oriented Design

The language separates concerns into **domains**. Each field can have multiple domain
blocks that define different aspects of behavior (validation, querying, indexing, etc.).
Plugins are associated with specific domains - they scan for structs/fields that have
their domain and generate code accordingly.

This enables:

- Clean separation of structure from behavior
- Plugin-specific concerns isolated in their own blocks
- Extensibility without language changes

## 2.4 - No Intermediate Representation

The ANTLR parse tree serves as the representation. Plugins receive:

- Parse tree (walk it directly)
- Resolution table (maps names to definitions)

No separate IR types that duplicate the AST structure.

---

# 3 - Schema Language

## 3.0 - File Extension

`.oracle`

## 3.1 - Core Syntax

The language has minimal reserved keywords. Domain names (like `validate`, `query`,
`index`) are not reserved - they are identifiers that plugins look for.

```oracle
// schema/ranger.oracle

import "schema/label"

struct Range {
    field key uuid {
        domain id
    }

    field name string {
        domain validate {
            required
            max_length 255
        }
        domain query {
            eq
            neq
            contains
            starts_with
        }
        domain index {
            lookup
        }
    }

    field labels uuid[] {
        domain relation {
            target label.Label
            cardinality many_to_many
        }
        domain query {
            has_any
            has_all
            has_none
        }
        domain index {
            lookup
        }
    }

    field time_range time_range {
        domain validate {
            required
        }
        domain query {
            overlaps
            contains_time
        }
        domain index {
            range
        }
    }

    field created_at timestamp {
        domain validate {
            default now
            immutable
        }
        domain query {
            eq
            gt
            gte
            lt
            lte
            between
        }
        domain index {
            sorted
        }
        domain sort
    }

    field parent uuid? {
        domain relation {
            target Range
            self
        }
    }

    // Output configuration - where to generate code for each target
    domain go {
        output "core/ranger"
    }

    domain ts {
        output "console/src/ranger"
    }

    domain python {
        output "client/py/synnax/ranger"
    }
}
```

## 3.2 - Reserved Keywords

Only these are reserved by the language:

| Keyword | Purpose |
|---------|---------|
| `struct` | Define a data structure |
| `field` | Define a field within a struct |
| `domain` | Define a domain block with rules/expressions |
| `enum` | Define an enumeration type |
| `import` | Import other schema files |

Everything else (like `validate`, `query`, `index`, `relation`) are just identifiers
that plugins look for - they are not reserved by the language.

## 3.3 - Primitive Types

All primitives are lowercase:

| Type | Go | TypeScript | Python |
|------|-----|------------|--------|
| `uuid` | `uuid.UUID` | `string` | `UUID` |
| `string` | `string` | `string` | `str` |
| `bool` | `bool` | `boolean` | `bool` |
| `int8` | `int8` | `number` | `int` |
| `int16` | `int16` | `number` | `int` |
| `int32` | `int32` | `number` | `int` |
| `int64` | `int64` | `number` | `int` |
| `uint8` | `uint8` | `number` | `int` |
| `uint16` | `uint16` | `number` | `int` |
| `uint32` | `uint32` | `number` | `int` |
| `uint64` | `uint64` | `number` | `int` |
| `float32` | `float32` | `number` | `float` |
| `float64` | `float64` | `number` | `float` |
| `timestamp` | `telem.TimeStamp` | `number` | `int` |
| `timespan` | `telem.TimeSpan` | `number` | `int` |
| `time_range` | `telem.TimeRange` | `TimeRange` | `TimeRange` |
| `json` | `map[string]any` | `Record<string, unknown>` | `dict[str, Any]` |
| `bytes` | `[]byte` | `Uint8Array` | `bytes` |

## 3.4 - Type Modifiers

| Syntax | Meaning |
|--------|---------|
| `type` | Required (default) |
| `type?` | Optional (nullable) |
| `type[]` | List/array |
| `type[]?` | Optional list |

## 3.5 - Field Syntax

Fields are defined with the `field` keyword, followed by name, type, and an optional
block containing domain definitions:

```oracle
// Full form with domains
field name string {
    domain validate {
        required
        max_length 255
    }
    domain query {
        eq
        contains
    }
}

// Minimal form (no domains)
field description string?

// With type modifiers
field labels uuid[]
field metadata json?
```

## 3.6 - Domain Syntax

Domains define behavior for a field. Each domain has a name and contains expressions:

```oracle
field name string {
    domain validate {
        required
        max_length 255
        min_length 1
    }
    domain query {
        eq
        neq
        contains
    }
    domain index {
        lookup
    }
}
```

Domains can be empty (presence is all that matters):

```oracle
field created_at timestamp {
    domain sort
}
```

## 3.7 - Domain Expression Syntax

Expressions within domains are newline-separated. The expression grammar is flexible:

```
expression = identifier
           | identifier value
           | identifier value value ...

value = identifier | string | number | bool
```

Examples:

```oracle
domain validate {
    required                    // flag
    max_length 255              // identifier + number
    default "untitled"          // identifier + string
    default now                 // identifier + identifier
    pattern "[a-z]+"            // identifier + string
}

domain relation {
    target Label                // identifier + identifier
    cardinality many_to_many    // identifier + identifier
}

domain query {
    eq                          // flag
    neq                         // flag
    between                     // flag
}
```

## 3.8 - Enums

Enums require explicit values (integers or strings):

```oracle
// Integer values
enum TaskState {
    pending = 0
    running = 1
    completed = 2
    failed = 3
}

// String values
enum DataType {
    float32 = "float32"
    float64 = "float64"
    int32 = "int32"
    timestamp = "timestamp"
}
```

Usage in fields:

```oracle
struct Task {
    field state TaskState {
        domain validate {
            default pending
        }
        domain query {
            eq
            in
        }
    }
}
```

## 3.9 - Imports

Imports use paths relative to the schema directory. The file name (without extension)
determines the namespace:

```oracle
import "schema/label"
import "schema/channel"

struct Range {
    field labels uuid[] {
        domain relation {
            target label.Label
            cardinality many_to_many
        }
    }

    field channel uuid {
        domain relation {
            target channel.Channel
        }
    }
}
```

File at `schema/label.oracle` → namespace `label`
File at `schema/schematic.oracle` → namespace `schematic`

## 3.10 - Embedded Structs

Structs without an `id` domain are treated as embedded value objects:

```oracle
struct Position {
    field x float64 {
        domain validate {
            required
        }
    }
    field y float64 {
        domain validate {
            required
        }
    }
}

struct Viewport {
    field position Position {
        domain validate {
            required
        }
    }
    field zoom float64 {
        domain validate {
            default 1.0
        }
    }
}

struct Schematic {
    field key uuid {
        domain id
    }

    field name string {
        domain validate {
            required
            max_length 255
        }
    }

    field viewport Viewport {
        domain validate {
            required
        }
    }

    field nodes Node[]
}
```

## 3.11 - Cross-Cutting Concerns

For concerns that span multiple fields (composite indexes, multi-field validation),
use struct-level domain blocks:

```oracle
struct Range {
    field key uuid {
        domain id
    }

    field name string {
        domain validate {
            required
        }
    }

    field workspace uuid {
        domain relation {
            target Workspace
        }
    }

    field created_at timestamp {
        domain validate {
            default now
        }
    }

    // Struct-level domains for cross-field concerns
    domain index {
        composite name created_at sorted
        unique name workspace
    }

    domain validate {
        time_range.end > time_range.start
    }

    domain on_delete {
        labels detach
        children cascade
    }

    // Output configuration
    domain go {
        output "core/ranger"
    }

    domain ts {
        output "console/src/ranger"
    }
}
```

---

# 4 - Plugin System

## 4.0 - Overview

Plugins are associated with specific domains. When processing a schema, each plugin
scans for structs and fields that have domains it cares about. Plugins can declare
dependencies on other plugins via `Requires()`, enabling composable code generation
where specialized plugins build on each other.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              ORACLE CORE                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────┐    ┌──────────┐    ┌─────────────────────────────────┐   │
│  │  Lexer   │───▶│  Parser  │───▶│  ANTLR Parse Tree               │   │
│  │ (ANTLR)  │    │ (ANTLR)  │    │  (ISchemaContext)               │   │
│  └──────────┘    └──────────┘    └────────────────┬────────────────┘   │
│                                                   │                     │
│                                  ┌────────────────▼────────────────┐   │
│                                  │  Analyzer                       │   │
│                                  │  - Resolve imports              │   │
│                                  │  - Resolve references           │   │
│                                  │  - Build resolution table       │   │
│                                  └────────────────┬────────────────┘   │
│                                                   │                     │
│                                  ┌────────────────▼────────────────┐   │
│                                  │  Plugin Orchestrator            │   │
│                                  │  - Topological sort by deps     │   │
│                                  │  - Invoke plugins in order      │   │
│                                  │  - Collect output files         │   │
│                                  └────────────────┬────────────────┘   │
│                                                   │                     │
└───────────────────────────────────────────────────┼─────────────────────┘
                                                    │
     ┌──────────────────────────────────────────────┼──────────────────────────────┐
     │                              │               │               │              │
     ▼                              ▼               ▼               ▼              ▼
┌──────────┐                 ┌──────────┐    ┌──────────┐    ┌──────────┐   ┌──────────┐
│go/types  │◄────────────────│go/query  │    │go/index  │    │go/validate   │ zod      │
│          │   requires      │          │    │          │    │          │   │          │
│Domains:  │◄────────────────│Domains:  │    │Domains:  │    │Domains:  │   │Domains:  │
│• go      │   requires      │• query   │    │• index   │    │• validate│   │• ts      │
│• id      │◄────────────────│          │    │          │    │          │   │• validate│
└──────────┘   requires      └──────────┘    └──────────┘    └──────────┘   └──────────┘
     │                              │               │               │              │
     ▼                              ▼               ▼               ▼              ▼
types.gen.go                 query.gen.go    index.gen.go    (in types)    schema.gen.ts
```

## 4.1 - Plugin ↔ Domain Mapping

Each plugin handles specific domains and can depend on other plugins. The orchestrator
runs plugins in topological order based on their dependencies.

### Go Plugins

| Plugin | Domains | Requires | Output |
|--------|---------|----------|--------|
| `go/types` | `go`, `id` | - | `types.gen.go` |
| `go/query` | `query` | `go/types` | `query.gen.go` |
| `go/index` | `index` | `go/types` | `index.gen.go` |
| `go/validate` | `validate` | `go/types` | validation in `types.gen.go` |

### TypeScript Plugins

| Plugin | Domains | Requires | Output |
|--------|---------|----------|--------|
| `ts/types` | `ts` | - | `types.gen.ts` |
| `zod` | `ts`, `validate` | - | `schema.gen.ts` |
| `ts/query` | `query` | `ts/types` | `query.gen.ts` |

### Python Plugins

| Plugin | Domains | Requires | Output |
|--------|---------|----------|--------|
| `py/types` | `python` | - | `types.gen.py` |
| `py/pydantic` | `python`, `validate` | - | Pydantic models |
| `py/query` | `query` | `py/types` | `query.gen.py` |

### Output Configuration

The `output` expression within each language domain specifies where generated files go:

```oracle
domain go {
    output "core/ranger"    // generates to core/ranger/*.gen.go
}

domain ts {
    output "console/src/ranger"    // generates to console/src/ranger/*.gen.ts
}
```

### Dependency Resolution

When plugins declare dependencies via `Requires()`, the orchestrator:

1. Builds a dependency graph of all registered plugins
2. Topologically sorts plugins to determine execution order
3. Runs plugins in order, ensuring dependencies complete first
4. Fails fast if circular dependencies are detected

```go
// Example: go/query depends on go/types
type GoQueryPlugin struct{}

func (p *GoQueryPlugin) Name() string      { return "go/query" }
func (p *GoQueryPlugin) Domains() []string { return []string{"query"} }
func (p *GoQueryPlugin) Requires() []string { return []string{"go/types"} }
```

## 4.2 - Core Responsibilities

| Responsibility | Description |
|----------------|-------------|
| Parsing | Lex and parse `.oracle` files using ANTLR |
| Import Resolution | Load and merge imported schema files |
| Reference Resolution | Resolve `label.Label` to actual definitions |
| Plugin Invocation | Pass parse tree + resolutions to plugins |

## 4.3 - Plugin Responsibilities

| Responsibility | Description |
|----------------|-------------|
| Domain Handling | Define which domains the plugin processes |
| Type Mapping | `uuid` → `uuid.UUID` (Go), `string` (TS), `UUID` (Py) |
| Naming Conventions | `created_at` → `CreatedAt` (Go), `createdAt` (TS) |
| Code Generation | Structs, interfaces, classes, validators |
| File Structure | One file per struct, directory layout |
| Language Idioms | Struct tags, Zod schemas, Pydantic models |

## 4.4 - Plugin Interface

```go
package plugin

import "github.com/synnaxlabs/synnax/oracle/parser"

type Plugin interface {
    // Domains returns the domain names this plugin handles
    Domains() []string

    // Generate produces output files from the parsed schemas
    Generate(req *Request) (*Response, error)
}

type Request struct {
    Schemas     []parser.ISchemaContext  // Parse trees
    Resolutions *Resolutions             // Resolved references
    Options     map[string]any           // Plugin options from config
}

type Response struct {
    Files []File
}

type File struct {
    Path    string   // Relative to output directory
    Content []byte
}

type Resolutions struct {
    Structs map[string]parser.IStructDefContext
    Enums   map[string]parser.IEnumDefContext
}
```

---

# 5 - Generated Code

## 5.0 - Go

### Types

```go
// core/pkg/service/ranger/types.gen.go

// Code generated by oracle. DO NOT EDIT.

package ranger

import (
    "github.com/google/uuid"
    "github.com/synnaxlabs/x/telem"
)

type Range struct {
    Key       uuid.UUID       `json:"key" msgpack:"key"`
    Name      string          `json:"name" msgpack:"name"`
    Labels    []uuid.UUID     `json:"labels" msgpack:"labels"`
    TimeRange telem.TimeRange `json:"time_range" msgpack:"time_range"`
    CreatedAt telem.TimeStamp `json:"created_at" msgpack:"created_at"`
    Parent    *uuid.UUID      `json:"parent,omitempty" msgpack:"parent,omitempty"`
}

func (r Range) GorpKey() uuid.UUID { return r.Key }
func (r Range) SetOptions() []any  { return nil }

func (r Range) Validate() error {
    if r.Name == "" {
        return errors.New("name is required")
    }
    if len(r.Name) > 255 {
        return errors.New("name exceeds max length 255")
    }
    if r.TimeRange.IsZero() {
        return errors.New("time_range is required")
    }
    return nil
}
```

### Query Builders

```go
// core/pkg/service/ranger/query.gen.go

// Code generated by oracle. DO NOT EDIT.

package ranger

// ---- Name Filters ----

type NameOp uint8

const (
    NameEq NameOp = iota
    NameNeq
    NameContains
    NameStartsWith
)

type NameFilter struct {
    Value string
    Op    NameOp
}

func (f NameFilter) isRangeFilter() {}

type nameBuilder struct{ v string }

func Name(v string) nameBuilder { return nameBuilder{v} }

func (b nameBuilder) Eq() NameFilter       { return NameFilter{b.v, NameEq} }
func (b nameBuilder) Neq() NameFilter      { return NameFilter{b.v, NameNeq} }
func (b nameBuilder) Contains() NameFilter { return NameFilter{b.v, NameContains} }
func (b nameBuilder) StartsWith() NameFilter { return NameFilter{b.v, NameStartsWith} }

// ---- Labels Filters ----

type LabelsOp uint8

const (
    LabelsHasAny LabelsOp = iota
    LabelsHasAll
    LabelsHasNone
)

type LabelsFilter struct {
    Values []uuid.UUID
    Op     LabelsOp
}

func (f LabelsFilter) isRangeFilter() {}

type labelsBuilder struct{ v []uuid.UUID }

func Labels(v ...uuid.UUID) labelsBuilder { return labelsBuilder{v} }

func (b labelsBuilder) HasAny() LabelsFilter  { return LabelsFilter{b.v, LabelsHasAny} }
func (b labelsBuilder) HasAll() LabelsFilter  { return LabelsFilter{b.v, LabelsHasAll} }
func (b labelsBuilder) HasNone() LabelsFilter { return LabelsFilter{b.v, LabelsHasNone} }

// ---- Composition ----

type Filter interface {
    isRangeFilter()
}

type AndFilter []Filter
type OrFilter []Filter

func (AndFilter) isRangeFilter() {}
func (OrFilter) isRangeFilter()  {}

func And(f ...Filter) AndFilter { return f }
func Or(f ...Filter) OrFilter   { return f }

// ---- Sort ----

type SortField uint8

const (
    SortByCreatedAt SortField = iota
)

type SortDir uint8

const (
    Asc SortDir = iota
    Desc
)
```

### Index Configuration

```go
// core/pkg/service/ranger/index.gen.go

// Code generated by oracle. DO NOT EDIT.

package ranger

import "github.com/synnaxlabs/x/gorp"

func ConfigureIndexes(mgr *gorp.IndexManager[uuid.UUID, Range]) {
    mgr.RegisterLookup("name", func(r Range) any { return r.Name })

    mgr.RegisterMultiLookup("labels", func(r Range) []any {
        out := make([]any, len(r.Labels))
        for i, l := range r.Labels {
            out[i] = l
        }
        return out
    })

    mgr.RegisterSorted("created_at",
        func(r Range) any { return r.CreatedAt },
        func(a, b any) int {
            return a.(telem.TimeStamp).Compare(b.(telem.TimeStamp))
        },
    )
}
```

## 5.1 - TypeScript

### Types + Zod

```typescript
// client/ts/src/ranger/types.gen.ts

// Code generated by oracle. DO NOT EDIT.

import { z } from "zod";
import { TimeRange, timeRangeZ } from "@synnaxlabs/x/telem";

export const keyZ = z.string().uuid();
export type Key = z.infer<typeof keyZ>;

export const rangeZ = z.object({
    key: keyZ,
    name: z.string().min(1).max(255),
    labels: z.array(z.string().uuid()),
    timeRange: timeRangeZ,
    createdAt: z.number(),
    parent: keyZ.nullable().optional(),
});

export type Range = z.infer<typeof rangeZ>;
```

### Query Builders

```typescript
// client/ts/src/ranger/query.gen.ts

// Code generated by oracle. DO NOT EDIT.

export type NameOp = "eq" | "neq" | "contains" | "starts_with";
export type LabelsOp = "has_any" | "has_all" | "has_none";
export type CreatedAtOp = "eq" | "gt" | "gte" | "lt" | "lte" | "between";

export interface NameFilter {
    readonly field: "name";
    readonly op: NameOp;
    readonly value: string;
}

export interface LabelsFilter {
    readonly field: "labels";
    readonly op: LabelsOp;
    readonly value: string[];
}

export interface CreatedAtFilter {
    readonly field: "created_at";
    readonly op: CreatedAtOp;
    readonly value: number | [number, number];
}

export type RangeFilter =
    | NameFilter
    | LabelsFilter
    | CreatedAtFilter
    | { readonly and: RangeFilter[] }
    | { readonly or: RangeFilter[] };

class NameBuilder {
    constructor(private value: string) {}
    eq(): NameFilter { return { field: "name", op: "eq", value: this.value }; }
    neq(): NameFilter { return { field: "name", op: "neq", value: this.value }; }
    contains(): NameFilter { return { field: "name", op: "contains", value: this.value }; }
    startsWith(): NameFilter { return { field: "name", op: "starts_with", value: this.value }; }
}

class LabelsBuilder {
    constructor(private values: string[]) {}
    hasAny(): LabelsFilter { return { field: "labels", op: "has_any", value: this.values }; }
    hasAll(): LabelsFilter { return { field: "labels", op: "has_all", value: this.values }; }
    hasNone(): LabelsFilter { return { field: "labels", op: "has_none", value: this.values }; }
}

export const Range = {
    name: (v: string) => new NameBuilder(v),
    labels: (...v: string[]) => new LabelsBuilder(v),
    createdAt: (v: number | Date) => new CreatedAtBuilder(typeof v === "number" ? v : v.getTime()),
    and: (...f: RangeFilter[]): { and: RangeFilter[] } => ({ and: f }),
    or: (...f: RangeFilter[]): { or: RangeFilter[] } => ({ or: f }),
} as const;

export type RangeSortField = "created_at";
export type SortDir = "asc" | "desc";

export interface RangeRetrieveParams {
    where?: RangeFilter;
    orderBy?: RangeSortField;
    orderDir?: SortDir;
    cursor?: string;
    limit?: number;
}
```

## 5.2 - Python

### Types + Pydantic

```python
# client/py/synnax/ranger/types.gen.py

# Code generated by oracle. DO NOT EDIT.

from __future__ import annotations
from pydantic import BaseModel, Field
from typing import Optional
from uuid import UUID

class Range(BaseModel):
    key: UUID
    name: str = Field(..., min_length=1, max_length=255)
    labels: list[UUID] = Field(default_factory=list)
    time_range: TimeRange
    created_at: int
    parent: Optional[UUID] = None
```

---

# 6 - File Organization

## 6.0 - Schema Location

Schemas use a **flat directory structure** at the repository root:

```
synnax/
├── schema/                          # All schemas (flat)
│   ├── ranger.oracle
│   ├── channel.oracle
│   ├── user.oracle
│   ├── workspace.oracle
│   ├── label.oracle
│   ├── framer.oracle
│   ├── device.oracle
│   ├── rack.oracle
│   ├── task.oracle
│   ├── schematic.oracle
│   ├── lineplot.oracle
│   ├── table.oracle
│   ├── log.oracle
│   ├── telem.oracle
│   └── ontology.oracle
```

## 6.1 - Output Configuration via Domains

Output locations are **not** configured in a separate config file. Instead, each struct
declares where its generated code should go using **struct-level domains**:

```oracle
// schema/ranger.oracle

import "schema/label"

struct Range {
    field key uuid {
        domain id
    }

    field name string {
        domain validate {
            required
            max_length 255
        }
    }

    field labels uuid[] {
        domain relation {
            target label.Label
        }
    }

    // Output configuration per target language
    domain go {
        output "core/ranger"
    }

    domain ts {
        output "console/src/ranger"
    }

    domain python {
        output "client/py/synnax/ranger"
    }
}
```

This approach has several benefits:

1. **Self-contained** - Each schema defines where its generated code goes
2. **No separate config file** - One less file to maintain
3. **Explicit** - Clear exactly where code lands when reading the schema
4. **Follows existing patterns** - Uses the same domain syntax as other concerns
5. **Per-struct control** - Different structs can output to different locations

Plugins read their domain's `output` expression to determine where to write generated
files. If a struct doesn't have a domain for a particular plugin, that plugin skips
generation for that struct.

## 6.2 - Generated Output

```
core/pkg/service/ranger/
├── types.gen.go          # Generated
├── query.gen.go          # Generated
├── index.gen.go          # Generated
├── service.go            # Hand-written
├── writer.go             # Hand-written
└── retrieve.go           # Hand-written (uses generated query types)

client/ts/src/ranger/
├── types.gen.ts          # Generated
├── query.gen.ts          # Generated
└── client.ts             # Hand-written

client/py/synnax/ranger/
├── types.gen.py          # Generated
├── query.gen.py          # Generated
└── client.py             # Hand-written
```

---

# 7 - ANTLR Grammar

## 7.0 - Lexer (OracleLexer.g4)

```antlr
lexer grammar OracleLexer;

// Keywords (only these are reserved)
STRUCT      : 'struct' ;
FIELD       : 'field' ;
DOMAIN      : 'domain' ;
ENUM        : 'enum' ;
IMPORT      : 'import' ;

// Symbols
LBRACE      : '{' ;
RBRACE      : '}' ;
LBRACKET    : '[' ;
RBRACKET    : ']' ;
QUESTION    : '?' ;
DOT         : '.' ;
EQUALS      : '=' ;

// Literals
STRING_LIT  : '"' (~["\r\n\\] | '\\' .)* '"' ;
INT_LIT     : '-'? [0-9]+ ;
FLOAT_LIT   : '-'? [0-9]+ '.' [0-9]+ ;
BOOL_LIT    : 'true' | 'false' ;

// Identifiers (includes primitives - they are not reserved)
IDENT       : [a-zA-Z_][a-zA-Z0-9_]* ;

// Skip
WS          : [ \t\r\n]+ -> skip ;
COMMENT     : '//' ~[\r\n]* -> skip ;
BLOCK_COMMENT : '/*' .*? '*/' -> skip ;
```

## 7.1 - Parser (OracleParser.g4)

```antlr
parser grammar OracleParser;

options { tokenVocab = OracleLexer; }

// Entry point
schema
    : import_stmt* definition* EOF
    ;

import_stmt
    : IMPORT STRING_LIT
    ;

definition
    : struct_def
    | enum_def
    ;

// struct Range { ... }
struct_def
    : STRUCT IDENT LBRACE struct_body RBRACE
    ;

struct_body
    : (field_def | domain_def)*
    ;

// field name string { ... }
field_def
    : FIELD IDENT type_ref (LBRACE domain_def* RBRACE)?
    ;

// domain validate { ... }
domain_def
    : DOMAIN IDENT (LBRACE expression* RBRACE)?
    ;

// Type reference with modifiers
type_ref
    : base_type (LBRACKET RBRACKET)? QUESTION?
    ;

base_type
    : qualified_ident
    ;

qualified_ident
    : IDENT (DOT IDENT)*
    ;

// Expression (flexible - identifier with optional values)
expression
    : IDENT expression_value*
    ;

expression_value
    : IDENT
    | STRING_LIT
    | INT_LIT
    | FLOAT_LIT
    | BOOL_LIT
    ;

// enum TaskState { pending = 0, running = 1 }
enum_def
    : ENUM IDENT LBRACE enum_value* RBRACE
    ;

enum_value
    : IDENT EQUALS (INT_LIT | STRING_LIT)
    ;
```

---

# 8 - Implementation Plan

## Phase 1: Core + Parser

1. Create `oracle/` module structure
2. Write ANTLR grammar files
3. Generate parser with `go generate`
4. Implement parser wrapper with error handling
5. Implement analyzer (import resolution, reference resolution)
6. Write tests with sample schemas

**Deliverable:** Can parse `.oracle` files and report errors

## Phase 2: Go Plugins

Each Go plugin is implemented independently with clear separation of concerns:

### 2.1: go/types Plugin

1. Implement struct type generation
2. Handle `go` domain for output path
3. Handle `id` domain for GorpKey() method
4. Generate proper import statements
5. Generate JSON/msgpack struct tags

**Deliverable:** `types.gen.go` with struct definitions

### 2.2: go/validate Plugin

1. Implement validation code generation
2. Handle `validate` domain expressions (`required`, `max_length`, etc.)
3. Generate `Validate() error` method on structs
4. Requires: `go/types`

**Deliverable:** Validation methods in `types.gen.go`

### 2.3: go/query Plugin

1. Implement filter type generation per field
2. Handle `query` domain expressions (`eq`, `contains`, `has_any`, etc.)
3. Generate builder pattern for filters
4. Generate `And`/`Or` composition types
5. Requires: `go/types`

**Deliverable:** `query.gen.go` with type-safe query builders

### 2.4: go/index Plugin

1. Implement index configuration generation
2. Handle `index` domain expressions (`lookup`, `sorted`, `range`)
3. Handle struct-level `index` domain (`composite`, `unique`)
4. Generate `ConfigureIndexes()` function
5. Requires: `go/types`

**Deliverable:** `index.gen.go` with gorp index configuration

## Phase 3: TypeScript Plugins

### 3.1: ts/types Plugin

1. Implement TypeScript type generation
2. Handle `ts` domain for output path
3. Generate proper naming conventions (camelCase)

**Deliverable:** `types.gen.ts`

### 3.2: zod Plugin (existing)

1. Already implemented
2. Generates Zod validation schemas

**Deliverable:** `schema.gen.ts`

### 3.3: ts/query Plugin

1. Implement TypeScript query builder generation
2. Generate filter interfaces and builder classes
3. Requires: `ts/types`

**Deliverable:** `query.gen.ts`

## Phase 4: Python Plugins

### 4.1: py/types Plugin

1. Implement Python type generation
2. Handle `python` domain for output path

**Deliverable:** `types.gen.py`

### 4.2: py/pydantic Plugin

1. Implement Pydantic model generation
2. Handle `validate` domain for Field constraints

**Deliverable:** Pydantic models

### 4.3: py/query Plugin

1. Implement Python query builder generation
2. Requires: `py/types`

**Deliverable:** `query.gen.py`

## Phase 5: Integration

1. Write schemas for all existing structs
2. Replace hand-written types with generated code
3. Update services to use generated query builders
4. Update gorp to support index manager
5. Deprecate old code paths

**Deliverable:** Full adoption across codebase

## Phase 6: Migrations (Future)

1. Design migration schema syntax
2. Implement migration diffing
3. Implement migration runner generation

---

# 9 - Open Questions

## 9.0 - Pagination Protocol

How should cursor-based pagination work across the stack?

```go
// Option A: Opaque cursor string
Page(cursor string, limit int)

// Option B: Typed cursor
Page(cursor *RangeCursor, limit int)
```

## 9.1 - Migration Strategy

Should migrations be:

1. **Declarative** - Version history in schema, migrations derived from diffs
2. **Imperative** - Explicit migration functions

## 9.2 - Nested Type Queries

Should we support querying into nested types?

```go
// Query by nested field
Schematic.Where(Nodes.Type.Eq("valve"))
```

Or keep nested types non-queryable (simpler)?

## 9.3 - Proto Generation

Should Oracle generate `.proto` files, or should proto remain separate?

## 9.4 - Relationship Cascade

How should deletes cascade through relationships?

```oracle
struct Range {
    // Struct-level domain
    domain on_delete {
        labels detach
        children cascade
    }
}
```

---

# 10 - References

- [RFC 0025 - Meta Data Structures](./0025-251214-meta-data.md)
- [Prisma Schema Language](https://www.prisma.io/docs/orm/prisma-schema)
- [ANTLR 4](https://www.antlr.org/)
- [Arc Grammar (internal)](../../arc/go/parser/)
