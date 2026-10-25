// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { patch } from "@/patch";

describe("patch", () => {
  describe("apply", () => {
    it("should return no inverse for an action that does nothing", () => {
      const doc = { channels: [{ key: "a" }] };
      expect(
        patch.apply(doc, {
          type: "set",
          set: { path: ["channels", "b", "port"], value: 1, guards: [] },
        }),
      ).toEqual([]);
    });

    it("should copy the value it stores", () => {
      const value = { rate: 1 };
      const doc: Record<string, unknown> = {};
      patch.apply(doc, { type: "set", set: { path: ["config"], value, guards: [] } });
      value.rate = 2;
      expect(doc).toEqual({ config: { rate: 1 } });
    });

    it("should snapshot every value an action replaces", () => {
      const snapped: unknown[] = [];
      const snapshot = <T>(v: T): T => {
        snapped.push(v);
        return structuredClone(v);
      };
      const doc = { config: { rate: 1 }, channels: [{ key: "a" }] };
      patch.apply(
        doc,
        { type: "set", set: { path: ["config"], value: {}, guards: [] } },
        { snapshot },
      );
      patch.apply(
        doc,
        { type: "remove", remove: { path: ["channels"], key: "a", guards: [] } },
        { snapshot },
      );
      expect(snapped).toEqual([{ rate: 1 }, { key: "a" }]);
    });

    it("should carry the guards onto the inverse", () => {
      const doc = { a: { type: "x", v: 1 } };
      const guards = [{ path: ["a", "type"], value: "x" }];
      expect(
        patch.apply(doc, { type: "set", set: { path: ["a", "v"], value: 2, guards } }),
      ).toEqual([{ type: "set", set: { path: ["a", "v"], value: 1, guards } }]);
    });

    it("should undo a move to the front by moving the entry back after its neighbor", () => {
      const doc = { l: [{ key: "a" }, { key: "b" }, { key: "c" }] };
      expect(
        patch.apply(doc, { type: "move", move: { path: ["l"], key: "c", guards: [] } }),
      ).toEqual([
        { type: "move", move: { path: ["l"], key: "c", after: "b", guards: [] } },
      ]);
    });
  });
});
