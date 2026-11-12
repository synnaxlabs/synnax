// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type vcs } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import {
  changeCount,
  chooseAll,
  group,
  isReady,
  runningNote,
  toChoices,
} from "@/feature/vcs/choices";

const createItem = (
  resource: string,
  item: Partial<vcs.PlanItem> = {},
): vcs.PlanItem => ({
  resource,
  type: "schematic",
  name: resource,
  kind: "change",
  conflict: false,
  running: false,
  restarts: false,
  ...item,
});

const createPlan = (...items: vcs.PlanItem[]): vcs.Plan => ({
  target: { kind: "pull" },
  items,
});

const PLAN = createPlan(
  createItem("a"),
  createItem("b", { conflict: true }),
  createItem("c", { conflict: true }),
);

describe("choices", () => {
  describe("isReady", () => {
    it("should be ready for a plan without conflicts", () => {
      expect(isReady(createPlan(createItem("a")), {})).toBe(true);
    });

    it("should not be ready while a conflict has no choice", () => {
      expect(isReady(PLAN, { b: "mine" })).toBe(false);
    });

    it("should be ready once every conflict has a choice", () => {
      expect(isReady(PLAN, { b: "mine", c: "incoming" })).toBe(true);
    });
  });

  describe("chooseAll", () => {
    it("should make the choice on each conflict and on nothing else", () => {
      expect(chooseAll(PLAN, "incoming")).toEqual({ b: "incoming", c: "incoming" });
    });
  });

  describe("toChoices", () => {
    it("should return the choice of each conflict in plan order", () => {
      expect(toChoices(PLAN, { c: "incoming", b: "mine" })).toEqual([
        { resource: "b", choice: "mine" },
        { resource: "c", choice: "incoming" },
      ]);
    });

    it("should leave out unchosen conflicts and items without a conflict", () => {
      expect(toChoices(PLAN, { a: "mine", c: "mine" })).toEqual([
        { resource: "c", choice: "mine" },
      ]);
    });
  });

  describe("changeCount", () => {
    it("should count each item without a conflict", () => {
      expect(changeCount(PLAN, {})).toBe(1);
    });

    it("should count a conflict only when it takes the incoming version", () => {
      expect(changeCount(PLAN, { b: "incoming", c: "mine" })).toBe(2);
    });
  });

  describe("group", () => {
    it("should group items as new, changed, and deleted, sorted by name", () => {
      const plan = createPlan(
        createItem("d2", { kind: "delete", name: "Valve" }),
        createItem("c1", { kind: "create", name: "Pump" }),
        createItem("d1", { kind: "delete", name: "Tank" }),
        createItem("c2", { kind: "create", name: "Heater" }),
      );
      expect(
        group(plan).map(({ kind, title, items }) => ({
          kind,
          title,
          names: items.map(({ name }) => name),
        })),
      ).toEqual([
        { kind: "create", title: "New", names: ["Heater", "Pump"] },
        { kind: "delete", title: "Deleted", names: ["Tank", "Valve"] },
      ]);
    });

    it("should return no groups for an empty plan", () => {
      expect(group(createPlan())).toEqual([]);
    });
  });

  describe("runningNote", () => {
    it("should show no note for a stopped item", () => {
      expect(runningNote(createItem("a", { restarts: true }))).toBeNull();
    });

    it("should warn that a running item restarts", () => {
      expect(runningNote(createItem("a", { running: true, restarts: true }))).toBe(
        "Restarts on apply",
      );
    });

    it("should note that a running item waits for its next start", () => {
      expect(runningNote(createItem("a", { running: true }))).toBe(
        "Takes effect on next start",
      );
    });
  });
});
