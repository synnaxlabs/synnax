# Arc Module System Specification

This document specifies the module system for the Arc programming language. Modules
provide namespaced access to standard library functions and constants.

## Overview

Modules organize related functionality into named units. Arc programs import modules
explicitly and access their members using qualified names.

```arc
import ( math time )

func main() f64 {
    start := time.now()
    return math.sqrt(16.0)
}
```

**Key characteristics:**

- Modules are imported explicitly via `import` blocks
- Members are accessed using qualified names: `module.member`
- Modules can be aliased on import: `import ( math as m )`
- All imports must appear at the top of the file
- Unused imports are compile errors
- User-defined modules are not supported (future consideration)

## Import Syntax

### Grammar

```
Program ::= ImportBlock? TopLevelItem*

ImportBlock ::= 'import' '(' ImportItem+ ')'

ImportItem ::= ModulePath ('as' Identifier)?

ModulePath ::= Identifier ('.' Identifier)*
```

The optional `as Identifier` clause provides an alias for the imported module.

### Import Block

The import block declares which modules are used in the program. It must appear before
any other declarations.

```arc
import (
    math
    time
)

func main() f64 {
    return math.sqrt(4.0)
}
```

Multiple modules are listed within a single `import ( )` block, separated by whitespace.
There is no separator (comma) between module names.

### Module Paths

Module paths identify modules by name. Paths can be simple (single identifier) or
hierarchical (dot-separated identifiers).

```arc
import (
    math            // Simple path
    math.trig       // Hierarchical path
    math.stats      // Another sub-module
)
```

### Import Aliases

The `as` keyword allows renaming a module on import. The alias becomes the local
qualifier used to access the module's members.

```arc
import ( math as m )

func example() f64 {
    return m.sqrt(16.0)       // Use alias 'm' instead of 'math'
}
```

Aliases are useful for:

- **Brevity**: Shortening frequently-used module names
- **Conflict resolution**: Disambiguating modules with the same final path segment
- **Clarity**: Giving context-specific names to generic modules

```arc
import (
    math.trig as trig
    physics.trig as ptrig     // Resolve conflict with alias
)

func example() f64 {
    x := trig.sin(1.0)        // math.trig
    y := ptrig.angle(2.0)     // physics.trig
    return x + y
}
```

When an alias is provided, **only the alias** can be used to access the module. The
original path and default qualifier are not available:

```arc
import ( math as m )

func example() f64 {
    x := m.sqrt(4.0)          // OK: alias is valid
    y := math.sqrt(4.0)       // Error: 'math' is not defined
    return x + y
}
```

## Member Access

### Grammar

```
PostfixExpression ::= PrimaryExpression (MemberAccess | IndexOrSlice | FunctionCall)*

MemberAccess ::= '.' Identifier
```

### Qualified Names

Module members (functions and constants) are accessed using dot notation:

```arc
import ( math )

func example() f64 {
    x := math.sqrt(16.0)      // Function call
    area := math.PI * r ^ 2   // Constant access (future)
    return x + area
}
```

Members cannot be imported directly into the local namespace. The module qualifier is
always required.

```arc
// NOT supported:
// import ( sqrt from math )
// x := sqrt(16.0)

// Must use qualified name:
import ( math )
x := math.sqrt(16.0)
```

## Hierarchical Modules

### Qualifier Resolution

When importing a hierarchical module path, the **last segment** becomes the local
qualifier:

```arc
import ( math.trig )

func example() f64 {
    return trig.sin(1.0)      // 'trig' is the qualifier
}
```

The full path is not required for member access unless there is ambiguity.

### Parent and Child Modules

Importing a sub-module does not automatically import the parent, and vice versa:

```arc
import ( math.trig )

func example() f64 {
    x := trig.sin(1.0)        // OK: trig is imported
    y := math.sqrt(4.0)       // Error: math is not imported
    return x + y
}
```

To use both, import both explicitly:

```arc
import (
    math
    math.trig
)

func example() f64 {
    x := math.sqrt(4.0)       // OK
    y := trig.sin(1.0)        // OK
    return x + y
}
```

### Ambiguous Qualifiers

If multiple imports produce the same qualifier (last path segment), the short form is
ambiguous. **The recommended solution is to use aliases**:

```arc
import (
    math.trig as mtrig
    physics.trig as ptrig
)

func example() f64 {
    x := mtrig.sin(1.0)             // Clear and concise
    y := ptrig.angle(2.0)
    return x + y
}
```

Alternatively, the full path can be used without aliases:

```arc
import (
    math.trig
    physics.trig
)

func example() f64 {
    // x := trig.sin(1.0)           // Error: 'trig' is ambiguous
    x := math.trig.sin(1.0)         // OK: explicit full path
    y := physics.trig.angle(2.0)    // OK: explicit full path
    return x + y
}
```

The ambiguity error occurs at the point of use, not at import time. This allows
importing multiple modules with the same qualifier if only one is actually used.

## Module Contents

Modules can export:

- **Functions** - Callable operations (e.g., `math.sqrt`)
- **Constants** - Named values (e.g., `math.PI`) _(future)_

### Functions

Module functions are called using standard function call syntax:

```arc
result := math.sqrt(16.0)
clamped := math.clamp(value, 0.0, 100.0)
```

Module functions use concrete types. For mathematical functions, this is typically
`f64`. Explicit casts are required for other numeric types:

```arc
x f32 := sensor_value
y := math.sqrt(f64(x))        // Cast f32 to f64
z f32 := f32(y)               // Cast result back if needed
```

### Constants

_(Future feature - requires global constant support in Arc)_

Module constants are accessed like functions but without parentheses:

```arc
area := math.PI * radius ^ 2
growth := math.E ^ rate
```

## Scoping and Name Resolution

### Resolution Order

When resolving an identifier, Arc searches in order:

1. Local scope (variables, parameters)
2. Enclosing function scope
3. Global scope (functions, sequences, channels, imported modules)

### Shadowing

Local identifiers can shadow module qualifiers (including aliases). The innermost
binding takes precedence:

```arc
import ( math )

func example() f64 {
    math := 5.0                   // Shadows the module
    return math + 1.0             // Uses local variable (6.0)
}

func other() f64 {
    return math.sqrt(4.0)         // Module accessible here
}
```

The same applies to aliases:

```arc
import ( math as m )

func example() f64 {
    m := 5.0                      // Shadows the alias
    return m + 1.0                // Uses local variable (6.0)
}
```

When a module is shadowed, its members become inaccessible within that scope. Attempting
to access a member on a non-module value is a type error:

```arc
import ( math )

func example() f64 {
    math := 5.0
    return math.sqrt(4.0)         // Error: f64 has no member 'sqrt'
}
```

### Reserved Names

Module qualifiers and aliases are not reserved words. They can be shadowed by local
declarations, though this is discouraged for readability.

## Compile-Time Errors

The following are compile-time errors:

### Unknown Module

```arc
import ( nonexistent )            // Error: unknown module 'nonexistent'
```

### Duplicate Import

```arc
import (
    math
    math                          // Error: duplicate import 'math'
)
```

### Duplicate Alias

```arc
import (
    math as m
    time as m                     // Error: duplicate alias 'm'
)
```

### Alias Conflicts with Qualifier

```arc
import (
    math
    time as math                  // Error: alias 'math' conflicts with import 'math'
)
```

### Duplicate Qualifier

Importing modules that produce the same qualifier and then using the short form:

```arc
import (
    math.trig
    physics.trig
)

x := trig.sin(1.0)                // Error: 'trig' is ambiguous
```

### Unused Import

```arc
import ( math time )              // Error: 'time' imported but never used

func main() f64 {
    return math.sqrt(4.0)
}
```

### Unknown Member

```arc
import ( math )

x := math.nonexistent(1.0)        // Error: module 'math' has no member 'nonexistent'
```

### Import Position

```arc
func helper() f64 { return 1.0 }

import ( math )                   // Error: import must appear before declarations

func main() f64 {
    return math.sqrt(4.0)
}
```

## Built-in Functions

The following functions remain global built-ins and do not require imports:

- `len(series)` - Returns the length of a series as `i64`
- `len(str)` - Returns the length of a string as `i64`

```arc
// No import required
data := [1.0, 2.0, 3.0]
length := len(data)
```

The `now()` function is part of the `time` module and requires an import:

```arc
import ( time )

timestamp := time.now()
```

## Future Considerations

The following features are not supported in the current version but may be added in
future versions:

### Selective Imports

Importing specific members directly into the local namespace:

```arc
// NOT currently supported
import ( sqrt from math )
x := sqrt(4.0)
```

### User-Defined Modules

Defining custom modules within Arc source code:

```arc
// NOT currently supported
module filters {
    func lowpass(x f64, cutoff f64) f64 { ... }
}
```

## Standard Library

The contents of standard library modules (available functions, constants, and their
signatures) are specified in a separate document.
