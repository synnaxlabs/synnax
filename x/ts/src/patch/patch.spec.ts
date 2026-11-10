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

  describe("pathOf", () => {
    it("should return the path of a set", () => {
      expect(
        patch.pathOf({ type: "set", set: { path: ["a", "b"], value: 1, guards: [] } }),
      ).toEqual(["a", "b"]);
    });

    it("should return the list path and key of a remove", () => {
      expect(
        patch.pathOf({ type: "remove", remove: { path: ["l"], key: "a", guards: [] } }),
      ).toEqual(["l", "a"]);
    });

    it("should return the list path and entry key of an insert", () => {
      expect(
        patch.pathOf({
          type: "insert",
          insert: { path: ["l"], value: { key: "a" }, guards: [] },
        }),
      ).toEqual(["l", "a"]);
    });

    it("should return the list path of an insert whose entry has no key", () => {
      expect(
        patch.pathOf({ type: "insert", insert: { path: ["l"], value: 1, guards: [] } }),
      ).toEqual(["l"]);
    });

    it("should return the list path and key of a move", () => {
      expect(
        patch.pathOf({ type: "move", move: { path: ["l"], key: "a", guards: [] } }),
      ).toEqual(["l", "a"]);
    });
  });

  describe("prefix", () => {
    const guards = [{ path: ["variant"], value: "tank" }];
    const prefixedGuards = [{ path: ["configs", "n1", "variant"], value: "tank" }];

    it("should prefix the path and guards of a set", () => {
      expect(
        patch.prefix({ type: "set", set: { path: ["label"], value: "T1", guards } }, [
          "configs",
          "n1",
        ]),
      ).toEqual({
        type: "set",
        set: { path: ["configs", "n1", "label"], value: "T1", guards: prefixedGuards },
      });
    });

    it("should prefix the path and guards of a remove", () => {
      expect(
        patch.prefix({ type: "remove", remove: { path: [], key: "color", guards } }, [
          "configs",
          "n1",
        ]),
      ).toEqual({
        type: "remove",
        remove: { path: ["configs", "n1"], key: "color", guards: prefixedGuards },
      });
    });

    it("should prefix the path of an insert and keep its position", () => {
      expect(
        patch.prefix(
          {
            type: "insert",
            insert: { path: ["l"], value: { key: "a" }, after: "b", guards: [] },
          },
          ["p"],
        ),
      ).toEqual({
        type: "insert",
        insert: { path: ["p", "l"], value: { key: "a" }, after: "b", guards: [] },
      });
    });

    it("should prefix the path of a move", () => {
      expect(
        patch.prefix({ type: "move", move: { path: ["l"], key: "a", guards: [] } }, [
          "p",
        ]),
      ).toEqual({ type: "move", move: { path: ["p", "l"], key: "a", guards: [] } });
    });

    it("should return the action unchanged for an empty path", () => {
      const action: patch.Action = {
        type: "set",
        set: { path: ["a"], value: 1, guards: [] },
      };
      expect(patch.prefix(action, [])).toBe(action);
    });
  });
});
