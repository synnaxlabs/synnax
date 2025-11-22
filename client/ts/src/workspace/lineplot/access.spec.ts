// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";
import { lineplot } from "@/workspace/lineplot";

const client = createTestClient();

describe("lineplot", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve lineplots with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(
        userClient.workspaces.lineplots.retrieve({ key: randomLinePlot.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve lineplots with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [lineplot.ontologyID("")],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      const retrieved = await userClient.workspaces.lineplots.retrieve({
        key: randomLinePlot.key,
      });
      expect(retrieved.key).toBe(randomLinePlot.key);
      expect(retrieved.name).toBe(randomLinePlot.name);
    });

    it("should allow the caller to create lineplots with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [lineplot.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await userClient.workspaces.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
    });

    it("should prevent the caller to create lineplots with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [lineplot.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(
        userClient.workspaces.lineplots.create(ws.key, {
          name: "test",
          data: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete lineplots with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [lineplot.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      await userClient.workspaces.lineplots.delete(randomLinePlot.key);
      await expect(
        userClient.workspaces.lineplots.retrieve({ key: randomLinePlot.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete lineplots with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [lineplot.ontologyID("")],
        actions: ["delete"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(
        userClient.workspaces.lineplots.delete(randomLinePlot.key),
      ).rejects.toThrow(AuthError);
    });
  });
});
