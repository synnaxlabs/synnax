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
	"github.com/synnaxlabs/alamos"
	arcpkg "github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/agent/llm"
	sarc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// Writer handles transactional database operations for agents. It does NOT
// perform LLM generation. Use Service.GenerateAndDeploy for the full pipeline.
type Writer struct {
	tx        gorp.Tx
	otg       ontology.Writer
	generator llm.Generator
	arc       *sarc.Service
	task      *task.Service
	alamos.Instrumentation
}

// CreatePending saves a new agent in StateGenerating and defines its ontology
// resource. If the agent has no messages, it is saved in StateStopped instead.
func (w Writer) CreatePending(ctx context.Context, a *Agent) error {
	if a.Key == uuid.Nil {
		a.Key = uuid.New()
	}
	if a.Name == "" && len(a.Messages) > 0 {
		a.Name = a.Messages[0].Content
	}
	if len(a.Messages) == 0 {
		a.State = StateStopped
	} else {
		a.State = StateGenerating
	}

	w.L.Info("creating agent",
		zap.String("name", a.Name),
		zap.Int("messages", len(a.Messages)),
	)

	if err := gorp.NewCreate[uuid.UUID, Agent]().Entry(a).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DefineResource(ctx, OntologyID(a.Key))
}

// PrepareSend retrieves an existing agent, appends the user message, saves it
// in StateGenerating, and returns the updated agent.
func (w Writer) PrepareSend(ctx context.Context, key uuid.UUID, content string) (*Agent, error) {
	var a Agent
	if err := gorp.NewRetrieve[uuid.UUID, Agent]().
		WhereKeys(key).Entry(&a).Exec(ctx, w.tx); err != nil {
		return nil, err
	}

	a.Messages = append(a.Messages, Message{
		Role:    RoleUser,
		Content: content,
		Time:    telem.Now(),
	})
	a.State = StateGenerating

	w.L.Info("agent received message",
		zap.Stringer("key", key),
		zap.String("content", content),
		zap.Int("total_messages", len(a.Messages)),
	)

	if err := gorp.NewCreate[uuid.UUID, Agent]().Entry(&a).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return &a, nil
}

// deploy saves generated Arc code, creates or updates the task, and transitions
// the agent to StateRunning. Must be called within a transaction.
func (w Writer) deploy(ctx context.Context, a *Agent, code string, response string) error {
	a.Messages = append(a.Messages, Message{
		Role:    RoleAgent,
		Content: response,
		Time:    telem.Now(),
	})

	arcEntry := sarc.Arc{
		Key:  a.ArcKey,
		Name: fmt.Sprintf("Agent: %s", a.Name),
		Mode: sarc.Text,
		Text: text.Text{Raw: code},
		Graph: graph.Graph{
			Nodes: graph.Nodes{},
			Edges: graph.Edges{},
		},
	}
	arcWriter := w.arc.NewWriter(w.tx)
	if err := arcWriter.Create(ctx, &arcEntry); err != nil {
		return fmt.Errorf("failed to create Arc: %w", err)
	}
	if a.ArcKey == uuid.Nil {
		a.ArcKey = arcEntry.Key
		w.L.Info("created Arc", zap.Stringer("arc_key", arcEntry.Key))

		t := task.Task{
			Name: fmt.Sprintf("Agent: %s", a.Name),
			Type: "arc",
			Config: binary.MsgpackEncodedJSON{
				"arc_key":    arcEntry.Key.String(),
				"auto_start": true,
			},
		}
		t.Key = task.NewKey(a.RackKey, 0)
		taskWriter := w.task.NewWriter(w.tx)
		if err := taskWriter.Create(ctx, &t); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
		a.TaskKey = t.Key
		w.L.Info("created task", zap.Stringer("task_key", t.Key))

		if err := w.otg.DefineRelationship(
			ctx,
			OntologyID(a.Key),
			ontology.RelationshipTypeParentOf,
			sarc.OntologyID(arcEntry.Key),
		); err != nil {
			return err
		}
		if err := w.otg.DefineRelationship(
			ctx,
			sarc.OntologyID(arcEntry.Key),
			ontology.RelationshipTypeParentOf,
			task.OntologyID(t.Key),
		); err != nil {
			return err
		}
	} else {
		w.L.Info("updated Arc", zap.Stringer("arc_key", arcEntry.Key))
		if a.TaskKey != 0 {
			var existingTask task.Task
			if err := w.task.NewRetrieve().
				WhereKeys(a.TaskKey).
				Entry(&existingTask).
				Exec(ctx, w.tx); err == nil {
				taskWriter := w.task.NewWriter(w.tx)
				if err := taskWriter.Create(ctx, &existingTask); err != nil {
					w.L.Warn("failed to trigger task reconfigure", zap.Error(err))
				}
			}
		}
	}

	a.State = StateRunning
	w.L.Info("agent ready", zap.Stringer("key", a.Key), zap.String("state", string(a.State)))
	return gorp.NewCreate[uuid.UUID, Agent]().Entry(a).Exec(ctx, w.tx)
}

func (w Writer) Delete(ctx context.Context, keys ...uuid.UUID) error {
	for _, key := range keys {
		var a Agent
		if err := gorp.NewRetrieve[uuid.UUID, Agent]().
			WhereKeys(key).Entry(&a).Exec(ctx, w.tx); err != nil {
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			return err
		}
		if a.ArcKey != uuid.Nil {
			arcWriter := w.arc.NewWriter(w.tx)
			if err := arcWriter.Delete(ctx, a.ArcKey); err != nil && !errors.Is(err, query.ErrNotFound) {
				return err
			}
		}
	}
	if err := gorp.NewDelete[uuid.UUID, Agent]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}

// generator handles the LLM generation pipeline outside of a database transaction.
type generator struct {
	generator llm.Generator
	providers []ContextProvider
	arc       *sarc.Service
	alamos.Instrumentation
}

// generate runs channel search, LLM generation, and compilation. It does not
// touch the database. Returns the compiled Arc code and the full LLM response.
func (g *generator) generate(ctx context.Context, a *Agent) (code string, response string, err error) {
	searchQuery := latestUserMessage(a.Messages)

	g.L.Info("building context from providers")
	var contextSections []string
	for _, p := range g.providers {
		section, pErr := p.BuildContext(ctx, searchQuery)
		if pErr != nil {
			g.L.Warn("context provider failed", zap.String("provider", p.Name()), zap.Error(pErr))
			continue
		}
		if section != "" {
			contextSections = append(contextSections, section)
			g.L.Info("context provider contributed", zap.String("provider", p.Name()))
		}
	}
	if len(contextSections) == 0 {
		return "", "", fmt.Errorf("no context available (no matching channels found)")
	}
	contextStr := strings.Join(contextSections, "\n")

	agentMsgs := make([]llm.AgentMessage, len(a.Messages))
	for i, m := range a.Messages {
		agentMsgs[i] = llm.AgentMessage{Role: string(m.Role), Content: m.Content}
	}
	messages := llm.BuildMessagesWithContext(agentMsgs, contextStr)

	g.L.Info("calling LLM to generate Arc code")
	response, err = g.generator.Generate(ctx, llm.SystemPrompt, messages)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate Arc code: %w", err)
	}
	code, extractErr := llm.ExtractArcCode(response)
	if extractErr != nil {
		g.L.Warn("no code block in response, retrying", zap.Error(extractErr))
		return g.retry(ctx, messages, response,
			"Your response did not contain a fenced Arc code block. You MUST respond with an explanation followed by Arc code inside a ```arc code block. Do NOT reference this error in your response. Respond as if this is your first attempt.")
	}
	g.L.Info("LLM generated Arc code", zap.String("code", code))

	g.L.Info("compiling Arc code")
	if err = g.compileText(ctx, code); err != nil {
		g.L.Warn("compilation failed, retrying with error context", zap.Error(err))
		return g.retry(ctx, messages, response,
			fmt.Sprintf("The code failed to compile with error: %s\n\nPlease fix the code and output it in a ```arc code block. Do NOT reference the compilation error in your explanation. Respond as if this is your first attempt.", err.Error()))
	}
	g.L.Info("Arc code compiled successfully")
	return code, response, nil
}

func (g *generator) retry(
	ctx context.Context,
	messages []llm.Message,
	prevResponse string,
	feedback string,
) (string, string, error) {
	retryMsgs := append(messages, llm.Message{
		Role:    "assistant",
		Content: prevResponse,
	}, llm.Message{
		Role:    "user",
		Content: feedback,
	})
	retryResponse, err := g.generator.Generate(ctx, llm.SystemPrompt, retryMsgs)
	if err != nil {
		return "", "", fmt.Errorf("LLM retry failed: %w", err)
	}
	retryCode, extractErr := llm.ExtractArcCode(retryResponse)
	if extractErr != nil {
		return "", "", fmt.Errorf("LLM did not produce a code block after retry")
	}
	g.L.Info("LLM generated retry Arc code", zap.String("code", retryCode))
	if err = g.compileText(ctx, retryCode); err != nil {
		return "", "", fmt.Errorf("Arc compilation failed after retry: %w", err)
	}
	return retryCode, retryResponse, nil
}

func (g *generator) compileText(ctx context.Context, code string) error {
	t := text.Text{Raw: code}
	resolver := g.arc.SymbolResolver()
	_, err := arcpkg.CompileText(ctx, t, arcpkg.WithResolver(resolver))
	return err
}

func latestUserMessage(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			return messages[i].Content
		}
	}
	return ""
}
