// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"strconv"
	"strings"

	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// Compiler compiles expressions to WASM bytecode
type Compiler struct {
	ctx     *compiler.Context
	encoder *wasm.Encoder
}

// NewCompiler creates a new expression compiler
func NewCompiler(ctx *compiler.Context) *Compiler {
	return &Compiler{ctx: ctx, encoder: wasm.NewEncoder()}
}

// Compile compiles an expression and returns its type
func (e *Compiler) Compile(expr parser.IExpressionContext) (types.Type, error) {
	// Main dispatch based on expression type
	// The grammar builds expressions in layers:
	// Expression -> LogicalOrExpression -> ... -> PrimaryExpression
	if expr == nil {
		return nil, errors.New("nil expression")
	}
	// Since Expression just wraps LogicalOrExpression, unwrap it
	if logicalOr := expr.LogicalOrExpression(); logicalOr != nil {
		return e.compileLogicalOr(logicalOr)
	}
	return nil, errors.New("unknown expression type")
}

// compileLogicalOr handles || operations
func (e *Compiler) compileLogicalOr(expr parser.ILogicalOrExpressionContext) (types.Type, error) {
	ands := expr.AllLogicalAndExpression()
	if len(ands) == 0 {
		return nil, errors.New("empty logical OR expression")
	}
	// Single expression - no OR operation
	if len(ands) == 1 {
		return e.compileLogicalAnd(ands[0])
	}
	// TODO: Implement || with short-circuit evaluation
	// For now, just compile the first expression
	return e.compileLogicalAnd(ands[0])
}

// compileLogicalAnd handles && operations
func (e *Compiler) compileLogicalAnd(expr parser.ILogicalAndExpressionContext) (types.Type, error) {
	eqs := expr.AllEqualityExpression()
	if len(eqs) == 0 {
		return nil, errors.New("empty logical AND expression")
	}
	// Single expression - no AND operation
	if len(eqs) == 1 {
		return e.compileEquality(eqs[0])
	}
	// TODO: Implement && with short-circuit evaluation
	// For now, just compile the first expression
	return e.compileEquality(eqs[0])
}

// compileEquality handles == and != operations
func (e *Compiler) compileEquality(expr parser.IEqualityExpressionContext) (types.Type, error) {
	rels := expr.AllRelationalExpression()
	if len(rels) == 0 {
		return nil, errors.New("empty equality expression")
	}
	// Single expression - no equality operation
	if len(rels) == 1 {
		return e.compileRelational(rels[0])
	}
	// Handle binary equality operations
	return e.compileBinaryEquality(expr)
}

// compileRelational handles <, >, <=, >= operations
func (e *Compiler) compileRelational(expr parser.IRelationalExpressionContext) (types.Type, error) {
	adds := expr.AllAdditiveExpression()
	if len(adds) == 0 {
		return nil, errors.New("empty relational expression")
	}
	// Single expression - no relational operation
	if len(adds) == 1 {
		return e.compileAdditive(adds[0])
	}
	// Handle binary relational operations
	return e.compileBinaryRelational(expr)
}

// compileAdditive handles + and - operations
func (e *Compiler) compileAdditive(expr parser.IAdditiveExpressionContext) (types.Type, error) {
	muls := expr.AllMultiplicativeExpression()
	if len(muls) == 0 {
		return nil, errors.New("empty additive expression")
	}
	// Single expression - no addition/subtraction
	if len(muls) == 1 {
		return e.compileMultiplicative(muls[0])
	}
	// Handle binary operations
	return e.compileBinaryAdditive(expr)
}

// compileMultiplicative handles *, /, % operations
func (e *Compiler) compileMultiplicative(expr parser.IMultiplicativeExpressionContext) (types.Type, error) {
	pows := expr.AllPowerExpression()
	if len(pows) == 0 {
		return nil, errors.New("empty multiplicative expression")
	}
	// Single expression - no multiplication/division
	if len(pows) == 1 {
		return e.compilePower(pows[0])
	}
	// Handle binary multiplication/division operations
	return e.compileBinaryMultiplicative(expr)
}

// compilePower handles ^ (exponentiation) operations
func (e *Compiler) compilePower(expr parser.IPowerExpressionContext) (types.Type, error) {
	unary := expr.UnaryExpression()
	if unary == nil {
		return nil, errors.New("empty power expression")
	}
	// Check if there's a power operation
	if expr.CARET() != nil && expr.PowerExpression() != nil {
		// TODO: Implement exponentiation (needs host function)
		return e.compileUnary(unary)
	}
	// No power operation
	return e.compileUnary(unary)
}

// compileUnary handles unary -, !, and blocking read operations
func (e *Compiler) compileUnary(expr parser.IUnaryExpressionContext) (types.Type, error) {
	// Check for unary minus
	if expr.MINUS() != nil {
		// Compile the inner expression first
		innerType, err := e.compileUnary(expr.UnaryExpression())
		if err != nil {
			return nil, err
		}
		// Emit negation based on type
		switch innerType.(type) {
		case types.I8, types.I16, types.I32, types.U8, types.U16, types.U32:
			// For 32-bit integers, negate by subtracting from 0
			// We need: 0 - value, so push 0, swap, then subtract
			// Actually, let's use a simpler approach: multiply by -1
			e.encoder.WriteI32Const(-1)
			e.encoder.WriteBinaryOp(wasm.OpI32Mul)
		case types.I64, types.U64:
			// For 64-bit integers
			e.encoder.WriteI64Const(-1)
			e.encoder.WriteBinaryOp(wasm.OpI64Mul)
		case types.F32:
			// For floats, use the native neg instruction
			e.encoder.WriteOpcode(wasm.OpF32Neg)
		case types.F64:
			e.encoder.WriteOpcode(wasm.OpF64Neg)
		default:
			return nil, errors.Newf("cannot negate type %s", innerType)
		}

		return innerType, nil
	}

	// Check for logical NOT
	if expr.NOT() != nil {
		// Compile the inner expression
		_, err := e.compileUnary(expr.UnaryExpression())
		if err != nil {
			return nil, err
		}
		// Logical NOT expects a boolean (u8) and returns boolean
		// Use i32.eqz to check if value is 0 (false becomes true, true becomes false)
		e.encoder.WriteOpcode(wasm.OpI32Eqz)
		return types.U8{}, nil
	}

	// Check for blocking read
	if blockRead := expr.BlockingReadExpr(); blockRead != nil {
		// TODO: Implement blocking channel read
		return types.F64{}, nil // Placeholder
	}
	// Must be a postfix expression
	if postfix := expr.PostfixExpression(); postfix != nil {
		return e.compilePostfix(postfix)
	}
	return nil, errors.New("unknown unary expression")
}

// compilePostfix handles array indexing, slicing, and function calls
func (e *Compiler) compilePostfix(expr parser.IPostfixExpressionContext) (types.Type, error) {
	primary := expr.PrimaryExpression()
	if primary == nil {
		return nil, errors.New("empty postfix expression")
	}
	// Start with the primary expression
	primaryType, err := e.compilePrimary(primary)
	if err != nil {
		return nil, err
	}
	// TODO: Handle postfix operations (indexing, function calls)
	// For now, just return the primary type
	return primaryType, nil
}

// compilePrimary handles literals, identifiers, parenthesized expressions, etc.
func (e *Compiler) compilePrimary(expr parser.IPrimaryExpressionContext) (types.Type, error) {
	// Check for literal
	if lit := expr.Literal(); lit != nil {
		return e.compileLiteral(lit)
	}
	// Check for identifier
	if expr.IDENTIFIER() != nil {
		return e.compileIdentifier(expr.IDENTIFIER().GetText())
	}
	// Check for parenthesized expression
	if expr.LPAREN() != nil && expr.Expression() != nil {
		return e.Compile(expr.Expression())
	}
	// Check for type cast
	if cast := expr.TypeCast(); cast != nil {
		// TODO: Implement type casting
		return nil, errors.New("type cast not yet implemented")
	}
	// Check for builtin function
	if builtin := expr.BuiltinFunction(); builtin != nil {
		// TODO: Implement builtin functions
		return nil, errors.New("builtin functions not yet implemented")
	}
	return nil, errors.New("unknown primary expression")
}

// compileLiteral compiles a literal value
func (e *Compiler) compileLiteral(lit parser.ILiteralContext) (types.Type, error) {
	// Check for numeric literal
	if num := lit.NumericLiteral(); num != nil {
		return e.compileNumericLiteral(num)
	}
	// Check for temporal literal
	if temp := lit.TemporalLiteral(); temp != nil {
		return types.TimeSpan{}, nil
	}
	// Check for string literal
	if str := lit.STRING_LITERAL(); str != nil {
		return types.String{}, nil
	}
	// Check for series literal
	if series := lit.SeriesLiteral(); series != nil {
		// TODO: Implement series literals
		return nil, errors.New("series literals not yet implemented")
	}
	return nil, errors.New("unknown literal type")
}

// compileNumericLiteral compiles integer and float literals
func (e *Compiler) compileNumericLiteral(num parser.INumericLiteralContext) (types.Type, error) {
	// Check for integer literal
	if intLit := num.INTEGER_LITERAL(); intLit != nil {
		text := intLit.GetText()
		// Parse type suffix if present
		var baseType string
		var value int64
		// Extract suffix
		suffixStart := len(text)
		for _, suffix := range []string{"i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64"} {
			if strings.HasSuffix(text, suffix) {
				baseType = suffix
				suffixStart = len(text) - len(suffix)
				break
			}
		}
		// Default to i64 if no suffix
		if baseType == "" {
			baseType = "i64"
		}
		// Parse the numeric part
		numPart := text[:suffixStart]
		// Handle different bases
		if strings.HasPrefix(numPart, "0x") || strings.HasPrefix(numPart, "0X") {
			// Hexadecimal
			v, err := strconv.ParseInt(numPart[2:], 16, 64)
			if err != nil {
				return nil, errors.Newf("invalid hex literal: %s", text)
			}
			value = v
		} else if strings.HasPrefix(numPart, "0b") || strings.HasPrefix(numPart, "0B") {
			// Binary
			v, err := strconv.ParseInt(numPart[2:], 2, 64)
			if err != nil {
				return nil, errors.Newf("invalid binary literal: %s", text)
			}
			value = v
		} else {
			// Decimal
			// Remove underscores
			numPart = strings.ReplaceAll(numPart, "_", "")
			v, err := strconv.ParseInt(numPart, 10, 64)
			if err != nil {
				return nil, errors.Newf("invalid decimal literal: %s", text)
			}
			value = v
		}
		// Emit the appropriate constant instruction
		switch baseType {
		case "i8", "i16", "i32", "u8", "u16", "u32":
			e.encoder.WriteI32Const(int32(value))
		case "i64", "u64":
			e.encoder.WriteI64Const(value)
		}
		return types.FromPrimitiveString(baseType), nil
	}

	// Check for float literal
	if floatLit := num.FLOAT_LITERAL(); floatLit != nil {
		text := floatLit.GetText()

		// Parse type suffix if present
		var baseType string
		var value float64

		// Extract suffix
		suffixStart := len(text)
		for _, suffix := range []string{"f32", "f64"} {
			if strings.HasSuffix(text, suffix) {
				baseType = suffix
				suffixStart = len(text) - len(suffix)
				break
			}
		}

		// Default to f64 if no suffix
		if baseType == "" {
			baseType = "f64"
		}

		// Parse the numeric part
		numPart := text[:suffixStart]
		// Remove underscores
		numPart = strings.ReplaceAll(numPart, "_", "")

		v, err := strconv.ParseFloat(numPart, 64)
		if err != nil {
			return nil, errors.Newf("invalid float literal: %s", text)
		}
		value = v

		// Emit the appropriate constant instruction
		if baseType == "f32" {
			e.encoder.WriteF32Const(float32(value))
		} else {
			e.encoder.WriteF64Const(value)
		}

		return types.FromPrimitiveString(baseType), nil
	}

	return nil, errors.New("unknown numeric literal")
}

// compileIdentifier compiles variable references
func (e *Compiler) compileIdentifier(name string) (types.Type, error) {
	// Look up in local variables
	if idx, ok := e.ctx.GetLocal(name); ok {
		e.encoder.WriteLocalGet(idx)

		// Get type from symbol table
		// TODO: Get actual type from symbol table
		return types.F64{}, nil // Placeholder
	}

	// TODO: Check for channels, functions, etc.

	return nil, errors.Newf("undefined identifier: %s", name)
}

// Bytes returns the compiled bytecode
func (e *Compiler) Bytes() []byte {
	return e.encoder.Bytes()
}

// Reset clears the bytecode buffer
func (e *Compiler) Reset() {
	e.encoder.Reset()
}
