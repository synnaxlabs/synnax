// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Draft, produce } from "immer";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { actions } from "@/actions";

interface DemoState {
  name: string;
  items: { key: string; value: number }[];
}

type DemoAction =
  | { type: "rename"; payload: { name: string } }
  | { type: "set_item"; payload: { key: string; value: number } }
  | { type: "remove_item"; payload: { key: string } };

const renameAction = (name: string): DemoAction => ({
  type: "rename",
  payload: { name },
});
const setItemAction = (key: string, value: number): DemoAction => ({
  type: "set_item",
  payload: { key, value },
});
const removeItemAction = (key: string): DemoAction => ({
  type: "remove_item",
  payload: { key },
});

const reduceOne = (
  draft: Draft<DemoState>,
  action: DemoAction,
): actions.HandlerResult<DemoAction> => {
  switch (action.type) {
    case "rename": {
      const prev = draft.name;
      draft.name = action.payload.name;
      return { inverse: [renameAction(prev)], targets: ["name"] };
    }
    case "set_item": {
      const idx = draft.items.findIndex((i) => i.key === action.payload.key);
      if (idx === -1) {
        draft.items.push(action.payload);
        return {
          inverse: [removeItemAction(action.payload.key)],
          targets: [action.payload.key],
        };
      }
      const prevValue = draft.items[idx].value;
      draft.items[idx].value = action.payload.value;
      return {
        inverse: [setItemAction(action.payload.key, prevValue)],
        targets: [action.payload.key],
      };
    }
    case "remove_item": {
      const idx = draft.items.findIndex((i) => i.key === action.payload.key);
      if (idx === -1) return actions.NO_OP_RESULT;
      const removed = draft.items[idx];
      draft.items.splice(idx, 1);
      return {
        inverse: [setItemAction(removed.key, removed.value)],
        targets: [removed.key],
      };
    }
  }
};

const reduceAll = actions.createReduceAll(reduceOne);

describe("snapshotDraft", () => {
  it("Should return plain values unchanged", () => {
    const v = { a: 1, b: [2, 3] };
    expect(actions.snapshotDraft(v)).toBe(v);
  });

  it("Should freeze a draft into a plain object that survives the produce closure", () => {
    const original = { items: [{ key: "a", value: 1 }] };
    let captured: typeof original | undefined;
    const next = produce(original, (draft) => {
      captured = actions.snapshotDraft(draft.items[0]) as unknown as typeof original;
      draft.items[0].value = 99;
    });
    expect(next.items[0].value).toBe(99);
    expect(captured).toEqual({ key: "a", value: 1 });
  });

  it("Should preserve primitive values without wrapping", () => {
    expect(actions.snapshotDraft(42)).toBe(42);
    expect(actions.snapshotDraft("hello")).toBe("hello");
    expect(actions.snapshotDraft(null)).toBe(null);
  });
});

describe("NO_OP_RESULT", () => {
  it("Should expose empty inverse and targets", () => {
    expect(actions.NO_OP_RESULT.inverse).toHaveLength(0);
    expect(actions.NO_OP_RESULT.targets).toHaveLength(0);
  });
});

describe("createReduceAll", () => {
  const initial: DemoState = { name: "before", items: [{ key: "a", value: 1 }] };

  it("Should apply a single action and return the new state", () => {
    const { next } = reduceAll(initial, [renameAction("after")]);
    expect(next.name).toBe("after");
    expect(initial.name).toBe("before");
  });

  it("Should accumulate inverses in reverse application order", () => {
    const { inverse } = reduceAll(initial, [
      renameAction("step-1"),
      setItemAction("a", 2),
      setItemAction("b", 7),
    ]);
    expect(inverse).toEqual([
      removeItemAction("b"),
      setItemAction("a", 1),
      renameAction("before"),
    ]);
  });

  it("Should round-trip state through forward then inverse application", () => {
    const forward = [
      renameAction("after"),
      setItemAction("a", 2),
      setItemAction("b", 7),
    ];
    const { next, inverse } = reduceAll(initial, forward);
    const { next: roundtripped } = reduceAll(next, inverse);
    expect(roundtripped).toEqual(initial);
  });

  it("Should de-duplicate targets across actions", () => {
    const { targets } = reduceAll(initial, [
      setItemAction("a", 5),
      setItemAction("a", 6),
      setItemAction("b", 7),
    ]);
    expect(targets).toHaveLength(2);
    expect(targets).toEqual(expect.arrayContaining(["a", "b"]));
  });

  it("Should drop NO_OP_RESULT entries from the inverse list", () => {
    const { inverse } = reduceAll(initial, [removeItemAction("missing")]);
    expect(inverse).toHaveLength(0);
  });

  it("Should leave the source state untouched after produce", () => {
    const before = JSON.parse(JSON.stringify(initial));
    reduceAll(initial, [
      renameAction("x"),
      setItemAction("a", 99),
      setItemAction("c", 3),
    ]);
    expect(initial).toEqual(before);
  });
});

describe("scopedZ", () => {
  const schema = actions.scopedZ(z.string(), z.object({ kind: z.string() }));

  it("Should parse a fully-populated envelope", () => {
    const parsed = schema.parse({
      key: "abc",
      dispatchKey: "dk",
      seq: 42,
      actions: [{ kind: "rename" }],
    });
    expect(parsed.key).toBe("abc");
    expect(parsed.seq).toBe(42);
    expect(parsed.actions).toEqual([{ kind: "rename" }]);
  });

  it("Should default seq to 0 when missing", () => {
    const parsed = schema.parse({ key: "abc", dispatchKey: "dk", actions: [] });
    expect(parsed.seq).toBe(0);
  });

  it("Should reject negative seq", () => {
    expect(() =>
      schema.parse({ key: "abc", dispatchKey: "dk", seq: -1, actions: [] }),
    ).toThrow();
  });

  it("Should reject non-integer seq", () => {
    expect(() =>
      schema.parse({ key: "abc", dispatchKey: "dk", seq: 1.5, actions: [] }),
    ).toThrow();
  });

  it("Should reject envelopes whose actions fail the inner schema", () => {
    expect(() =>
      schema.parse({
        key: "abc",
        dispatchKey: "dk",
        actions: [{ kind: 7 }],
      }),
    ).toThrow();
  });
});

describe("dispatchReqZ", () => {
  const schema = actions.dispatchReqZ(z.string(), z.object({ kind: z.string() }));

  it("Should parse a camelCase dispatchKey body", () => {
    const parsed = schema.parse({
      key: "abc",
      dispatchKey: "dk",
      actions: [{ kind: "x" }],
    });
    expect(parsed.dispatchKey).toBe("dk");
    expect(parsed.actions).toEqual([{ kind: "x" }]);
  });

  it("Should reject a body missing dispatchKey", () => {
    expect(() => schema.parse({ key: "abc", actions: [] })).toThrow();
  });
});
