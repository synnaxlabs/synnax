// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, type patch } from "@synnaxlabs/x";
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

describe("applyPath", () => {
  const docZ = z.object({
    maxSize: z.number(),
    params: caseconv.preserveCase(z.record(z.string(), z.unknown())),
  });
  type Doc = z.infer<typeof docZ>;
  const apply = (doc: Doc, action: patch.Action) => {
    let inverse: patch.Action[] = [];
    const next = produce(doc, (draft) => {
      inverse = actions.applyPath(draft, action, docZ);
    });
    return { next, inverse };
  };

  it("should apply a wire-form path to the camelCase document", () => {
    const { next, inverse } = apply(
      { maxSize: 1, params: {} },
      { type: "set", set: { path: ["max_size"], value: 2, guards: [] } },
    );
    expect(next.maxSize).toBe(2);
    expect(inverse).toEqual([
      { type: "set", set: { path: ["max_size"], value: 1, guards: [] } },
    ]);
  });

  it("should keep the case of a key the schema preserves", () => {
    const { next } = apply(
      { maxSize: 1, params: {} },
      { type: "set", set: { path: ["params", "minValue"], value: 2, guards: [] } },
    );
    expect(next.params).toEqual({ minValue: 2 });
  });

  it("should return no inverse when a guard fails", () => {
    const { next, inverse } = apply(
      { maxSize: 1, params: {} },
      {
        type: "set",
        set: { path: ["max_size"], value: 2, guards: [{ path: ["kind"], value: "a" }] },
      },
    );
    expect(next.maxSize).toBe(1);
    expect(inverse).toEqual([]);
  });
});

describe("applyEntryPath", () => {
  const docZ = z.object({
    name: z.string(),
    graph: z.object({ inputs: z.record(z.string(), z.record(z.string(), z.number())) }),
  });
  type Doc = z.infer<typeof docZ>;
  const DOC: Doc = { name: "a", graph: { inputs: { n1: { value: 1 } } } };
  const ROOT = ["graph", "inputs"];
  const setAt = (path: string[], value: unknown, guards: patch.Guard[] = []) => ({
    type: "set" as const,
    set: { path, value, guards },
  });
  const apply = (action: patch.Action) => {
    let result: actions.HandlerResult<patch.Action> = actions.NO_OP_RESULT;
    const next = produce(DOC, (draft) => {
      result = actions.applyEntryPath(draft, action, ROOT, docZ);
    });
    return { next, result };
  };

  it("should apply an action inside an entry and target that entry", () => {
    const { next, result } = apply(setAt(["graph", "inputs", "n1", "value"], 2));
    expect(next.graph.inputs.n1.value).toBe(2);
    expect(result).toEqual({
      inverse: [setAt(["graph", "inputs", "n1", "value"], 1)],
      targets: ["n1"],
    });
  });

  it.each([
    [["name"]],
    [["graph"]],
    [["graph", "inputs"]],
    [["graph", "outputs", "n1"]],
  ])("should throw on an action at %j", (path) => {
    expect(() => apply(setAt(path, 2))).toThrow(
      `${path.join(".")} is not inside an entry of graph.inputs`,
    );
  });

  it("should return NO_OP_RESULT when a guard fails", () => {
    const { next, result } = apply(
      setAt(["graph", "inputs", "n1", "value"], 2, [{ path: ["kind"], value: "a" }]),
    );
    expect(next).toBe(DOC);
    expect(result).toBe(actions.NO_OP_RESULT);
  });
});

describe("createPathHandlers", () => {
  const set: patch.Set = { path: ["a"], value: 1, guards: [] };
  const remove: patch.Remove = { path: [], key: "a", guards: [] };
  const insert: patch.Insert = { path: ["a"], value: 1, after: "", guards: [] };
  const move: patch.Move = { path: ["a"], key: "b", after: "", guards: [] };

  it.each<patch.Action>([
    { type: "set", set },
    { type: "remove", remove },
    { type: "insert", insert },
    { type: "move", move },
  ])("should pass a $type payload to apply as a $type action", (action) => {
    const received: patch.Action[] = [];
    const result = { inverse: [action], targets: ["t"] };
    const handlers = actions.createPathHandlers((_: null, a: patch.Action) => {
      received.push(a);
      return result;
    });
    const payload = (action as Record<string, unknown>)[action.type];
    const handler = handlers[action.type] as (s: null, p: unknown) => unknown;
    expect(handler(null, payload)).toBe(result);
    expect(received).toEqual([action]);
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

describe("dispatchResZ", () => {
  const schema = actions.dispatchResZ(z.object({ kind: z.string() }));

  it("Should parse the actions the Core added", () => {
    expect(schema.parse({ actions: [{ kind: "x" }] })).toEqual({
      actions: [{ kind: "x" }],
    });
  });

  it("Should parse an empty body as no actions", () => {
    expect(schema.parse({})).toEqual({ actions: [] });
  });
});
