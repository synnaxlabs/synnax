package llm_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/service/agent/llm"
)

func TestGenerateArcCode(t *testing.T) {
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
		{
			Name:     "pump_3_current",
			Key:      1001,
			DataType: "float64",
			Mean:     1.95,
			StdDev:   0.08,
			Min:      1.78,
			Max:      2.15,
			Count:    360000,
		},
		{
			Name:     "pump_3_status",
			Key:      1002,
			DataType: "uint8",
			Mean:     0.85,
			StdDev:   0.36,
			Min:      0,
			Max:      1,
			Count:    3600,
		},
	}

	messages := llm.BuildMessages(
		[]llm.AgentMessage{{
			Role:    "user",
			Content: "Monitor the current on pump 3 and tell me when it's out of family",
		}},
		channels,
	)

	fmt.Println("=== USER MESSAGE ===")
	fmt.Println(messages[0].Content)
	fmt.Println()

	response, err := generator.Generate(context.Background(), llm.SystemPrompt, messages)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("=== RAW RESPONSE ===")
	fmt.Println(response)
	fmt.Println()

	code, err := llm.ExtractArcCode(response)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("=== EXTRACTED ARC CODE ===")
	fmt.Println(code)
}
