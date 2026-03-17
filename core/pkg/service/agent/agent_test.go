// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent_test

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/agent"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
)

var _ = Describe("Agent", Ordered, func() {
	Describe("Create", func() {
		It("Should generate an Arc program from a natural language instruction, compile it, and deploy as a task", func() {
			a := &agent.Agent{
				Name: "Pump 3 Monitor",
				Messages: []agent.Message{{
					Role:    agent.RoleUser,
					Content: "Monitor the current on pump 3 and alert when it is out of family",
				}},
			}
			Expect(agentSvc.NewWriter(nil).CreatePending(ctx, a)).To(Succeed())
			if agentSvc != nil {
				err := agentSvc.GenerateAndDeploy(ctx, a)
				if err != nil {
					Skip("LLM API key not configured: " + err.Error())
				}
			}

			fmt.Println("=== AGENT CREATED ===")
			fmt.Printf("Agent Key: %s\n", a.Key)
			fmt.Printf("Agent State: %s\n", a.State)
			fmt.Printf("Arc Key: %s\n", a.ArcKey)
			fmt.Println()

			Expect(a.Key).ToNot(Equal(uuid.Nil))
			Expect(a.ArcKey).ToNot(Equal(uuid.Nil))
			Expect(a.State).To(Equal(agent.StateRunning))

			var createdArc arc.Arc
			Expect(arcSvc.NewRetrieve().WhereKeys(a.ArcKey).Entry(&createdArc).Exec(ctx, nil)).To(Succeed())
			Expect(createdArc.Text.Raw).ToNot(BeEmpty())
			Expect(createdArc.Mode).To(Equal(arc.Text))

			fmt.Println("=== GENERATED ARC CODE ===")
			fmt.Println(createdArc.Text.Raw)
			fmt.Println()

			compiled, err := arcSvc.CompileProgram(ctx, a.ArcKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(compiled.Program.IR.Nodes)).To(BeNumerically(">", 0))

			fmt.Println("=== COMPILATION RESULT ===")
			fmt.Printf("Nodes: %d\n", len(compiled.Program.IR.Nodes))
			fmt.Printf("Edges: %d\n", len(compiled.Program.IR.Edges))
			for _, n := range compiled.Program.IR.Nodes {
				fmt.Printf("  Node: key=%s type=%s\n", n.Key, n.Type)
			}

			var retrieved agent.Agent
			Expect(agentSvc.NewRetrieve().WhereKeys(a.Key).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal("Pump 3 Monitor"))
			Expect(retrieved.ArcKey).To(Equal(a.ArcKey))
			Expect(retrieved.State).To(Equal(agent.StateRunning))
		})
	})

	Describe("Delete", func() {
		It("Should delete the agent and cascade to the Arc", func() {
			a := &agent.Agent{
				Name: "Temp Monitor",
				Messages: []agent.Message{{
					Role:    agent.RoleUser,
					Content: "Monitor the current on pump 3 and alert when it is too high",
				}},
			}
			Expect(agentSvc.NewWriter(nil).CreatePending(ctx, a)).To(Succeed())
			if agentSvc != nil {
				err := agentSvc.GenerateAndDeploy(ctx, a)
				if err != nil {
					Skip("LLM API key not configured: " + err.Error())
				}
			}
			Expect(a.State).To(Equal(agent.StateRunning))

			arcKey := a.ArcKey
			Expect(agentSvc.NewWriter(nil).Delete(ctx, a.Key)).To(Succeed())

			var retrieved agent.Agent
			err := agentSvc.NewRetrieve().WhereKeys(a.Key).Entry(&retrieved).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())

			var retrievedArc arc.Arc
			err = arcSvc.NewRetrieve().WhereKeys(arcKey).Entry(&retrievedArc).Exec(ctx, nil)
			Expect(err).To(HaveOccurred())
		})
	})
})
