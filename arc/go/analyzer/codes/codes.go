// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package codes declares the diagnostic kinds of the Arc compiler.
package codes

//go:generate go run gen.go

import (
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

var (
	// Syntax is source text that the parser rejects.
	Syntax = diagnostics.Kind{
		Code:     "E1000",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "syntax error",
		Problem:  "The syntax is not correct at that position.",
		Example: `func f() {
    x := (1 + 2
}`,
	}
	// SourceLimit is source text that is too large or too deeply nested to parse.
	SourceLimit = diagnostics.Kind{
		Code:     "E1001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "source limit exceeded",
		Problem: "The source text is larger than 1 MiB, or its nesting is deeper " +
			"than the parser allows.",
	}
	// InvalidStringLiteral is a malformed string literal or format placeholder.
	InvalidStringLiteral = diagnostics.Kind{
		Code:     "E1002",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid string literal",
		Problem:  "A string literal or a format placeholder is not valid.",
		Example: `func f() str {
    return f"value {}"
}`,
	}
	// SubtractionSpacing is a binary minus with no space on one of its sides.
	SubtractionSpacing = diagnostics.Kind{
		Code:     "E1003",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "subtraction requires whitespace on both sides of '-'",
		Problem: "Only possible when the flag for channel names with dashes is " +
			"on. Then `a-b` is a channel name, so a subtraction needs a space on " +
			"each side of the `-`.",
		Example: `func f(a f64, b f64) f64 {
    return a -b
}`,
	}

	// DivisionByZero is a constant integer division or modulo by zero.
	DivisionByZero = diagnostics.Kind{
		Code:     "E2000",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "division by zero",
		Problem: "An integer `/` or `%` divides by a zero that is made only of " +
			"literals.",
		Example: `func f() {
    x := 1 / 0
}`,
	}
	// TypeMismatch is a value of one type where the code needs a different type.
	TypeMismatch = diagnostics.Kind{
		Code:     "E2001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "type mismatch",
		Problem:  "A value has one type and the code needs a different type.",
		Example: `func f() {
    x i32 := 1
    y str := x
}`,
	}
	// UnknownUnit is a unit name that does not exist.
	UnknownUnit = diagnostics.Kind{
		Code:     "E2002",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "unknown unit",
		Problem:  "A unit name does not exist.",
		Example: `func f(x f64 = 5zzz) {
}`,
	}
	// UnresolvedType is a value that has no type the compiler can determine.
	UnresolvedType = diagnostics.Kind{
		Code:     "E2003",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "unresolved type",
		Problem:  "The compiler cannot determine the type of a value from the code.",
	}
	// InvalidOperand is an operator or statement on a type that it does not support.
	InvalidOperand = diagnostics.Kind{
		Code:     "E2004",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid operand",
		Problem:  "An operator or a statement does not support the type of a value.",
		Example: `func f() {
    x := "a" - "b"
}`,
	}
	// InvalidCast is a cast between two types that Arc does not convert.
	InvalidCast = diagnostics.Kind{
		Code:     "E2005",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid cast",
		Problem:  "A cast converts between two types that Arc cannot convert.",
		Example: `func f() {
    x := i32("a")
}`,
	}
	// NotConstant is a computed value where the code needs a constant.
	NotConstant = diagnostics.Kind{
		Code:     "E2006",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "not a constant",
		Problem: "The code needs a constant, and the value is one that the " +
			"compiler cannot compute. A variable outside a function also " +
			"needs a constant when its type has no zero value, such as a " +
			"series.",
		Example: `k := 1 + 2
func f() i64 {
    return k
}`,
	}
	// ConstantOverflow is a constant that does not fit its type.
	ConstantOverflow = diagnostics.Kind{
		Code:     "E2007",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "constant overflow",
		Problem:  "A constant does not fit its type.",
		Example: `func f() {
    x i8 := 100 + 100
}`,
	}
	// DimensionMismatch is an operation on units that have different dimensions.
	DimensionMismatch = diagnostics.Kind{
		Code:     "E2008",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "incompatible dimensions",
		Problem: "An operation combines units of different dimensions, such as a " +
			"time and a length.",
		Example: `func f() {
    x := 5s + 3m
}`,
	}
	// InvalidLiteral is a number literal that is malformed or has no exact value.
	InvalidLiteral = diagnostics.Kind{
		Code:     "E2009",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid literal",
		Problem: "A number literal is malformed, or it has no exact value in its " +
			"type.",
		Example: `func f(x i32 = 1.5) {
}`,
	}
	// NaNComparison is a comparison with a NaN constant, which always has one result.
	NaNComparison = diagnostics.Kind{
		Code:     "E2010",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "comparison with NaN",
		Problem: "A comparison has a NaN constant as an operand. NaN is not equal " +
			"to, less than, or greater than a value, so the comparison always " +
			"has the same result.",
		Example: `import math

func f(x f64) u8 {
    return x == math.nan
}`,
	}
	// UnitMissing is a value with a unit that goes to a name with no unit.
	UnitMissing = diagnostics.Kind{
		Code:     "E2011",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "missing unit",
		Problem: "A value with a unit goes to a variable, a parameter, or a " +
			"function result that has no unit.",
		Example: `func f(t i64 s) i64 {
    return t
}`,
	}
	// TruthyCondition is a number or a string used as a condition.
	TruthyCondition = diagnostics.Kind{
		Code:     "W2000",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "non-boolean condition",
		Problem:  "An `if` or a `=>` uses a non-boolean value as a condition.",
		Example: `func f(x i32) {
    if x {
    }
}`,
	}
	// BoolCast is a cast of a bool to a number type other than u8.
	BoolCast = diagnostics.Kind{
		Code:     "W2001",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "bool cast",
		Problem:  "A cast converts a `bool` to a number type other than `u8`.",
		Example: `func f() {
    b := true
    x := i32(b)
}`,
	}
	// UnitPrecisionLoss is a unit operation that can lose precision or overflow.
	UnitPrecisionLoss = diagnostics.Kind{
		Code:     "W2002",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "unit precision loss",
		Problem: "An operation on values with units can cause truncation or " +
			"overflow.",
		Example: `func f() {
    x := 1K + 1pK
}`,
	}
	// UnitDropped is a cast with no unit of a value that goes to another unit.
	UnitDropped = diagnostics.Kind{
		Code:     "W2003",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "unit dropped",
		Problem: "A cast with no unit removes the unit of a value before the value " +
			"goes to a different unit. `i64(5s)` written to a name in `ms` gives " +
			"`5`, not `5000`.",
		Example: `func f() {
    d i64 ms := i64(5s)
}`,
	}

	// NotCallable is a call of a name that the code cannot call there.
	NotCallable = diagnostics.Kind{
		Code:     "E3000",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "not callable",
		Problem: "The code calls a name that is not a function, or calls a " +
			"function in a place that does not allow that call.",
		Example: `func f() {
    x := 1
    y := x()
}`,
	}
	// FuncArgMissing is a call with no argument for a required parameter.
	FuncArgMissing = diagnostics.Kind{
		Code:     "E3001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "missing required argument",
		Problem: "A call has no argument for a parameter that has no default " +
			"value.",
		Example: `func add(x f64, y f64) f64 {
    return x + y
}

func f() {
    z := add(1.0)
}`,
	}
	// FuncArgExtra is a call with more arguments than parameters.
	FuncArgExtra = diagnostics.Kind{
		Code:     "E3002",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "too many arguments",
		Problem:  "A call has more arguments than the function has parameters.",
		Example: `func add(x f64, y f64) f64 {
    return x + y
}

func f() {
    z := add(1.0, 2.0, 3.0)
}`,
	}
	// FuncArgDuplicate is a call with two arguments of the same name.
	FuncArgDuplicate = diagnostics.Kind{
		Code:     "E3003",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "duplicate argument",
		Problem:  "Two arguments of a call have the same name.",
		Example:  `len(input=label, input=label) -> count`,
	}
	// FuncArgUnknown is a named argument that the function has no parameter for.
	FuncArgUnknown = diagnostics.Kind{
		Code:     "E3004",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "unknown parameter",
		Problem:  "A call names a parameter that the function does not have.",
		Example:  `len(unknown=label) -> count`,
	}
	// FuncArgValue is an argument with a value the function rejects.
	FuncArgValue = diagnostics.Kind{
		Code:     "E3005",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid argument value",
		Problem:  "An argument has a value that the function does not accept.",
		Example: `import time

time.interval(period=0ms) -> valve_cmd`,
	}
	// ParamOrder is a required parameter after an optional parameter.
	ParamOrder = diagnostics.Kind{
		Code:     "E3006",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid parameter order",
		Problem: "A parameter that has no default value comes after a parameter " +
			"that has one.",
		Example: `func f(a i32 = 1, b i32) {
}`,
	}
	// ReturnValue is a return with a value missing or a value not allowed.
	ReturnValue = diagnostics.Kind{
		Code:     "E3007",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid return",
		Problem: "A `return` has no value in a function that returns one, a " +
			"`return` has a value in a function that returns none, or a path " +
			"through a function ends with no `return`.",
		Example: `func f() {
    return 1
}`,
	}
	// BreakOutsideLoop is a break or a continue that is not in a for loop.
	BreakOutsideLoop = diagnostics.Kind{
		Code:     "E3008",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "break or continue outside a loop",
		Problem:  "A `break` or a `continue` is not in a `for` loop.",
		Example: `func f() {
    break
}`,
	}
	// StatefulVariable is a stateful variable in a place that does not allow one.
	StatefulVariable = diagnostics.Kind{
		Code:     "E3009",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid stateful variable",
		Problem: "A stateful variable is in a place that does not allow one, such " +
			"as the top level of a program.",
		Example: `count_total $= 0`,
	}
	// AssignmentTarget is an assignment to a target that does not accept it.
	AssignmentTarget = diagnostics.Kind{
		Code:     "E3010",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid assignment",
		Problem: "An assignment has a target that does not accept it, such as a " +
			"loop variable or a slice.",
		Example: `func f() {
    for i := range(3) {
        i = 1
    }
}`,
	}
	// CircularCall is a function that calls itself through a chain of calls.
	CircularCall = diagnostics.Kind{
		Code:     "E3011",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "circular function call",
		Problem:  "A function calls itself through a circular chain of calls.",
		Example: `func a() {
    b()
}

func b() {
    a()
}`,
	}
	// OutputNeverAssigned is a named output that no path of its function sets.
	OutputNeverAssigned = diagnostics.Kind{
		Code:     "W3000",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "output never assigned",
		Problem: "A function has a named output, and no path through the function " +
			"gives it a value.",
		Example: `func f() (out i32) {
}`,
	}

	// ImportFailed is an import of a module that does not exist or does not load.
	ImportFailed = diagnostics.Kind{
		Code:     "E4000",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "import failed",
		Problem: "An import names a module that does not exist, that does not " +
			"compile, or that imports itself through a chain of imports.",
		Example: `import pumps`,
	}
	// ImportOrder is an import after a different top-level declaration.
	ImportOrder = diagnostics.Kind{
		Code:     "E4001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "import after other code",
		Problem:  "An import comes after other code.",
		Example: `func f() {
}

import time`,
	}
	// DuplicateImport is an import of a name that an earlier import already has.
	DuplicateImport = diagnostics.Kind{
		Code:     "E4002",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "duplicate import",
		Problem:  "Two imports give the same name.",
		Example: `import math
import math`,
	}
	// NotExported is a reference to a member that its module does not export.
	NotExported = diagnostics.Kind{
		Code:     "E4003",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "not exported",
		Problem: "The code uses a member of a module, and the module does not " +
			"export that member.",
		Example: `import valves

sensor -> valves.open() -> count`,
	}
	// SymbolUndefined is a reference to a name that no scope declares.
	SymbolUndefined = diagnostics.Kind{
		Code:     "E4004",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "undefined symbol",
		Problem:  "The name is not declared.",
		Example: `func f() {
    x := y
}`,
	}
	// SymbolRedefined is a declaration of a name that its scope already has.
	SymbolRedefined = diagnostics.Kind{
		Code:     "E4005",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "name conflict",
		Problem: "A declaration has a name that its scope or an outer scope " +
			"already has.",
		Example: `x := 1
func f() {
    x := 2
}`,
	}
	// ModuleAuthority is an authority block in a module.
	ModuleAuthority = diagnostics.Kind{
		Code:     "E4006",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "a module cannot have an authority block",
		Problem:  "A module has an authority block.",
		Example:  `authority 255`,
	}
	// ModuleFlow is a top-level flow in a module.
	ModuleFlow = diagnostics.Kind{
		Code:     "E4007",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "a module cannot have a top-level flow",
		Problem:  "A module has a top-level flow.",
		Example:  `sensor -> sensor_avg`,
	}
	// ExportOutsideModule is an export in a file that is not a module.
	ExportOutsideModule = diagnostics.Kind{
		Code:     "E4008",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "export is only valid in a module",
		Problem:  "An `export` is in a file that is not a module.",
		Example: `export func f() {
}`,
	}
	// UnnamedExport is an export of a declaration that has no name.
	UnnamedExport = diagnostics.Kind{
		Code:     "E4009",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "an exported declaration needs a name",
		Problem:  "An exported declaration has no name.",
		Example: `export sequence {
    stage s {
    }
}`,
	}
	// UnusedImport is an imported module that nothing references.
	UnusedImport = diagnostics.Kind{
		Code:     "W4000",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "unused import",
		Problem:  "The code imports a module and does not use it.",
		Example:  `import math`,
	}
	// DeprecatedSymbol is a reference to a name that has a replacement.
	DeprecatedSymbol = diagnostics.Kind{
		Code:     "W4001",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "deprecated symbol",
		Problem:  "The code uses a name that has a replacement.",
		Example:  `sensor -> avg() -> sensor_avg`,
	}

	// AuthorityPlacement is an authority block after a function, flow, or sequence.
	AuthorityPlacement = diagnostics.Kind{
		Code:     "E5000",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "misplaced authority declaration",
		Problem: "An authority block comes after a function, a flow, or a " +
			"sequence.",
		Example: `func f() {
}

authority 100`,
	}
	// AuthorityDefault is an authority block with two default values.
	AuthorityDefault = diagnostics.Kind{
		Code:     "E5001",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "multiple default authority values",
		Problem:  "An authority block has two default values.",
		Example: `authority 200
authority 100`,
	}
	// AuthorityDuplicate is an authority block that names a channel two times.
	AuthorityDuplicate = diagnostics.Kind{
		Code:     "E5002",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "duplicate authority",
		Problem:  "An authority block names a channel two times.",
		Example:  `authority (valve_cmd 100 valve_cmd 150)`,
	}
	// AuthorityValue is an authority value that is not an integer from 0 to 255.
	AuthorityValue = diagnostics.Kind{
		Code:     "E5003",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid authority value",
		Problem:  "An authority value is not an integer from 0 to 255.",
		Example:  `authority 300`,
	}
	// FlowConnection is a flow node with an output or input that cannot connect.
	FlowConnection = diagnostics.Kind{
		Code:     "E5004",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid flow connection",
		Problem:  "An output or an input of a node in a flow cannot connect.",
		Example: `func source() {
}

func sink() {v u8} {
}

source() -> sink()`,
	}
	// TransitionTarget is a transition to a target that the code cannot reach.
	TransitionTarget = diagnostics.Kind{
		Code:     "E5005",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid transition target",
		Problem: "A transition has a target that the code cannot reach from there, " +
			"such as `next` outside a sequence or in the last stage.",
		Example: `sensor > 10 => next`,
	}
	// FlowOperator is a => where the flow needs a ->.
	FlowOperator = diagnostics.Kind{
		Code:     "E5006",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid flow operator",
		Problem:  "A `=>` sends values into `select` or into a routing table.",
		Example: `sensor -> select() => {
    true: 1 -> valve_cmd,
    false: 0 -> valve_cmd
}`,
	}
	// EmptySequence is a sequence that has no steps.
	EmptySequence = diagnostics.Kind{
		Code:     "E5007",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "empty sequence",
		Problem:  "A sequence has no steps.",
		Example: `sequence main {
}

sensor > 10 => main`,
	}
	// NamedInlineBody is an inline stage or sequence that has a name.
	NamedInlineBody = diagnostics.Kind{
		Code:     "E5008",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "an inline stage or sequence cannot have a name",
		Problem:  "An inline stage or sequence has a name.",
		Example: `sequence main {
    stage a {
        sensor -> select() -> {
            true: stage my_stage { 1 -> valve_cmd }
        }
    }
    stage b {
        1 -> valve_cmd
    }
}

sensor > 1 => main`,
	}
	// TriggerBinding is a function input with two sources or a missing parameter.
	TriggerBinding = diagnostics.Kind{
		Code:     "E5009",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid trigger binding",
		Problem:  "A function input has two sources.",
		Example: `func consumer() {v f64} {
}

sensor -> consumer(v=5.0)`,
	}
	// ChannelExpected is a value where the code needs a channel.
	ChannelExpected = diagnostics.Kind{
		Code:     "E5010",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "channel expected",
		Problem:  "The code has a value where it needs a channel.",
		Example: `import math

sensor -> math.nan`,
	}
	// ReadOnlyWrite is a write to a name that the code can only read.
	ReadOnlyWrite = diagnostics.Kind{
		Code:     "E5011",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "read-only write",
		Problem:  "The code writes to a name that is read-only.",
		Example: `sequence main {
    stage first {
        v := sensor * 2.0
        1.0 -> v
    }
}`,
	}
	// RoutingTable is a routing table that is malformed or out of place.
	RoutingTable = diagnostics.Kind{
		Code:     "E5012",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid routing table",
		Problem:  "A routing table is malformed or out of place.",
		Example: `func target(v f64) {
}

sensor -> {
    v: target()
}`,
	}
	// RoutingName is a routing table entry that names a missing output or input.
	RoutingName = diagnostics.Kind{
		Code:     "E5013",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "unknown routing name",
		Problem: "A routing table names an output or a parameter that does not " +
			"exist.",
		Example: `func split() {v f64} (low f64, high f64) {
    low = v
    high = v
}

sensor -> split() -> {
    low: 1.0 -> sensor_avg,
    middle: 1.0 -> sensor_avg
}`,
	}
	// DataflowCycle is a set of flows that feed each other in a loop.
	DataflowCycle = diagnostics.Kind{
		Code:     "E5014",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "dataflow cycle",
		Problem: "A set of flows feed each other in a loop, so no flow can run " +
			"first.",
	}
	// ChannelLimit is a program with more local channels than Arc allows.
	ChannelLimit = diagnostics.Kind{
		Code:     "E5015",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "channel limit exceeded",
		Problem:  "A program has more top-level variables than Arc allows.",
	}
	// BodyDeclaration is a stage or sequence declared in a form that is not valid.
	BodyDeclaration = diagnostics.Kind{
		Code:     "E5016",
		Severity: protocol.DiagnosticSeverityError,
		Message:  "invalid stage or sequence declaration",
		Problem: "Only a named top-level stage or sequence can have parameters. " +
			"A parameter list cannot be empty.",
		Example: `sequence main {
    stage hold(limit f64) {
    }
}`,
	}
	// DeclarationAfterStep is a variable declared after a step of its scope.
	DeclarationAfterStep = diagnostics.Kind{
		Code:     "W5000",
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "declare variables at the top of a stage or sequence",
		Problem: "A stage or a sequence declares a variable after one of its " +
			"steps. Declare each variable before the first step.",
		Example: `import time

sequence main {
    stage a {
        true => next
    }
    t0 := time.now()
    stage b {
        time.wait(1s)
    }
}

true => main`,
	}
)

// All is each kind of the table.
var All = []diagnostics.Kind{
	Syntax,
	SourceLimit,
	InvalidStringLiteral,
	SubtractionSpacing,
	DivisionByZero,
	TypeMismatch,
	UnknownUnit,
	UnresolvedType,
	InvalidOperand,
	InvalidCast,
	NotConstant,
	ConstantOverflow,
	DimensionMismatch,
	InvalidLiteral,
	NaNComparison,
	UnitMissing,
	TruthyCondition,
	BoolCast,
	UnitPrecisionLoss,
	UnitDropped,
	NotCallable,
	FuncArgMissing,
	FuncArgExtra,
	FuncArgDuplicate,
	FuncArgUnknown,
	FuncArgValue,
	ParamOrder,
	ReturnValue,
	BreakOutsideLoop,
	StatefulVariable,
	AssignmentTarget,
	CircularCall,
	OutputNeverAssigned,
	ImportFailed,
	ImportOrder,
	DuplicateImport,
	NotExported,
	SymbolUndefined,
	SymbolRedefined,
	ModuleAuthority,
	ModuleFlow,
	ExportOutsideModule,
	UnnamedExport,
	UnusedImport,
	DeprecatedSymbol,
	AuthorityPlacement,
	AuthorityDefault,
	AuthorityDuplicate,
	AuthorityValue,
	FlowConnection,
	TransitionTarget,
	FlowOperator,
	EmptySequence,
	NamedInlineBody,
	TriggerBinding,
	ChannelExpected,
	ReadOnlyWrite,
	RoutingTable,
	RoutingName,
	DataflowCycle,
	ChannelLimit,
	BodyDeclaration,
	DeclarationAfterStep,
}
