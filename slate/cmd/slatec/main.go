package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/slate/compiler"
)

const version = "0.1.0"

func main() {
	var (
		outputPath   = flag.String("o", "", "Output WASM file (default: <input>.wasm)")
		metadataPath = flag.String("meta", "", "Output metadata JSON file (optional)")
		showVersion  = flag.Bool("version", false, "Show version")
		verbose      = flag.Bool("v", false, "Verbose output")
		stdout       = flag.Bool("stdout", false, "Output WASM to stdout")
		showHelp     = flag.Bool("h", false, "Show help")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Slate Compiler v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: slatec [options] <input.slate>\n\n")
		fmt.Fprintf(os.Stderr, "Compiles Slate source files to WebAssembly modules.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  slatec program.slate                  # Creates program.wasm\n")
		fmt.Fprintf(os.Stderr, "  slatec -o output.wasm program.slate   # Specify output file\n")
		fmt.Fprintf(os.Stderr, "  slatec -meta meta.json program.slate  # Also output metadata\n")
		fmt.Fprintf(os.Stderr, "  slatec -stdout program.slate > out.wasm # Output to stdout\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("Slate Compiler v%s\n", version)
		return
	}

	if *showHelp || flag.NArg() == 0 {
		flag.Usage()
		return
	}

	// Get input file
	inputPath := flag.Arg(0)

	// Read input file
	source, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Compiling %s...\n", inputPath)
	}

	// Compile
	c := compiler.New()
	module, err := c.Compile(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	// Determine output path
	var output io.Writer
	if *stdout {
		output = os.Stdout
	} else {
		outPath := *outputPath
		if outPath == "" {
			// Default: replace .slate with .wasm
			base := strings.TrimSuffix(inputPath, filepath.Ext(inputPath))
			outPath = base + ".wasm"
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", outPath, err)
			os.Exit(1)
		}
		defer outFile.Close()
		output = outFile

		if *verbose {
			fmt.Fprintf(os.Stderr, "Writing WASM to %s (%d bytes)\n", outPath, len(module.WASM))
		}
	}

	// Write WASM
	if _, err := output.Write(module.WASM); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing WASM: %v\n", err)
		os.Exit(1)
	}

	// Write metadata if requested
	if *metadataPath != "" {
		metaData, err := json.MarshalIndent(module.Metadata, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling metadata: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*metadataPath, metaData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing metadata file %s: %v\n", *metadataPath, err)
			os.Exit(1)
		}

		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote metadata to %s\n", *metadataPath)
		}
	}

	if *verbose && !*stdout {
		fmt.Fprintf(os.Stderr, "Compilation successful!\n")
		fmt.Fprintf(os.Stderr, "  Functions: %d\n", len(module.Metadata.Functions))
		fmt.Fprintf(os.Stderr, "  Bindings: %d\n", len(module.Metadata.Bindings))
		
		if len(module.Metadata.Functions) > 0 {
			fmt.Fprintf(os.Stderr, "  Exported functions:\n")
			for _, fn := range module.Metadata.Functions {
				fmt.Fprintf(os.Stderr, "    - %s(", fn.Name)
				for i, param := range fn.Parameters {
					if i > 0 {
						fmt.Fprintf(os.Stderr, ", ")
					}
					fmt.Fprintf(os.Stderr, "%s %s", param.Name, param.Type)
				}
				fmt.Fprintf(os.Stderr, ") %s\n", fn.ReturnType)
			}
		}

		if len(module.Metadata.Bindings) > 0 {
			fmt.Fprintf(os.Stderr, "  Reactive bindings:\n")
			for _, binding := range module.Metadata.Bindings {
				fmt.Fprintf(os.Stderr, "    - %s: %s -> %s()\n", 
					binding.Type, binding.Trigger, binding.FunctionName)
			}
		}
	}
}