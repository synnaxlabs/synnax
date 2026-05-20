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
      const p = await client.projects.create({ name: "First Project" });
      expect(p.name).toEqual("First Project");
      expect(p.key).not.toEqual(uuid.ZERO);
    });
  });
  describe("rename", () => {
    test("renames the project", async () => {
      const p = await client.projects.create({ name: "Before" });
      await client.projects.rename(p.key, "After");
      const res = await client.projects.retrieve(p.key);
      expect(res.name).toEqual("After");
    });
  });
  describe("delete", () => {
    test("deletes the project", async () => {
      const p = await client.projects.create({ name: "ToDelete" });
      await client.projects.delete(p.key);
    });
  });
});
