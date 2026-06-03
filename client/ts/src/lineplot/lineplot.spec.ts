// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("LinePlot", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create(ws.key, { name: "Line Plot" });
      expect(linePlot.name).toEqual("Line Plot");
      expect(linePlot.key).not.toEqual(uuid.ZERO);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create(ws.key, { name: "Line Plot" });
      await client.lineplots.rename(linePlot.key, "Line Plot2");
      const res = await client.lineplots.retrieve({ key: linePlot.key });
      expect(res.name).toEqual("Line Plot2");
    });
  });
  describe("setData", () => {
    test("replaces the body while preserving key and name", async () => {
      const ws = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create(ws.key, { name: "Line Plot" });
      const next = await client.lineplots.create(ws.key, { name: "intermediate" });
      next.title.level = "h2";
      const lineColor = color.construct("#abcdef");
      next.lines = [
        {
          key: "l1",
          color: lineColor,
          strokeWidth: 2,
          downsample: 1,
          downsampleMode: "decimate",
        },
      ];
      await client.lineplots.setData(linePlot.key, next);
      const res = await client.lineplots.retrieve({ key: linePlot.key });
      expect(res.name).toEqual("Line Plot");
      expect(res.title.level).toEqual("h2");
      expect(res.lines).toHaveLength(1);
      expect(res.lines[0].color).toEqual(lineColor);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create(ws.key, { name: "Line Plot" });
      await client.lineplots.delete(linePlot.key);
      await expect(client.lineplots.retrieve({ key: linePlot.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
