// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { arc } from "@/arc";
import { AuthError, NotFoundError } from "@/errors";
import { createClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("arc", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve arcs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const a: arc.New = {
        name: "test",
        graph: {
          nodes: [],
          edges: [],
        },
        text: { raw: "" },
        deploy: false,
        version: "1.0.0",
      };
      const randomArc = await client.arcs.create(a);
      await expect(userClient.arcs.retrieve({ key: randomArc.key })).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve arcs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [arc.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomArc = await client.arcs.create({
        name: "test",
        graph: {
          nodes: [],
          edges: [],
        },
        text: { raw: "" },
        deploy: false,
        version: "1.0.0",
      });
      const retrieved = await userClient.arcs.retrieve({ key: randomArc.key });
      expect(retrieved.key).toBe(randomArc.key);
      expect(retrieved.name).toBe(randomArc.name);
    });

    it("should allow the caller to create arcs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [arc.ontologyID("")],
        actions: ["create"],
      });
      await userClient.arcs.create({
        name: "test",
        graph: {
          nodes: [],
          edges: [],
        },
        text: { raw: "" },
        deploy: false,
        version: "1.0.0",
      });
    });

    it("should prevent the caller to create arcs with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [arc.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.arcs.create({
          name: "test",
          graph: {
            nodes: [],
            edges: [],
          },
          text: { raw: "" },
          deploy: false,
          version: "1.0.0",
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete arcs with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [arc.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const randomArc = await client.arcs.create({
        name: "test",
        graph: {
          nodes: [],
          edges: [],
        },
        text: { raw: "" },
        deploy: false,
        version: "1.0.0",
      });
      await userClient.arcs.delete(randomArc.key);
      await expect(userClient.arcs.retrieve({ key: randomArc.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should prevent the caller to delete arcs with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [arc.ontologyID("")],
        actions: ["delete"],
      });
      const randomArc = await client.arcs.create({
        name: "test",
        graph: {
          nodes: [],
          edges: [],
        },
        text: { raw: "" },
        deploy: false,
        version: "1.0.0",
      });
      await expect(userClient.arcs.delete(randomArc.key)).rejects.toThrow(AuthError);
    });
  });
});
