package compiler

import (
	"bytes"
	"fmt"

	"github.com/synnaxlabs/slate/compiler/codegen"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	generated "github.com/synnaxlabs/slate/parser/generated"
)

// Metadata contains information about the compiled module for the runtime
type Metadata struct {
	Functions []FunctionMeta
	Bindings  []BindingMeta
}

// FunctionMeta describes an exported function
type FunctionMeta struct {
	Name       string
	Parameters []ParameterMeta
	ReturnType string
}

// ParameterMeta describes a function parameter
type ParameterMeta struct {
	Name string
	Type string // "number", "bool", "<-chan", "->chan", "chan"
}

// BindingMeta describes a reactive binding
type BindingMeta struct {
	Type         string // "channel", "channel_list", "interval"
	Trigger      string // channel name or interval duration
	FunctionName string // exported function name
}

// CompiledModule contains the WASM binary and metadata
type CompiledModule struct {
	WASM     []byte
	Metadata *Metadata
}

// Compiler compiles Slate source to WASM
type Compiler struct {
	module   *wasm.Module
	metadata *Metadata
	exprComp *codegen.ExpressionCompiler
	
	// Track imported functions for channel operations
	channelReadIdx  uint32
	channelWriteIdx uint32
}

// New creates a new compiler
func New() *Compiler {
	return &Compiler{
		module: wasm.NewModule(),
		metadata: &Metadata{
			Functions: make([]FunctionMeta, 0),
			Bindings:  make([]BindingMeta, 0),
		},
		exprComp: codegen.NewExpressionCompiler(),
	}
}

// Compile compiles Slate source code to WASM
func (c *Compiler) Compile(source string) (*CompiledModule, error) {
	// Parse the source
	tree, err := parser.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	
	// Setup standard imports
	c.setupImports()
	
	// Compile each top-level statement
	for _, stmt := range tree.AllTopLevelStatement() {
		if funcDecl := stmt.FunctionDecl(); funcDecl != nil {
			if err := c.compileFunction(funcDecl); err != nil {
				return nil, err
			}
		} else if binding := stmt.ReactiveBinding(); binding != nil {
			if err := c.compileBinding(binding); err != nil {
				return nil, err
			}
		}
	}
	
	// Generate WASM binary
	wasmBytes := c.module.Generate()
	
	return &CompiledModule{
		WASM:     wasmBytes,
		Metadata: c.metadata,
	}, nil
}

func (c *Compiler) setupImports() {
	// Import channel read function
	c.channelReadIdx = c.module.AddImport("env", "channel_read", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32}, // channel ID
		Results: []wasm.ValueType{wasm.F64},  // value
	})
	
	// Import channel write function
	c.channelWriteIdx = c.module.AddImport("env", "channel_write", wasm.FunctionType{
		Params:  []wasm.ValueType{wasm.I32, wasm.F64}, // channel ID, value
		Results: []wasm.ValueType{},
	})
}

func (c *Compiler) compileFunction(funcDecl generated.IFunctionDeclContext) error {
	name := funcDecl.IDENTIFIER().GetText()
	
	// Build function metadata
	funcMeta := FunctionMeta{
		Name:       name,
		Parameters: make([]ParameterMeta, 0),
	}
	
	// Build function type
	funcType := wasm.FunctionType{
		Params:  make([]wasm.ValueType, 0),
		Results: make([]wasm.ValueType, 0),
	}
	
	// Reset expression compiler for this function
	c.exprComp.Reset()
	
	// Process parameters
	if paramList := funcDecl.ParameterList(); paramList != nil {
		for i, param := range paramList.AllParameter() {
			paramName := param.IDENTIFIER().GetText()
			paramType := c.getParameterType(param.Type_())
			
			funcMeta.Parameters = append(funcMeta.Parameters, ParameterMeta{
				Name: paramName,
				Type: paramType,
			})
			
			// Add to function signature
			if paramType == "<-chan" || paramType == "->chan" || paramType == "chan" {
				// Channel parameters are passed as i32 (channel ID)
				funcType.Params = append(funcType.Params, wasm.I32)
			} else if paramType == "bool" {
				funcType.Params = append(funcType.Params, wasm.I32)
			} else { // number
				funcType.Params = append(funcType.Params, wasm.F64)
			}
			
			// Register parameter in expression compiler
			c.exprComp.SetParameter(paramName, uint32(i))
		}
	}
	
	// Process return type
	if returnType := funcDecl.ReturnType(); returnType != nil {
		retType := c.getReturnType(returnType.Type_())
		funcMeta.ReturnType = retType
		
		if retType == "number" {
			funcType.Results = append(funcType.Results, wasm.F64)
		} else if retType == "bool" {
			funcType.Results = append(funcType.Results, wasm.I32)
		}
		// void has no results
	}
	
	// Add function type to module
	typeIdx := c.module.AddType(funcType)
	
	// Compile function body
	bodyCode, locals, err := c.compileFunctionBody(funcDecl.Block())
	if err != nil {
		return fmt.Errorf("error compiling function %s: %w", name, err)
	}
	
	// Create WASM function
	wasmFunc := wasm.Function{
		Name:    name,
		TypeIdx: typeIdx,
		Locals:  locals,
		Body:    bodyCode,
		Export:  true, // Export all functions for runtime to call
	}
	
	// Add function to module
	c.module.AddFunction(wasmFunc)
	
	// Add to metadata
	c.metadata.Functions = append(c.metadata.Functions, funcMeta)
	
	return nil
}

func (c *Compiler) compileFunctionBody(block generated.IBlockContext) ([]byte, []wasm.ValueType, error) {
	var code bytes.Buffer
	locals := make([]wasm.ValueType, 0)
	localCount := uint32(0)
	
	// Compile each statement
	for _, stmt := range block.AllStatement() {
		// Clear the expression compiler code buffer for each statement
		c.exprComp.ResetCode()
		
		if returnStmt := stmt.ReturnStatement(); returnStmt != nil {
			// Compile return expression
			if expr := returnStmt.Expression(); expr != nil {
				if err := c.exprComp.CompileExpression(expr); err != nil {
					return nil, nil, err
				}
				code.Write(c.exprComp.GetCode())
			}
			// Add return instruction
			code.WriteByte(byte(wasm.OpReturn))
			
		} else if varDecl := stmt.VariableDecl(); varDecl != nil {
			// Handle variable declaration
			varName := varDecl.IDENTIFIER().GetText()
			
			// Compile initialization expression
			if err := c.exprComp.CompileExpression(varDecl.Expression()); err != nil {
				return nil, nil, err
			}
			code.Write(c.exprComp.GetCode())
			
			// Store in local
			code.WriteByte(byte(wasm.OpLocalSet))
			paramCount := uint32(c.exprComp.GetParameterCount())
			wasm.WriteLEB128(&code, uint64(paramCount+localCount))
			
			// Register the local for future use
			c.exprComp.SetLocal(varName, localCount)
			
			// Add local variable
			locals = append(locals, wasm.F64) // Assume number for now
			localCount++
			
		} else if assignment := stmt.Assignment(); assignment != nil {
			// Handle assignment
			varName := assignment.IDENTIFIER().GetText()
			
			// Compile expression
			if err := c.exprComp.CompileExpression(assignment.Expression()); err != nil {
				return nil, nil, err
			}
			code.Write(c.exprComp.GetCode())
			
			// Store in variable
			if idx, ok := c.exprComp.GetParameterIndex(varName); ok {
				code.WriteByte(byte(wasm.OpLocalSet))
				wasm.WriteLEB128(&code, uint64(idx))
			} else if idx, ok := c.exprComp.GetLocalIndex(varName); ok {
				code.WriteByte(byte(wasm.OpLocalSet))
				paramCount := uint32(c.exprComp.GetParameterCount())
				wasm.WriteLEB128(&code, uint64(paramCount+idx))
			} else {
				return nil, nil, fmt.Errorf("undefined variable: %s", varName)
			}
			
		} else if channelWrite := stmt.ChannelWrite(); channelWrite != nil {
			// Handle channel write: value -> channel
			// Compile the value expression
			if err := c.exprComp.CompileExpression(channelWrite.Expression()); err != nil {
				return nil, nil, err
			}
			code.Write(c.exprComp.GetCode())
			
			// Get channel ID (assuming it's a parameter)
			channelName := channelWrite.IDENTIFIER().GetText()
			if idx, ok := c.exprComp.GetParameterIndex(channelName); ok {
				// Load channel ID
				code.WriteByte(byte(wasm.OpLocalGet))
				wasm.WriteLEB128(&code, uint64(idx))
				// Swap so we have: channel_id, value
				// (WASM doesn't have swap, so we need to use locals)
				// For now, assuming the value is on top and channel param is loaded
			}
			
			// Call channel_write
			code.WriteByte(byte(wasm.OpCall))
			wasm.WriteLEB128(&code, uint64(c.channelWriteIdx))
			
		} else if ifStmt := stmt.IfStatement(); ifStmt != nil {
			// Handle if statement (simplified for now)
			return nil, nil, fmt.Errorf("if statements not yet implemented")
			
		} else if exprStmt := stmt.ExpressionStatement(); exprStmt != nil {
			// Handle expression statement
			if err := c.exprComp.CompileExpression(exprStmt.Expression()); err != nil {
				return nil, nil, err
			}
			code.Write(c.exprComp.GetCode())
			// Drop the result if any
			// For now, assuming expressions used as statements don't produce values
		}
	}
	
	return code.Bytes(), locals, nil
}

func (c *Compiler) compileBinding(binding generated.IReactiveBindingContext) error {
	meta := BindingMeta{}
	
	// Determine binding type
	if intervalBinding := binding.IntervalBinding(); intervalBinding != nil {
		meta.Type = "interval"
		meta.Trigger = intervalBinding.NUMBER_LITERAL().GetText()
		meta.FunctionName = intervalBinding.FunctionCall().IDENTIFIER().GetText()
		
	} else if channelList := binding.ChannelList(); channelList != nil {
		meta.Type = "channel_list"
		// Collect channel names
		channels := ""
		for i, ident := range channelList.AllIDENTIFIER() {
			if i > 0 {
				channels += ","
			}
			channels += ident.GetText()
		}
		meta.Trigger = channels
		meta.FunctionName = binding.FunctionCall().IDENTIFIER().GetText()
		
	} else if ident := binding.IDENTIFIER(); ident != nil {
		meta.Type = "channel"
		meta.Trigger = ident.GetText()
		meta.FunctionName = binding.FunctionCall().IDENTIFIER().GetText()
	}
	
	// Add to metadata (runtime will handle the actual binding)
	c.metadata.Bindings = append(c.metadata.Bindings, meta)
	
	return nil
}

func (c *Compiler) getParameterType(typeNode generated.ITypeContext) string {
	if typeNode.NUMBER() != nil {
		return "number"
	} else if typeNode.BOOL() != nil {
		return "bool"
	} else if typeNode.VOID() != nil {
		return "void"
	} else if typeNode.CHANNEL_RECV() != nil {
		return "<-chan"
	} else if typeNode.CHANNEL_SEND() != nil {
		return "->chan"
	} else if typeNode.CHAN() != nil {
		return "chan"
	}
	return "unknown"
}

func (c *Compiler) getReturnType(typeNode generated.ITypeContext) string {
	return c.getParameterType(typeNode)
}