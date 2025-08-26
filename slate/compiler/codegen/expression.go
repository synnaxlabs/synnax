package codegen

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/synnaxlabs/slate/compiler/wasm"
	generated "github.com/synnaxlabs/slate/parser/generated"
)

// ExpressionCompiler compiles Slate expressions to WASM instructions
type ExpressionCompiler struct {
	// Map variable names to local indices
	locals map[string]uint32
	// Map parameter names to indices  
	params map[string]uint32
	// Buffer for generated code
	Code *bytes.Buffer
}

// NewExpressionCompiler creates a new expression compiler
func NewExpressionCompiler() *ExpressionCompiler {
	return &ExpressionCompiler{
		locals: make(map[string]uint32),
		params: make(map[string]uint32),
		Code:   &bytes.Buffer{},
	}
}

// CompileExpression compiles an expression to WASM instructions
func (ec *ExpressionCompiler) CompileExpression(expr generated.IExpressionContext) error {
	if expr == nil {
		return nil
	}
	
	// Expression -> LogicalOrExpr
	logicalOr := expr.LogicalOrExpr()
	return ec.compileLogicalOr(logicalOr)
}

func (ec *ExpressionCompiler) compileLogicalOr(expr generated.ILogicalOrExprContext) error {
	andExprs := expr.AllLogicalAndExpr()
	
	// Compile first expression
	if err := ec.compileLogicalAnd(andExprs[0]); err != nil {
		return err
	}
	
	// Handle OR operations
	for i := 1; i < len(andExprs); i++ {
		if err := ec.compileLogicalAnd(andExprs[i]); err != nil {
			return err
		}
		// For booleans, we'd need i32 operations
		// For now, focusing on numeric operations
	}
	
	return nil
}

func (ec *ExpressionCompiler) compileLogicalAnd(expr generated.ILogicalAndExprContext) error {
	eqExprs := expr.AllEqualityExpr()
	
	if err := ec.compileEquality(eqExprs[0]); err != nil {
		return err
	}
	
	for i := 1; i < len(eqExprs); i++ {
		if err := ec.compileEquality(eqExprs[i]); err != nil {
			return err
		}
		// Handle AND operations
	}
	
	return nil
}

func (ec *ExpressionCompiler) compileEquality(expr generated.IEqualityExprContext) error {
	relExprs := expr.AllRelationalExpr()
	
	if err := ec.compileRelational(relExprs[0]); err != nil {
		return err
	}
	
	// Handle equality operators
	for i := 1; i < len(relExprs); i++ {
		if err := ec.compileRelational(relExprs[i]); err != nil {
			return err
		}
		
		if expr.EQUAL(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Eq))
		} else if expr.NOT_EQUAL(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Ne))
		}
	}
	
	return nil
}

func (ec *ExpressionCompiler) compileRelational(expr generated.IRelationalExprContext) error {
	addExprs := expr.AllAdditiveExpr()
	
	if err := ec.compileAdditive(addExprs[0]); err != nil {
		return err
	}
	
	// Handle relational operators
	for i := 1; i < len(addExprs); i++ {
		if err := ec.compileAdditive(addExprs[i]); err != nil {
			return err
		}
		
		if expr.LESS_THAN(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Lt))
		} else if expr.LESS_EQUAL(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Le))
		} else if expr.GREATER_THAN(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Gt))
		} else if expr.GREATER_EQUAL(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Ge))
		}
	}
	
	return nil
}

func (ec *ExpressionCompiler) compileAdditive(expr generated.IAdditiveExprContext) error {
	multExprs := expr.AllMultiplicativeExpr()
	
	// Compile first operand
	if err := ec.compileMultiplicative(multExprs[0]); err != nil {
		return err
	}
	
	// Handle addition and subtraction
	for i := 1; i < len(multExprs); i++ {
		// Compile next operand
		if err := ec.compileMultiplicative(multExprs[i]); err != nil {
			return err
		}
		
		// Apply operator
		if expr.PLUS(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Add))
		} else if expr.MINUS(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Sub))
		}
	}
	
	return nil
}

func (ec *ExpressionCompiler) compileMultiplicative(expr generated.IMultiplicativeExprContext) error {
	unaryExprs := expr.AllUnaryExpr()
	
	// Compile first operand
	if err := ec.compileUnary(unaryExprs[0]); err != nil {
		return err
	}
	
	// Handle multiplication and division
	for i := 1; i < len(unaryExprs); i++ {
		// Compile next operand
		if err := ec.compileUnary(unaryExprs[i]); err != nil {
			return err
		}
		
		// Apply operator
		if expr.MULTIPLY(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Mul))
		} else if expr.DIVIDE(i-1) != nil {
			ec.Code.WriteByte(byte(wasm.OpF64Div))
		}
	}
	
	return nil
}

func (ec *ExpressionCompiler) compileUnary(expr generated.IUnaryExprContext) error {
	// Handle unary operators
	if expr.MINUS() != nil {
		// Compile operand
		if err := ec.compileUnary(expr.UnaryExpr()); err != nil {
			return err
		}
		// Negate: 0 - value
		wasm.WriteF64Const(ec.Code, 0.0)
		ec.Code.WriteByte(byte(wasm.OpF64Sub))
		return nil
	}
	
	if expr.NOT() != nil {
		// Handle logical NOT
		if err := ec.compileUnary(expr.UnaryExpr()); err != nil {
			return err
		}
		// For boolean NOT, we'd need i32 operations
		ec.Code.WriteByte(byte(wasm.OpI32Eqz))
		return nil
	}
	
	// No unary operator, compile primary expression
	return ec.compilePrimary(expr.PrimaryExpr())
}

func (ec *ExpressionCompiler) compilePrimary(expr generated.IPrimaryExprContext) error {
	// Handle number literals
	if numLit := expr.NUMBER_LITERAL(); numLit != nil {
		val, err := strconv.ParseFloat(numLit.GetText(), 64)
		if err != nil {
			return fmt.Errorf("invalid number literal: %s", numLit.GetText())
		}
		wasm.WriteF64Const(ec.Code, val)
		return nil
	}
	
	// Handle identifiers (variables/parameters)
	if ident := expr.IDENTIFIER(); ident != nil {
		name := ident.GetText()
		
		// Check if it's a parameter
		if idx, ok := ec.params[name]; ok {
			ec.Code.WriteByte(byte(wasm.OpLocalGet))
			wasm.WriteLEB128(ec.Code, uint64(idx))
			return nil
		}
		
		// Check if it's a local variable
		if idx, ok := ec.locals[name]; ok {
			ec.Code.WriteByte(byte(wasm.OpLocalGet))
			// Local indices come after parameters
			wasm.WriteLEB128(ec.Code, uint64(uint32(len(ec.params))+idx))
			return nil
		}
		
		return fmt.Errorf("undefined variable: %s", name)
	}
	
	// Handle boolean literals
	if expr.TRUE() != nil {
		wasm.WriteI32Const(ec.Code, 1)
		return nil
	}
	
	if expr.FALSE() != nil {
		wasm.WriteI32Const(ec.Code, 0)
		return nil
	}
	
	// Handle parenthesized expressions
	if innerExpr := expr.Expression(); innerExpr != nil {
		return ec.CompileExpression(innerExpr)
	}
	
	// Handle function calls
	if funcCall := expr.FunctionCall(); funcCall != nil {
		return ec.compileFunctionCall(funcCall)
	}
	
	// Handle channel reads
	if channelRead := expr.ChannelRead(); channelRead != nil {
		return ec.compileChannelRead(channelRead)
	}
	
	// Handle string literals (not supported in basic version)
	if expr.STRING() != nil {
		return fmt.Errorf("string literals not yet supported")
	}
	
	return fmt.Errorf("unsupported primary expression")
}

func (ec *ExpressionCompiler) compileFunctionCall(call generated.IFunctionCallContext) error {
	// For now, we don't support function calls in expressions
	// This will be added when we have a function table
	return fmt.Errorf("function calls not yet supported in expressions")
}

func (ec *ExpressionCompiler) compileChannelRead(read generated.IChannelReadContext) error {
	// Channel operations will be compiled as imported function calls
	// For now, not supported
	return fmt.Errorf("channel operations not yet supported")
}

// GetCode returns the generated WASM bytecode
func (ec *ExpressionCompiler) GetCode() []byte {
	return ec.Code.Bytes()
}

// Reset clears the compiler state
func (ec *ExpressionCompiler) Reset() {
	ec.Code.Reset()
	ec.locals = make(map[string]uint32)
	ec.params = make(map[string]uint32)
}

// ResetCode clears just the code buffer, keeping variable mappings
func (ec *ExpressionCompiler) ResetCode() {
	ec.Code.Reset()
}

// SetParameter registers a parameter with its index
func (ec *ExpressionCompiler) SetParameter(name string, idx uint32) {
	ec.params[name] = idx
}

// SetLocal registers a local variable with its index
func (ec *ExpressionCompiler) SetLocal(name string, idx uint32) {
	ec.locals[name] = idx
}

// GetParameterIndex returns the index of a parameter if it exists
func (ec *ExpressionCompiler) GetParameterIndex(name string) (uint32, bool) {
	idx, ok := ec.params[name]
	return idx, ok
}

// GetLocalIndex returns the index of a local if it exists
func (ec *ExpressionCompiler) GetLocalIndex(name string) (uint32, bool) {
	idx, ok := ec.locals[name]
	return idx, ok
}

// GetParameterCount returns the number of parameters
func (ec *ExpressionCompiler) GetParameterCount() int {
	return len(ec.params)
}