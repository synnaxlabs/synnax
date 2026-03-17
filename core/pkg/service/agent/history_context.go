// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	sarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/gorp"
)

const maxHistoryExamples = 3

// HistoryContextProvider retrieves recent successful agent Arc programs and
// provides them as few-shot examples for the LLM. Research shows that 2-3
// real examples from the same system are more effective than synthetic ones.
type HistoryContextProvider struct {
	DB  *gorp.DB
	Arc *sarc.Service
}

var _ ContextProvider = (*HistoryContextProvider)(nil)

func (p *HistoryContextProvider) Name() string { return "history" }

func (p *HistoryContextProvider) BuildContext(ctx context.Context, _ string) (string, error) {
	var agents []Agent
	if err := gorp.NewRetrieve[uuid.UUID, Agent]().
		Entries(&agents).
		Exec(ctx, p.DB); err != nil {
		return "", nil
	}

	var examples []historyExample
	for _, a := range agents {
		if a.State != StateRunning || a.ArcKey == uuid.Nil {
			continue
		}
		instruction := firstUserMessage(a.Messages)
		if instruction == "" {
			continue
		}
		var arcEntry sarc.Arc
		if err := p.Arc.NewRetrieve().
			WhereKeys(a.ArcKey).
			Entry(&arcEntry).
			Exec(ctx, nil); err != nil {
			continue
		}
		if arcEntry.Text.Raw == "" {
			continue
		}
		examples = append(examples, historyExample{
			instruction: instruction,
			code:        arcEntry.Text.Raw,
		})
		if len(examples) >= maxHistoryExamples {
			break
		}
	}

	if len(examples) == 0 {
		return "", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Previously successful Arc programs on this system:\n\n")
	for i, ex := range examples {
		fmt.Fprintf(&b, "Example %d - Instruction: %q\n```arc\n%s\n```\n\n", i+1, ex.instruction, ex.code)
	}
	return b.String(), nil
}

type historyExample struct {
	instruction string
	code        string
}

func firstUserMessage(messages []Message) string {
	for _, m := range messages {
		if m.Role == RoleUser {
			return m.Content
		}
	}
	return ""
}
