// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { lineplot } from "@/lineplot";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("lineplot", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(
        userClient.lineplots.retrieve({ key: randomLinePlot.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve lineplots with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [lineplot.ontologyID("")],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      const retrieved = await userClient.lineplots.retrieve({
        key: randomLinePlot.key,
      });
      expect(retrieved.key).toBe(randomLinePlot.key);
      expect(retrieved.name).toBe(randomLinePlot.name);
    });

    it("should allow the caller to create lineplots with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [lineplot.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await userClient.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [lineplot.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(
        userClient.lineplots.create(ws.key, {
          name: "test",
          data: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete lineplots with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [lineplot.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      await userClient.lineplots.delete(randomLinePlot.key);
      await expect(
        userClient.lineplots.retrieve({ key: randomLinePlot.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [lineplot.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomLinePlot = await client.lineplots.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(userClient.lineplots.delete(randomLinePlot.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
