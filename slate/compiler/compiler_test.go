package compiler_test

import (
	"context"
	"testing"

	"github.com/synnaxlabs/slate/compiler"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func TestCompileAdd(t *testing.T) {
	source := `func add(a number, b number) number {
		return a + b
	}`
	
	// Compile the Slate source
	c := compiler.New()
	module, err := c.Compile(source)
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	
	// Create wazero runtime
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	
	// Define host functions (channel operations)
	_, err = r.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(channelID int32) float64 {
			// Mock channel read
			return 0.0
		}).
		Export("channel_read").
		NewFunctionBuilder().
		WithFunc(func(channelID int32, value float64) {
			// Mock channel write
		}).
		Export("channel_write").
		Instantiate(ctx)
	if err != nil {
		t.Fatalf("Failed to instantiate host module: %v", err)
	}
	
	// Instantiate the compiled module
	mod, err := r.Instantiate(ctx, module.WASM)
	if err != nil {
		t.Fatalf("Failed to instantiate WASM module: %v", err)
	}
	
	// Get the exported add function
	addFunc := mod.ExportedFunction("add")
	if addFunc == nil {
		t.Fatal("add function not exported")
	}
	
	// Test the add function
	testCases := []struct {
		a, b, expected float64
	}{
		{1.0, 2.0, 3.0},
		{-5.0, 3.0, -2.0},
		{10.5, 20.5, 31.0},
		{0.0, 0.0, 0.0},
	}
	
	for _, tc := range testCases {
		results, err := addFunc.Call(ctx, api.EncodeF64(tc.a), api.EncodeF64(tc.b))
		if err != nil {
			t.Errorf("Failed to call add(%f, %f): %v", tc.a, tc.b, err)
			continue
		}
		
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
			continue
		}
		
		result := api.DecodeF64(results[0])
		if result != tc.expected {
			t.Errorf("add(%f, %f) = %f, expected %f", tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestCompileMultipleOperations(t *testing.T) {
	source := `
	func multiply(x number, y number) number {
		return x * y
	}
	
	func subtract(x number, y number) number {
		return x - y
	}
	
	func divide(x number, y number) number {
		return x / y
	}
	`
	
	// Compile the Slate source
	c := compiler.New()
	module, err := c.Compile(source)
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	
	// Verify metadata
	if len(module.Metadata.Functions) != 3 {
		t.Errorf("Expected 3 functions in metadata, got %d", len(module.Metadata.Functions))
	}
	
	// Create wazero runtime
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	
	// Define host functions
	_, err = r.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(channelID int32) float64 {
			return 0.0
		}).
		Export("channel_read").
		NewFunctionBuilder().
		WithFunc(func(channelID int32, value float64) {}).
		Export("channel_write").
		Instantiate(ctx)
	if err != nil {
		t.Fatalf("Failed to instantiate host module: %v", err)
	}
	
	// Instantiate the compiled module
	mod, err := r.Instantiate(ctx, module.WASM)
	if err != nil {
		t.Fatalf("Failed to instantiate WASM module: %v", err)
	}
	
	// Test multiply
	multiplyFunc := mod.ExportedFunction("multiply")
	if multiplyFunc != nil {
		results, err := multiplyFunc.Call(ctx, api.EncodeF64(3.0), api.EncodeF64(4.0))
		if err != nil {
			t.Errorf("Failed to call multiply: %v", err)
		} else {
			result := api.DecodeF64(results[0])
			if result != 12.0 {
				t.Errorf("multiply(3, 4) = %f, expected 12", result)
			}
		}
	}
	
	// Test subtract
	subtractFunc := mod.ExportedFunction("subtract")
	if subtractFunc != nil {
		results, err := subtractFunc.Call(ctx, api.EncodeF64(10.0), api.EncodeF64(3.0))
		if err != nil {
			t.Errorf("Failed to call subtract: %v", err)
		} else {
			result := api.DecodeF64(results[0])
			if result != 7.0 {
				t.Errorf("subtract(10, 3) = %f, expected 7", result)
			}
		}
	}
	
	// Test divide
	divideFunc := mod.ExportedFunction("divide")
	if divideFunc != nil {
		results, err := divideFunc.Call(ctx, api.EncodeF64(15.0), api.EncodeF64(3.0))
		if err != nil {
			t.Errorf("Failed to call divide: %v", err)
		} else {
			result := api.DecodeF64(results[0])
			if result != 5.0 {
				t.Errorf("divide(15, 3) = %f, expected 5", result)
			}
		}
	}
}

func TestCompileComplexExpression(t *testing.T) {
	source := `func calculate(a number, b number, c number) number {
		return (a + b) * c
	}`
	
	// Compile the Slate source
	c := compiler.New()
	module, err := c.Compile(source)
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	
	// Create wazero runtime
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	
	// Define host functions
	_, err = r.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(channelID int32) float64 {
			return 0.0
		}).
		Export("channel_read").
		NewFunctionBuilder().
		WithFunc(func(channelID int32, value float64) {}).
		Export("channel_write").
		Instantiate(ctx)
	if err != nil {
		t.Fatalf("Failed to instantiate host module: %v", err)
	}
	
	// Instantiate the compiled module
	mod, err := r.Instantiate(ctx, module.WASM)
	if err != nil {
		t.Fatalf("Failed to instantiate WASM module: %v", err)
	}
	
	// Get the exported calculate function
	calcFunc := mod.ExportedFunction("calculate")
	if calcFunc == nil {
		t.Fatal("calculate function not exported")
	}
	
	// Test: (2 + 3) * 4 = 20
	results, err := calcFunc.Call(ctx, api.EncodeF64(2.0), api.EncodeF64(3.0), api.EncodeF64(4.0))
	if err != nil {
		t.Fatalf("Failed to call calculate: %v", err)
	}
	
	result := api.DecodeF64(results[0])
	if result != 20.0 {
		t.Errorf("calculate(2, 3, 4) = %f, expected 20", result)
	}
}