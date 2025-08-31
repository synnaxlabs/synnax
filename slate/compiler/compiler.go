// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/synnaxlabs/slate/analyzer/result"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/graph"
	"github.com/synnaxlabs/slate/parser"
)

// Compile generates WASM and metadata from the parsed, analyzed, and assembled program
func Compile(
	program parser.IProgramContext,
	analysisResult *result.Result,
	g *graph.Graph,
) (*Module, error) {
	c := &compiler{
		program:    program,
		scope:      analysisResult.Symbols,
		graph:      g,
		wasmModule: wasm.NewModule(),
		imports:    &wasm.ImportIndex{},
		nodeWASM:   make(map[string]uint32), // Track WASM function indices for nodes
	}

	// Setup imports for all typed host functions
	if err := c.setupImports(); err != nil {
		return nil, err
	}

	// Generate WASM for each node in the graph
	for _, node := range g.Nodes {
		if err := c.generateNodeWASM(&node); err != nil {
			return nil, fmt.Errorf("generating WASM for node %s: %w", node.Key, err)
		}
	}

	// Extract task/function specs from AST for metadata
	tasks := c.extractTaskSpecs(program)
	functions := c.extractFunctionSpecs(program)

	// Encode WASM module
	wasmBytes := c.wasmModule.Generate()

	// Return the compiled module with metadata
	return &Module{
		Version:   "1.0.0",
		Module:    wasmBytes,
		Tasks:     tasks,
		Functions: functions,
		Nodes:     g.Nodes,
		Edges:     g.Edges,
	}, nil
}

// Internal compiler state
type compiler struct {
	program    parser.IProgramContext
	scope      *symbol.Scope
	graph      *graph.Graph
	wasmModule *wasm.Module
	imports    *wasm.ImportIndex
	nodeWASM   map[string]uint32 // Maps node key to WASM function index
}

// setupImports adds all typed host function imports
func (c *compiler) setupImports() error {
	c.imports = wasm.SetupTypedImports(c.wasmModule)
	return nil
}

// generateNodeWASM generates WASM based on node type
func (c *compiler) generateNodeWASM(node *graph.Node) error {
	switch node.Type {
	case "on":
		// Virtual node for channel subscription
		return c.generateOnNode(node)
	case "filter":
		// Virtual node for expression evaluation
		return c.generateFilterNode(node)
	default:
		// Check if it's a stdlib task (no WASM generation needed)
		if c.isStdlibTask(node.Type) {
			// Runtime handles stdlib tasks
			return nil
		}
		// User-defined task
		return c.generateUserTaskNode(node)
	}
}

// generateOnNode generates WASM for a virtual "on" task that reads from a channel
func (c *compiler) generateOnNode(node *graph.Node) error {
	channelName, ok := node.Config["channel"].(string)
	if !ok {
		return fmt.Errorf("on node missing channel config")
	}

	// Create function type: () -> T where T is the channel type
	// For now, assume f64 channels (should look up type from scope)
	funcType := wasm.FunctionType{
		Params:  []wasm.ValueType{}, // No parameters
		Results: []wasm.ValueType{wasm.F64}, // Returns channel value
	}
	typeIdx := c.wasmModule.AddType(funcType)

	// Generate function body
	var body bytes.Buffer

	// Get channel ID (hash of name for now)
	channelID := hashChannelName(channelName)
	
	// i32.const channel_id
	wasm.WriteI32Const(&body, int32(channelID))
	
	// call channel_read_f64
	body.WriteByte(byte(wasm.OpCall))
	wasm.WriteLEB128(&body, uint64(c.imports.ChannelRead["f64"]))

	// Add function to module
	function := wasm.Function{
		Name:    fmt.Sprintf("on_%s", channelName),
		TypeIdx: typeIdx,
		Locals:  []wasm.ValueType{},
		Body:    body.Bytes(),
		Export:  true,
	}

	funcIdx := c.wasmModule.AddFunction(function)
	c.nodeWASM[node.Key] = funcIdx

	return nil
}

// generateFilterNode generates WASM for a virtual filter task that evaluates an expression
func (c *compiler) generateFilterNode(node *graph.Node) error {
	exprText, ok := node.Config["expression"].(string)
	if !ok {
		return fmt.Errorf("filter node missing expression config")
	}

	// Parse the expression string to get an AST
	// This requires parsing just the expression
	exprAST, err := parser.Parse(exprText)
	if err != nil {
		return fmt.Errorf("parsing filter expression: %w", err)
	}

	funcType := wasm.FunctionType{
		Params:  []wasm.ValueType{}, // No parameters (reads channels internally)
		Results: []wasm.ValueType{wasm.I32}, // Returns boolean as i32
	}
	typeIdx := c.wasmModule.AddType(funcType)

	// Generate function body
	var body bytes.Buffer

	// TODO: Properly compile the filter expression
	// This requires:
	// 1. Parsing the expression text into an AST (may need a dedicated expression parser)
	// 2. Walking the expression AST to generate WASM bytecode
	// 3. Handling channel reads within the expression
	// 4. Generating comparison and logical operations
	// 
	// For now, just return true (pass all items through the filter)
	wasm.WriteI32Const(&body, 1)
	
	// Unused variable for now - will be needed when we implement expression compilation
	_ = exprAST

	// Add function to module
	function := wasm.Function{
		Name:    fmt.Sprintf("filter_%s", node.Key),
		TypeIdx: typeIdx,
		Locals:  []wasm.ValueType{},
		Body:    body.Bytes(),
		Export:  true,
	}

	funcIdx := c.wasmModule.AddFunction(function)
	c.nodeWASM[node.Key] = funcIdx

	return nil
}

// generateUserTaskNode generates WASM for a user-defined task
func (c *compiler) generateUserTaskNode(node *graph.Node) error {
	// Find the task declaration in the AST
	var taskDecl parser.ITaskDeclarationContext
	for _, item := range c.program.AllTopLevelItem() {
		if task := item.TaskDeclaration(); task != nil {
			if task.IDENTIFIER().GetText() == node.Type {
				taskDecl = task
				break
			}
		}
	}

	if taskDecl == nil {
		return fmt.Errorf("task declaration not found for %s", node.Type)
	}

	// Create function type for the task
	params := []wasm.ValueType{wasm.I32} // task instance ID

	// Add runtime argument types
	if paramList := taskDecl.ParameterList(); paramList != nil {
		for _, param := range paramList.AllParameter() {
			paramType := extractType(param.Type_())
			params = append(params, slateTypeToWASM(paramType))
		}
	}

	// Add return type if present
	var results []wasm.ValueType
	if retType := taskDecl.ReturnType(); retType != nil {
		returnType := extractType(retType.Type_())
		results = append(results, slateTypeToWASM(returnType))
	}

	funcType := wasm.FunctionType{
		Params:  params,
		Results: results,
	}
	typeIdx := c.wasmModule.AddType(funcType)

	// Generate function body
	var body bytes.Buffer
	
	// TODO: Properly compile the task body
	// This requires:
	// 1. Walking the task's block statements
	// 2. Tracking local variables and their types
	// 3. Compiling expressions to WASM bytecode
	// 4. Handling stateful variables (load at start, store at end)
	// 5. Generating proper control flow (if/else, loops)
	
	// For now, generate a stub that returns a default value
	if len(results) > 0 {
		switch results[0] {
		case wasm.I32:
			wasm.WriteI32Const(&body, 0)
		case wasm.I64:
			body.WriteByte(byte(wasm.OpI64Const))
			wasm.WriteLEB128(&body, 0)
		case wasm.F32:
			wasm.WriteF32Const(&body, 0.0)
		case wasm.F64:
			wasm.WriteF64Const(&body, 0.0)
		}
	}
	
	// TODO: Calculate actual local variables needed
	// For now, use empty locals
	var localTypes []wasm.ValueType

	// Add function to module
	function := wasm.Function{
		Name:    fmt.Sprintf("task_%s", node.Key),
		TypeIdx: typeIdx,
		Locals:  localTypes,
		Body:    body.Bytes(),
		Export:  true,
	}

	funcIdx := c.wasmModule.AddFunction(function)
	c.nodeWASM[node.Key] = funcIdx

	return nil
}

// isStdlibTask checks if a task type is a standard library task
func (c *compiler) isStdlibTask(taskType string) bool {
	stdlibTasks := []string{
		"interval", "all", "once", "any", 
		"throttle", "merge", "tee",
	}
	for _, stdlib := range stdlibTasks {
		if taskType == stdlib {
			return true
		}
	}
	return false
}

// extractTaskSpecs extracts task specifications from the AST
func (c *compiler) extractTaskSpecs(program parser.IProgramContext) []TaskSpec {
	var tasks []TaskSpec

	for _, item := range program.AllTopLevelItem() {
		if taskDecl := item.TaskDeclaration(); taskDecl != nil {
			spec := TaskSpec{
				Name: taskDecl.IDENTIFIER().GetText(),
				Key:  fmt.Sprintf("task_%s", taskDecl.IDENTIFIER().GetText()),
			}

			// Extract config parameters
			if configBlock := taskDecl.ConfigBlock(); configBlock != nil {
				for _, configParam := range configBlock.AllConfigParameter() {
					spec.Config = append(spec.Config, ParamSpec{
						Name: configParam.IDENTIFIER().GetText(),
						Type: extractType(configParam.Type_()),
					})
				}
			}

			// Extract runtime parameters
			if paramList := taskDecl.ParameterList(); paramList != nil {
				for _, param := range paramList.AllParameter() {
					spec.Args = append(spec.Args, ParamSpec{
						Name: param.IDENTIFIER().GetText(),
						Type: extractType(param.Type_()),
					})
				}
			}

			// Extract return type
			if retType := taskDecl.ReturnType(); retType != nil {
				returnType := extractType(retType.Type_())
				spec.Return = &returnType
			}

			// TODO: Extract stateful variables from task body
			// This requires walking the block to find $= declarations
			// For now, return empty list
			spec.StatefulVars = []StateVar{}

			tasks = append(tasks, spec)
		}
	}

	return tasks
}

// extractFunctionSpecs extracts function specifications from the AST
func (c *compiler) extractFunctionSpecs(program parser.IProgramContext) []FuncSpec {
	var functions []FuncSpec

	for _, item := range program.AllTopLevelItem() {
		if funcDecl := item.FunctionDeclaration(); funcDecl != nil {
			spec := FuncSpec{
				Name: funcDecl.IDENTIFIER().GetText(),
				Key:  fmt.Sprintf("func_%s", funcDecl.IDENTIFIER().GetText()),
			}

			// Extract parameters
			if paramList := funcDecl.ParameterList(); paramList != nil {
				for _, param := range paramList.AllParameter() {
					spec.Params = append(spec.Params, ParamSpec{
						Name: param.IDENTIFIER().GetText(),
						Type: extractType(param.Type_()),
					})
				}
			}

			// Extract return type
			if retType := funcDecl.ReturnType(); retType != nil {
				returnType := extractType(retType.Type_())
				spec.Return = &returnType
			}

			functions = append(functions, spec)
		}
	}

	return functions
}

// extractType extracts the type string from a type context
func extractType(typeCtx parser.ITypeContext) string {
	if typeCtx == nil {
		return ""
	}

	// Handle primitive types
	if prim := typeCtx.PrimitiveType(); prim != nil {
		if num := prim.NumericType(); num != nil {
			if intType := num.IntegerType(); intType != nil {
				if intType.I8() != nil {
					return "i8"
				} else if intType.I16() != nil {
					return "i16"
				} else if intType.I32() != nil {
					return "i32"
				} else if intType.I64() != nil {
					return "i64"
				} else if intType.U8() != nil {
					return "u8"
				} else if intType.U16() != nil {
					return "u16"
				} else if intType.U32() != nil {
					return "u32"
				} else if intType.U64() != nil {
					return "u64"
				}
			} else if floatType := num.FloatType(); floatType != nil {
				if floatType.F32() != nil {
					return "f32"
				} else if floatType.F64() != nil {
					return "f64"
				}
			} else if temp := num.TemporalType(); temp != nil {
				if temp.TIMESTAMP() != nil {
					return "timestamp"
				} else if temp.TIMESPAN() != nil {
					return "timespan"
				}
			}
		} else if prim.STRING() != nil {
			return "string"
		}
	}

	// Handle channel types
	if chanType := typeCtx.ChannelType(); chanType != nil {
		prefix := "chan"
		if chanType.RECV_CHAN() != nil {
			prefix = "<-chan"
		} else if chanType.SEND_CHAN() != nil {
			prefix = "->chan"
		}
		
		innerType := ""
		if chanType.PrimitiveType() != nil {
			// Recursively extract primitive type
			innerType = extractPrimitiveType(chanType.PrimitiveType())
		} else if chanType.SeriesType() != nil {
			innerType = extractSeriesType(chanType.SeriesType())
		}
		
		return fmt.Sprintf("%s %s", prefix, innerType)
	}

	// Handle series types
	if seriesType := typeCtx.SeriesType(); seriesType != nil {
		return extractSeriesType(seriesType)
	}

	return ""
}

// extractPrimitiveType is a helper for extracting primitive types
func extractPrimitiveType(prim parser.IPrimitiveTypeContext) string {
	if num := prim.NumericType(); num != nil {
		if intType := num.IntegerType(); intType != nil {
			if intType.I8() != nil {
				return "i8"
			} else if intType.I16() != nil {
				return "i16"
			} else if intType.I32() != nil {
				return "i32"
			} else if intType.I64() != nil {
				return "i64"
			} else if intType.U8() != nil {
				return "u8"
			} else if intType.U16() != nil {
				return "u16"
			} else if intType.U32() != nil {
				return "u32"
			} else if intType.U64() != nil {
				return "u64"
			}
		} else if floatType := num.FloatType(); floatType != nil {
			if floatType.F32() != nil {
				return "f32"
			} else if floatType.F64() != nil {
				return "f64"
			}
		} else if temp := num.TemporalType(); temp != nil {
			if temp.TIMESTAMP() != nil {
				return "timestamp"
			} else if temp.TIMESPAN() != nil {
				return "timespan"
			}
		}
	} else if prim.STRING() != nil {
		return "string"
	}
	return ""
}

// extractSeriesType extracts series type string
func extractSeriesType(series parser.ISeriesTypeContext) string {
	if series == nil {
		return ""
	}
	innerType := extractPrimitiveType(series.PrimitiveType())
	return fmt.Sprintf("series %s", innerType)
}

// hashChannelName creates a simple hash for channel names
func hashChannelName(name string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(name); i++ {
		hash ^= uint32(name[i])
		hash *= 16777619
	}
	return hash
}

// slateTypeToWASM converts a Slate type to a WASM value type
func slateTypeToWASM(slateType string) wasm.ValueType {
	switch slateType {
	case "i8", "i16", "i32", "u8", "u16", "u32", "bool":
		return wasm.I32
	case "i64", "u64", "timestamp", "timespan":
		return wasm.I64
	case "f32":
		return wasm.F32
	case "f64":
		return wasm.F64
	case "string":
		return wasm.I32 // String handle
	default:
		// For series and channels, return handle (i32)
		if strings.HasPrefix(slateType, "series") || 
		   strings.Contains(slateType, "chan") {
			return wasm.I32
		}
		return wasm.I32 // Default to i32 for unknown types
	}
}