package llm_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/agent/llm"
)

func TestGenerateAndCompile(t *testing.T) {
	apiKey := os.Getenv("SYNNAX_LLM_API_KEY")
	if apiKey == "" {
		t.Skip("SYNNAX_LLM_API_KEY not set")
	}

	generator, err := llm.NewGenerator(llm.Config{
		APIKey:  apiKey,
		Model:   "gpt-4o",
		BaseURL: "https://api.openai.com/v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	channels := []llm.ChannelInfo{
		{Name: "pump_3_current", Key: 1001, DataType: "float64", Mean: 1.95, StdDev: 0.08, Min: 1.78, Max: 2.15, Count: 360000},
		{Name: "pump_3_status", Key: 1002, DataType: "uint8", Mean: 0.85, StdDev: 0.36, Min: 0, Max: 1, Count: 3600},
	}

	messages := llm.BuildMessages(
		[]llm.AgentMessage{{
			Role:    "user",
			Content: "Monitor the current on pump 3 and tell me when it's out of family",
		}},
		channels,
	)

	response, err := generator.Generate(context.Background(), llm.SystemPrompt, messages)
	if err != nil {
		t.Fatal(err)
	}

	code, err := llm.ExtractArcCode(response)
	if err != nil {
		t.Fatalf("No code block in response:\n%s", response)
	}
	fmt.Println("=== GENERATED ARC CODE ===")
	fmt.Println(code)
	fmt.Println()

	resolver := symbol.CompoundResolver{}
	resolver = append(resolver, stl.SymbolResolver...)
	resolver = append(resolver, arcstatus.SymbolResolver)
	resolver = append(resolver, symbol.MapResolver{
		"pump_3_current": arc.Symbol{
			Name: "pump_3_current",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   1001,
		},
		"pump_3_status": arc.Symbol{
			Name: "pump_3_status",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.U8()),
			ID:   1002,
		},
	})

	txt := arc.Text{Raw: code}
	program, err := arc.CompileText(context.Background(), txt, arc.WithResolver(resolver))
	if err != nil {
		t.Fatalf("Compilation failed:\n%v\n\nGenerated code:\n%s", err, code)
	}

	fmt.Printf("=== COMPILATION SUCCESS ===\n")
	fmt.Printf("Nodes: %d\n", len(program.IR.Nodes))
	fmt.Printf("Edges: %d\n", len(program.IR.Edges))
	for _, n := range program.IR.Nodes {
		fmt.Printf("  Node: key=%s type=%s\n", n.Key, n.Type)
	}
}
