// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Project", () => {
  describe("create", () => {
    test("create one", async () => {
      const p = await client.projects.create({ name: "Test Cell A" });
      expect(p.name).toEqual("Test Cell A");
      expect(p.key).not.toEqual(uuid.ZERO);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const p = await client.projects.create({ name: "Test Cell A" });
      await client.projects.rename(p.key, "Test Cell B");
      const res = await client.projects.retrieve(p.key);
      expect(res.name).toEqual("Test Cell B");
    });
  });
  describe("retrieve", () => {
    test("retrieve one by key", async () => {
      const p = await client.projects.create({ name: "Test Cell A" });
      const res = await client.projects.retrieve(p.key);
      expect(res.name).toEqual("Test Cell A");
    });
    test("retrieve many by keys", async () => {
      const p1 = await client.projects.create({ name: "Project 1" });
      const p2 = await client.projects.create({ name: "Project 2" });
      const res = await client.projects.retrieve([p1.key, p2.key]);
      expect(res).toHaveLength(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const p = await client.projects.create({ name: "Test Cell A" });
      await client.projects.delete(p.key);
      await expect(client.projects.retrieve(p.key)).rejects.toThrow();
    });
  });
});
