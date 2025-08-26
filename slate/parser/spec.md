# Slate Language Specification

## Channel Operations

### Writing to Channels
```slate
value -> channel      // Write value to channel
channel <- value      // Equivalent - write value to channel
```

### Reading from Channels
```slate
value := <- channel   // Read value from channel
```

### Channel Piping
```slate
channel_one <- channel_two    // Pipe channel_two to channel_one
channel_two -> channel_one    // Equivalent - same data flow
```

## Variable Assignment

### Local Variables
```slate
value := 0    // Local variable (function-scoped)
```

### Stateful Variables
```slate
value $= 0    // Persistent across reactive function executions
```

## Functions

### Basic Function Definition
```slate
func (arg1 number, arg2 number, arg3 number) {
    // Function body
}
```

### Channel-Typed Parameters
```slate
func (read_chan <-chan, write_chan ->chan) {
    // read_chan can only be read from
    // write_chan can only be written to
}
```

### Return Types
- Functions without explicit return type are `void`
- Return types can be specified: `func () number { ... }`

## Control Flow

### Supported Statements
- `if` statements
- `else if` statements  
- `else` statements

### Restrictions
- **No loops**: No `while`, `for`, or other loop constructs
- **Function-only**: Control flow statements only allowed inside functions
- **No top-level variables**: Variables can only be defined inside functions
- **No closures**: Functions cannot capture variables from outer scopes
- **No variable shadowing**: Variable names must be unique within their scope
- **No recursion**: Functions cannot call themselves (directly or indirectly)
- **No generics**: All types must be concrete and explicit

## Reactive Execution Model

### Top-Level Reactive Calls
```slate
ox_pt_2 -> calculation()    // Executes when ox_pt_2 value changes
```

Top-level expressions create reactive bindings that automatically execute when their input channels update, enabling real-time data processing without explicit scheduling.