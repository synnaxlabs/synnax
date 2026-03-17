// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package llm

import (
	"context"
	"fmt"
)

// Generator is the interface for LLM text generation. Implementations handle
// provider-specific API details (auth, request format, etc.) behind this
// single method.
type Generator interface {
	Generate(ctx context.Context, system string, messages []Message) (string, error)
}

// Provider identifies an LLM provider.
type Provider string

const (
	ProviderOpenAI Provider = "openai"
)

// NewGenerator creates a Generator for the configured provider. If no provider
// is specified, it defaults to OpenAI.
func NewGenerator(cfg Config) (Generator, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderOpenAI
	}
	switch provider {
	case ProviderOpenAI:
		return newOpenAIClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}
