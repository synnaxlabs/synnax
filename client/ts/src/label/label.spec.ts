// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { label } from "@/label";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Label", () => {
  describe("create", () => {
    it("should create a label", async () => {
      const v = await client.labels.create({
        name: "Label",
        color: "#E774D0",
      });
      expect(v.key).not.toHaveLength(0);
    });
  });

  describe("retrieve", () => {
    it("should retrieve a label by its key", async () => {
      const v = await client.labels.create({
        name: "Label",
        color: "#E774D0",
      });
      const retrieved = await client.labels.retrieve({ key: v.key });
      expect(retrieved).toEqual(v);
    });
  });

  describe("delete", () => {
    it("should delete a label by its key", async () => {
      const v = await client.labels.create({
        name: "Label",
        color: "#E774D0",
      });
      await client.labels.delete(v.key);
      await expect(
        async () => await client.labels.retrieve({ key: v.key }),
      ).rejects.toThrow();
    });
  });

  describe("label", () => {
    it("should set a label on an item", async () => {
      const l1 = await client.labels.create({
        name: "Label One",
        color: "#E774D0",
      });
      const l2 = await client.labels.create({
        name: "Label Two",
        color: "#E774D0",
      });
      await client.labels.label(label.ontologyID(l1.key), [l2.key]);
      const labels = await client.labels.retrieve({ for: label.ontologyID(l1.key) });
      expect(labels).toHaveLength(1);
      expect(labels[0].key).toEqual(l2.key);
    });
    it("should replace the labels on an item", async () => {
      const l1 = await client.labels.create({
        name: "Label One",
        color: "#E774D0",
      });
      const l2 = await client.labels.create({
        name: "Label Two",
        color: "#E774D0",
      });
      await client.labels.label(label.ontologyID(l1.key), [l2.key]);
      await client.labels.label(label.ontologyID(l1.key), [l1.key], { replace: true });
      const labels = await client.labels.retrieve({ for: label.ontologyID(l1.key) });
      expect(labels).toHaveLength(1);
      expect(labels[0].key).toEqual(l1.key);
    });
  });
});
