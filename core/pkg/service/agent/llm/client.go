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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

const (
	defaultBaseURL   = "https://api.openai.com/v1"
	defaultModel     = "gpt-4o"
	defaultMaxTokens = 4096
	defaultTimeout   = 60 * time.Second
)

type Config struct {
	Provider Provider
	APIKey   string
	Model    string
	BaseURL  string
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{
		Model:   defaultModel,
		BaseURL: defaultBaseURL,
	}
)

func (c Config) Override(other Config) Config {
	c.Provider = override.String(c.Provider, other.Provider)
	c.APIKey = override.String(c.APIKey, other.APIKey)
	c.Model = override.String(c.Model, other.Model)
	c.BaseURL = override.String(c.BaseURL, other.BaseURL)
	return c
}

func (c Config) Validate() error {
	v := validate.New("llm")
	validate.NotEmptyString(v, "api_key", c.APIKey)
	validate.NotEmptyString(v, "model", c.Model)
	validate.NotEmptyString(v, "base_url", c.BaseURL)
	return v.Error()
}

type openAIClient struct {
	cfg  Config
	http *http.Client
}

var _ Generator = (*openAIClient)(nil)

func newOpenAIClient(cfg Config) (*openAIClient, error) {
	resolved, err := config.New(DefaultConfig, cfg)
	if err != nil {
		return nil, err
	}
	return &openAIClient{
		cfg:  resolved,
		http: &http.Client{Timeout: defaultTimeout},
	}, nil
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_completion_tokens"`
	Messages  []Message `json:"messages"`
}

type choice struct {
	Message Message `json:"message"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type response struct {
	Choices []choice  `json:"choices"`
	Error   *apiError `json:"error,omitempty"`
}

func (c *openAIClient) Generate(ctx context.Context, system string, messages []Message) (string, error) {
	allMessages := make([]Message, 0, len(messages)+1)
	allMessages = append(allMessages, Message{Role: "system", Content: system})
	allMessages = append(allMessages, messages...)
	body := request{
		Model:     c.cfg.Model,
		MaxTokens: defaultMaxTokens,
		Messages:  allMessages,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp response
	if err = json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	if apiResp.Error != nil {
		return "", fmt.Errorf("API error (%s): %s", apiResp.Error.Type, apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	return apiResp.Choices[0].Message.Content, nil
}

var codeBlockRegex = regexp.MustCompile("(?s)```(?:arc)?\\s*\\n(.*?)\\n```")

// ErrNoCodeBlock is returned when the LLM response does not contain a fenced
// code block. This typically means the model went off-topic.
var ErrNoCodeBlock = fmt.Errorf("response does not contain a fenced code block")

// ExtractArcCode pulls Arc source from a fenced code block in the LLM response.
// Returns ErrNoCodeBlock if no block is found.
func ExtractArcCode(response string) (string, error) {
	matches := codeBlockRegex.FindStringSubmatch(response)
	if len(matches) >= 2 {
		code := strings.TrimSpace(matches[1])
		if code == "" {
			return "", ErrNoCodeBlock
		}
		return code, nil
	}
	return "", ErrNoCodeBlock
}
