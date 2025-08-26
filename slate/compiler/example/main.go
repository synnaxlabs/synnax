package main

import (
	"context"
	"fmt"
	"log"

	"github.com/synnaxlabs/slate/compiler"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func main() {
	// Slate source code
	source := `
	func add(a number, b number) number {
		return a + b
	}
	
	func subtract(a number, b number) number {
		return a - b
	}
	
	func multiply(a number, b number) number {
		return a * b
	}
	
	func divide(a number, b number) number {
		return a / b
	}
	
	func calculate(x number, y number, z number) number {
		sum := x + y
		product := sum * z
		return product
	}`

	// Compile Slate to WASM
	fmt.Println("Compiling Slate source...")
	c := compiler.New()
	module, err := c.Compile(source)
	if err != nil {
		log.Fatalf("Compilation failed: %v", err)
	}

	fmt.Printf("Compiled successfully! WASM size: %d bytes\n", len(module.WASM))
	fmt.Printf("Exported functions:\n")
	for _, fn := range module.Metadata.Functions {
		fmt.Printf("  - %s(", fn.Name)
		for i, param := range fn.Parameters {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s %s", param.Name, param.Type)
		}
		fmt.Printf(") %s\n", fn.ReturnType)
	}

	// Create wazero runtime
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Define host functions (channel operations - not used in this example)
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
		log.Fatalf("Failed to instantiate host module: %v", err)
	}

	// Instantiate the compiled WASM module
	mod, err := r.Instantiate(ctx, module.WASM)
	if err != nil {
		log.Fatalf("Failed to instantiate WASM module: %v", err)
	}

	// Test the add function
	fmt.Println("\nTesting add function:")
	addFunc := mod.ExportedFunction("add")
	if addFunc != nil {
		result, _ := addFunc.Call(ctx, api.EncodeF64(10), api.EncodeF64(32))
		fmt.Printf("  add(10, 32) = %.0f\n", api.DecodeF64(result[0]))
	}

	// Test the calculate function
	fmt.Println("\nTesting calculate function:")
	calcFunc := mod.ExportedFunction("calculate")
	if calcFunc != nil {
		result, _ := calcFunc.Call(ctx, api.EncodeF64(5), api.EncodeF64(3), api.EncodeF64(2))
		fmt.Printf("  calculate(5, 3, 2) = %.0f\n", api.DecodeF64(result[0]))
		fmt.Println("  (which is (5 + 3) * 2 = 16)")
	}

	// Test more functions
	fmt.Println("\nTesting arithmetic functions:")

	subFunc := mod.ExportedFunction("subtract")
	if subFunc != nil {
		result, _ := subFunc.Call(ctx, api.EncodeF64(100), api.EncodeF64(12))
		fmt.Printf("  subtract(100, 40) = %.0f\n", api.DecodeF64(result[0]))
	}

	mulFunc := mod.ExportedFunction("multiply")
	if mulFunc != nil {
		result, _ := mulFunc.Call(ctx, api.EncodeF64(7), api.EncodeF64(6))
		fmt.Printf("  multiply(7, 6) = %.0f\n", api.DecodeF64(result[0]))
	}

	divFunc := mod.ExportedFunction("divide")
	if divFunc != nil {
		result, _ := divFunc.Call(ctx, api.EncodeF64(84), api.EncodeF64(2))
		fmt.Printf("  divide(84, 2) = %.0f\n", api.DecodeF64(result[0]))
	}

	fmt.Println("\nSlate successfully compiled to WASM and executed!")
}
