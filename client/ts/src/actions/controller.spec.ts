// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor, observe, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import z from "zod";

import {
  Controller,
  type Edit,
  type Frame,
  type Reducer,
  type SendDispatch,
} from "@/actions/controller";
import { Table } from "@/query/table";

interface Doc {
  values: Record<string, number>;
}

type Action =
  | { type: "set"; key: string; value: number }
  | { type: "tag"; key: string; tag: string }
  | { type: "create"; doc: Doc }
  | { type: "invalid" }
  | { type: "noop" };

const reducer: Reducer<Doc, Action> = (state, actions) => {
  let next = state;
  const inverse: Action[] = [];
  const targets = new Set<string>();
  for (const a of actions)
    if (a.type === "set") {
      const prev = next.values[a.key] ?? 0;
      next = { values: { ...next.values, [a.key]: a.value } };
      inverse.unshift({ type: "set", key: a.key, value: prev });
      targets.add(a.key);
    } else if (a.type === "tag")
      // Touches a target but contributes no inverse — covers the "marks
      // targets without producing reversible work" branch.
      targets.add(a.key);
    else if (a.type === "create") next = a.doc;
    else if (a.type === "invalid") throw new Error("the doc cannot hold the action");
  return { next, inverse, targets: [...targets] };
};

const frameZ = z.object({
  key: z.string(),
  dispatchKey: z.string(),
  seq: z.number().int().nonnegative(),
  actions: z.array(z.any()),
}) as unknown as z.ZodType<Frame<string, Action>>;

/** A manually advanced clock, so coalesce-window tests need no real sleeps. */
const createFakeClock = () => {
  let current = TimeStamp.now();
  return {
    now: () => current,
    advance: (span: TimeSpan) => {
      current = current.add(span);
    },
  };
};

const setupStore = (
  opts: Partial<{
    isUndoable: (a: Action) => boolean;
    kindOf: (acts: Action[]) => string;
    coalesceWindow: TimeSpan;
    stackCap: number;
    now: () => TimeStamp;
    fetch: (keys: string[]) => Promise<Array<Doc & { key: string }>>;
    encode: (doc: Doc, action: Action) => Action;
    send: SendDispatch<string, Action>;
  }> = {},
) => {
  const errors: Error[] = [];
  const onError = (error: Error) => errors.push(error);
  const docs = new Table<string, Doc>({ onError, fetch: opts.fetch });
  const epochs = new observe.Observer<number>();
  const send = vi.fn<SendDispatch<string, Action>>(opts.send ?? (async () => []));
  const controller = new Controller<string, Doc, Action>({
    store: docs,
    onError,
    onEpoch: (callback) => epochs.onChange(callback),
    reduce: reducer,
    send,
    encode: opts.encode,
    isUndoable: opts.isUndoable,
    kindOf: opts.kindOf,
    createOf: (a) => (a.type === "create" ? a.doc : undefined),
    coalesceWindow: opts.coalesceWindow,
    stackCap: opts.stackCap,
    now: opts.now,
  });
  return { errors, docs, epochs, controller, send };
};

const set = (key: string, value: number): Action => ({ type: "set", key, value });

interface HeldCall {
  actions: Action[];
  dispatchKey: string;
  /** Resolves the send with the actions the Core added to the dispatch. */
  resolve: (added?: Action[]) => void;
  reject: (error: Error) => void;
}

/** A send that settles only when the spec says, to hold a dispatch pending. */
const createHeldSend = () => {
  const calls: HeldCall[] = [];
  const send: SendDispatch<string, Action> = async (_, actions, dispatchKey) =>
    await new Promise<Action[]>((resolve, reject) =>
      calls.push({
        actions,
        dispatchKey,
        resolve: (added = []) => resolve(added),
        reject,
      }),
    );
  return { send, calls };
};

const prime = (
  docs: Table<string, Doc>,
  key: string,
  values: Record<string, number> = {},
) => docs.set(key, { values });

describe("actions.Controller", () => {
  describe("dispatch", () => {
    it("returns false and sends nothing when the doc is not cached", async () => {
      const { controller, send } = setupStore();
      expect(await controller.dispatch("missing", [set("a", 1)])).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("applies the actions, sends them with a dispatch key, and records an undo entry", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 5 });
      expect(await controller.dispatch("k", [set("a", 9)])).toBe(true);
      expect(docs.get("k")).toEqual({ values: { a: 9 } });
      expect(send).toHaveBeenCalledWith("k", [set("a", 9)], expect.any(String));
      expect(controller.prepareUndo("k")?.actions).toEqual([set("a", 5)]);
    });

    it("sends nothing when no action changes the doc", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 5 });
      const before = docs.get("k");
      expect(await controller.dispatch("k", [{ type: "noop" }])).toBe(true);
      expect(send).not.toHaveBeenCalled();
      expect(docs.get("k")).toBe(before);
      expect(controller.hasPending("k")).toBe(false);
    });

    it("runs the preprocess option against the cached doc", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 3 });
      await controller.dispatch("k", [set("a", 4)], {
        preprocess: (doc, acts) => [...acts, set("b", doc.values.a)],
      });
      expect(docs.get("k")).toEqual({ values: { a: 4, b: 3 } });
      expect(send).toHaveBeenCalledWith(
        "k",
        [set("a", 4), set("b", 3)],
        expect.any(String),
      );
    });

    it("removes the local apply and the undo entry when the send fails", async () => {
      const { docs, controller } = setupStore({
        send: async () => {
          throw new Error("boom");
        },
      });
      prime(docs, "k", { a: 0 });
      await expect(controller.dispatch("k", [set("a", 1)])).rejects.toThrow("boom");
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
      expect(controller.hasUndo("k")).toBe(false);
      expect(controller.hasPending("k")).toBe(false);
    });

    it("keeps a foreign change that lands before a failed send", async () => {
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0, b: 0 });
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      controller.applyRemote("k", 1, "foreign", [set("b", 7)]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 7 } });
      calls[0].reject(new Error("boom"));
      await expect(dispatched).rejects.toThrow("boom");
      expect(docs.get("k")).toEqual({ values: { a: 0, b: 7 } });
    });

    it("holds the dispatch pending until its echo arrives", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      expect(controller.hasPending("k")).toBe(true);
      const dispatchKey = send.mock.calls[0][2];
      controller.applyRemote("k", 1, dispatchKey, [set("a", 1)]);
      expect(controller.hasPending("k")).toBe(false);
    });
  });

  describe("recordEntry", () => {
    it("is a no-op when targets is empty", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k");
      controller.recordEntry("k", [{ type: "set", key: "a", value: 1 }], [], []);
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("is a no-op when all forward actions are non-undoable", () => {
      const { docs, controller } = setupStore({
        isUndoable: (a) => a.type !== "noop",
      });
      prime(docs, "k");
      controller.recordEntry("k", [{ type: "noop" }], [], ["a"]);
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("derives kind via kindOf", () => {
      const kindOf = vi.fn((acts: Action[]) => acts[0].type);
      const { docs, controller } = setupStore({ kindOf });
      prime(docs, "k");
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(kindOf).toHaveBeenCalledTimes(1);
    });

    it("uses kindOverride when supplied without calling kindOf", () => {
      const kindOf = vi.fn(() => "auto");
      const { docs, controller } = setupStore({ kindOf });
      prime(docs, "k");
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
        "manual",
      );
      expect(kindOf).not.toHaveBeenCalled();
    });

    it("clears the redo stack on a new entry", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      const undo = controller.prepareUndo("k");
      undo?.commit();
      expect(controller.hasRedo("k")).toBe(true);
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 2 }],
        [{ type: "set", key: "a", value: 1 }],
        ["a"],
      );
      expect(controller.hasRedo("k")).toBe(false);
    });
  });

  describe("coalescing", () => {
    const push = (
      controller: Controller<string, Doc, Action>,
      key: string,
      target: string,
      value: number,
      kind: string,
    ) =>
      controller.recordEntry(
        key,
        [{ type: "set", key: target, value }],
        [{ type: "set", key: target, value: value - 1 }],
        [target],
        kind,
      );

    it("merges entries with same kind and same targets within the window", () => {
      const { docs, controller } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "move");
      push(controller, "k", "a", 2, "move");
      push(controller, "k", "a", 3, "move");
      // One coalesced entry — single undo clears the stack.
      const r = controller.prepareUndo("k");
      r?.commit();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("does not merge across kinds", () => {
      const { docs, controller } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "move");
      push(controller, "k", "a", 2, "config");
      // Two entries — two undos required.
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(true);
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("does not merge when targets differ", () => {
      const { docs, controller } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(docs, "k", { a: 0, b: 0 });
      push(controller, "k", "a", 1, "move");
      push(controller, "k", "b", 1, "move");
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(true);
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("does not merge when the prior entry is older than the window", () => {
      const clock = createFakeClock();
      const { docs, controller } = setupStore({
        coalesceWindow: TimeSpan.NANOSECOND,
        now: clock.now,
      });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "move");
      // Move the next entry's stamp past the 1ns window.
      clock.advance(TimeSpan.milliseconds(2));
      push(controller, "k", "a", 2, "move");
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(true);
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("preserves stack ordering when merging — newest inverse undoes first", () => {
      const { docs, controller } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "move");
      push(controller, "k", "a", 2, "move");
      const r = controller.prepareUndo("k");
      // The merged inverse should restore to the pre-first-push state.
      // First inverse runs the latest's inverse (a:1), then the prior's (a:0).
      expect(r?.actions[r.actions.length - 1]).toEqual({
        type: "set",
        key: "a",
        value: 0,
      });
    });

    it("marks a merged entry stale when a remote touch lands mid-merge", () => {
      const { docs, controller } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "move");
      controller.markRemoteTouched("k", ["a"]);
      push(controller, "k", "a", 2, "move");
      // Undoing the merged entry would restore "a" to 0, wiping the remote
      // edit that landed between the two pushes.
      expect(controller.prepareUndo("k")).toBeNull();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("trims to stackCap by dropping the oldest", () => {
      const clock = createFakeClock();
      const { docs, controller } = setupStore({
        coalesceWindow: TimeSpan.NANOSECOND,
        stackCap: 2,
        now: clock.now,
      });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "k1");
      clock.advance(TimeSpan.milliseconds(2));
      push(controller, "k", "a", 2, "k2");
      clock.advance(TimeSpan.milliseconds(2));
      push(controller, "k", "a", 3, "k3");
      controller.prepareUndo("k")?.commit();
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(false);
    });
  });

  describe("prepareUndo", () => {
    it("returns null when the stack is empty", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k");
      expect(controller.prepareUndo("k")).toBeNull();
    });

    it("returns the top entry's inverse and pushes a redo on commit", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      const r = controller.prepareUndo("k");
      expect(r?.actions).toEqual([{ type: "set", key: "a", value: 0 }]);
      r?.commit();
      expect(controller.hasUndo("k")).toBe(false);
      expect(controller.hasRedo("k")).toBe(true);
    });

    it("skips a stale top entry (target touched after the entry's ts)", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0, b: 0 });
      // Two entries, oldest first.
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.recordEntry(
        "k",
        [{ type: "set", key: "b", value: 1 }],
        [{ type: "set", key: "b", value: 0 }],
        ["b"],
      );
      // Mark "b" as remote-touched after its entry → top entry is stale.
      controller.markRemoteTouched("k", ["b"]);
      const r = controller.prepareUndo("k");
      // Walks past the stale "b" entry and returns "a"'s inverse.
      expect(r?.actions).toEqual([{ type: "set", key: "a", value: 0 }]);
    });

    it("returns null and drops stale entries when the entire stack is stale", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.markRemoteTouched("k", ["a"]);
      expect(controller.prepareUndo("k")).toBeNull();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("ignores a remote touch older than the entry", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Record the touch before the entry exists.
      controller.markRemoteTouched("k", ["a"]);
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      // The entry was pushed after the touch — it is not stale.
      expect(controller.prepareUndo("k")?.actions).toEqual([
        { type: "set", key: "a", value: 0 },
      ]);
    });
  });

  describe("prepareRedo", () => {
    it("returns null when the redo stack is empty", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k");
      expect(controller.prepareRedo("k")).toBeNull();
    });

    it("returns the original forward to re-apply, and pushes back to undo on commit", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.prepareUndo("k")?.commit();
      const r = controller.prepareRedo("k");
      // Redo must re-apply the original forward, not the inverse.
      expect(r?.actions).toEqual([{ type: "set", key: "a", value: 1 }]);
      r?.commit();
      expect(controller.hasRedo("k")).toBe(false);
      expect(controller.hasUndo("k")).toBe(true);
    });

    it("skips a stale top entry (target touched after the entry's ts)", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0, b: 0 });
      // Two entries pushed then undone, oldest first into redo.
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.recordEntry(
        "k",
        [{ type: "set", key: "b", value: 1 }],
        [{ type: "set", key: "b", value: 0 }],
        ["b"],
      );
      controller.prepareUndo("k")?.commit();
      controller.prepareUndo("k")?.commit();
      // Mark "a" as remote-touched after its entry → top of redo is stale.
      controller.markRemoteTouched("k", ["a"]);
      const r = controller.prepareRedo("k");
      // Walks past the stale "a" entry and returns "b"'s forward.
      expect(r?.actions).toEqual([{ type: "set", key: "b", value: 1 }]);
    });

    it("returns null and drops stale entries when the entire redo stack is stale", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.prepareUndo("k")?.commit();
      controller.markRemoteTouched("k", ["a"]);
      expect(controller.prepareRedo("k")).toBeNull();
      expect(controller.hasRedo("k")).toBe(false);
    });

    it("ignores a remote touch older than the redo entry", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.markRemoteTouched("k", ["a"]);
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.prepareUndo("k")?.commit();
      expect(controller.prepareRedo("k")?.actions).toEqual([
        { type: "set", key: "a", value: 1 },
      ]);
    });

    it("undo → redo → undo cycle restores state correctly at each step", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      expect(docs.get("k")?.values.a).toBe(1);
      expect(await controller.undo("k")).toBe(true);
      expect(docs.get("k")?.values.a).toBe(0);
      expect(await controller.redo("k")).toBe(true);
      expect(docs.get("k")?.values.a).toBe(1);
      expect(await controller.undo("k")).toBe(true);
      expect(docs.get("k")?.values.a).toBe(0);
    });
  });

  describe("send queue", () => {
    // A send that waits on the returned gate before its first call settles.
    const createGatedSend = () => {
      const gate = Promise.withResolvers<void>();
      const send = vi.fn<SendDispatch<string, Action>>(async () => {
        if (send.mock.calls.length === 1) await gate.promise;
        return [];
      });
      return { gate, send };
    };

    it("sends a dispatch at once when nothing is in flight", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const done = controller.dispatch("k", [set("a", 1)]);
      expect(send).toHaveBeenCalledTimes(1);
      expect(await done).toBe(true);
      expect(send.mock.calls[0][1]).toEqual([set("a", 1)]);
    });

    it("sends dispatches made during an in-flight send as one batch after it settles", async () => {
      const { gate, send } = createGatedSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("a", 2)]);
      const three = controller.dispatch("k", [set("b", 3)]);
      expect(send).toHaveBeenCalledTimes(1);
      gate.resolve();
      expect(await Promise.all([one, two, three])).toEqual([true, true, true]);
      expect(send).toHaveBeenCalledTimes(2);
      const [[, first, firstKey], [, batch, batchKey]] = send.mock.calls;
      expect(first).toEqual([set("a", 1)]);
      expect(batch).toEqual([set("a", 2), set("b", 3)]);
      expect(batchKey).not.toEqual(firstKey);
      expect(docs.get("k")).toEqual({ values: { a: 2, b: 3 } });
    });

    it("sends dispatches for different documents in parallel", async () => {
      const { send } = createGatedSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k1", { a: 0 });
      prime(docs, "k2", { a: 0 });
      void controller.dispatch("k1", [set("a", 1)]);
      void controller.dispatch("k2", [set("a", 1)]);
      expect(send.mock.calls.map(([key]) => key)).toEqual(["k1", "k2"]);
    });

    it("holds nothing pending once the echo of a batch arrives", async () => {
      const { gate, send } = createGatedSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("b", 2)]);
      const three = controller.dispatch("k", [set("c", 3)]);
      gate.resolve();
      await Promise.all([one, two, three]);
      const [[, first, firstKey], [, batch, batchKey]] = send.mock.calls;
      controller.applyRemote("k", 1, firstKey, first);
      controller.applyRemote("k", 2, batchKey, batch);
      expect(controller.hasPending("k")).toBe(false);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 2, c: 3 } });
      expect(controller.hasUndo("k")).toBe(true);
    });

    // Fails the sends whose call numbers are listed, counting from 1.
    const createFailingSend = (...failing: number[]) => {
      const send = vi.fn<SendDispatch<string, Action>>(async () => {
        if (failing.includes(send.mock.calls.length)) throw new Error("rejected");
        return [];
      });
      return send;
    };

    // Sets a value outside the reducer, as a client's rename does.
    const name = (docs: Table<string, Doc>, value: number): Edit => {
      const reapply = () =>
        docs.set("k", (d) => (d == null ? d : { values: { ...d.values, n: value } }));
      return { rollback: reapply(), reapply };
    };

    it("applies the batch over the state before a failed in-flight dispatch", async () => {
      const send = createFailingSend(1);
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("a", 2)]);
      await expect(one).rejects.toThrow("rejected");
      expect(await two).toBe(true);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      const [, batch, batchKey] = send.mock.calls[1];
      controller.applyRemote("k", 1, batchKey, batch);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      expect(controller.prepareUndo("k")?.actions).toEqual([set("a", 0)]);
    });

    it("restores the original state when the in-flight dispatch and the batch fail", async () => {
      const { docs, controller } = setupStore({ send: createFailingSend(1, 2) });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("a", 2)]);
      await expect(one).rejects.toThrow("rejected");
      await expect(two).rejects.toThrow("rejected");
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
      expect(controller.hasUndo("k")).toBe(false);
      expect(controller.hasPending("k")).toBe(false);
    });

    it("moves the rebuilt entry to redo for a queued undo after a failed send", async () => {
      const send = createFailingSend(1);
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0, b: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("b", 2)]);
      const undone = controller.undo("k");
      await expect(one).rejects.toThrow("rejected");
      expect(await Promise.all([two, undone])).toEqual([true, true]);
      expect(send.mock.calls[1][1]).toEqual([set("b", 2), set("b", 0)]);
      expect(docs.get("k")).toEqual({ values: { a: 0, b: 0 } });
      expect(controller.hasUndo("k")).toBe(false);
      expect(controller.prepareRedo("k")?.actions).toEqual([set("b", 2)]);
    });

    it("drops a queued undo of a dispatch whose send failed", async () => {
      const send = createFailingSend(2);
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0, b: 0 });
      expect(await controller.dispatch("k", [set("a", 1)])).toBe(true);
      const two = controller.dispatch("k", [set("b", 2)]);
      const undone = controller.undo("k");
      await expect(two).rejects.toThrow("rejected");
      expect(await undone).toBe(true);
      expect(send).toHaveBeenCalledTimes(2);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 0 } });
      expect(controller.prepareUndo("k")?.actions).toEqual([set("a", 0)]);
      expect(controller.hasRedo("k")).toBe(false);
    });

    it("keeps a batched rename when only the in-flight dispatch fails", async () => {
      const { docs, controller } = setupStore({ send: createFailingSend(1) });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const renamed = controller.send("k", [{ type: "noop" }], name(docs, 7));
      const two = controller.dispatch("k", [set("b", 2)]);
      await expect(one).rejects.toThrow("rejected");
      await Promise.all([renamed, two]);
      expect(docs.get("k")).toEqual({ values: { a: 0, n: 7, b: 2 } });
    });

    it("rolls back a batched rename in order with the batch's dispatches", async () => {
      const { docs, controller } = setupStore({ send: createFailingSend(1, 2) });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("b", 2)]);
      const renamed = controller.send("k", [{ type: "noop" }], name(docs, 7));
      const three = controller.dispatch("k", [set("c", 3)]);
      await expect(one).rejects.toThrow("rejected");
      await expect(two).rejects.toThrow("rejected");
      await expect(renamed).rejects.toThrow("rejected");
      await expect(three).rejects.toThrow("rejected");
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
    });

    it("sends a rename for a doc that is not cached", async () => {
      const { controller, send } = setupStore();
      await controller.send("missing", [{ type: "noop" }], {
        rollback: destructor.NOOP,
        reapply: () => destructor.NOOP,
      });
      expect(send.mock.calls.map(([key, actions]) => [key, actions])).toEqual([
        ["missing", [{ type: "noop" }]],
      ]);
      expect(controller.hasPending("missing")).toBe(false);
    });

    it("rolls back every dispatch in a failed batch, newest first", async () => {
      const gate = Promise.withResolvers<void>();
      const send = vi.fn<SendDispatch<string, Action>>(async () => {
        if (send.mock.calls.length === 1) {
          await gate.promise;
          return [];
        }
        throw new Error("rejected");
      });
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("b", 2)]);
      const three = controller.dispatch("k", [set("c", 3)]);
      gate.resolve();
      expect(await one).toBe(true);
      await expect(two).rejects.toThrow("rejected");
      await expect(three).rejects.toThrow("rejected");
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
      expect(controller.prepareUndo("k")?.actions).toEqual([set("a", 0)]);
    });

    it("sends undo and redo through the queue in program order", async () => {
      const send = vi.fn<SendDispatch<string, Action>>(async () => []);
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      const gate = Promise.withResolvers<void>();
      send.mockImplementationOnce(async () => {
        await gate.promise;
        return [];
      });
      const undone = controller.undo("k");
      const redone = controller.redo("k");
      const again = controller.dispatch("k", [set("b", 2)]);
      expect(send).toHaveBeenCalledTimes(2);
      gate.resolve();
      expect(await Promise.all([undone, redone, again])).toEqual([true, true, true]);
      expect(send.mock.calls.map(([, actions]) => actions)).toEqual([
        [set("a", 1)],
        [set("a", 0)],
        [set("a", 1), set("b", 2)],
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 2 } });
    });

    it("sends actions the caller applied after queued dispatches", async () => {
      const { gate, send } = createGatedSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("a", 2)]);
      const tagged = controller.send("k", [{ type: "tag", key: "a", tag: "x" }], {
        rollback: destructor.NOOP,
        reapply: () => destructor.NOOP,
      });
      gate.resolve();
      await Promise.all([one, two, tagged]);
      expect(send.mock.calls[1][1]).toEqual([
        set("a", 2),
        { type: "tag", key: "a", tag: "x" },
      ]);
    });

    it("keeps a queued transaction on top of a foreign frame", async () => {
      const { gate, send } = createGatedSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const tx = controller.transaction("k");
      tx.add(set("a", 2));
      const committed = tx.commit();
      controller.applyRemote("k", 1, "foreign-1", [set("a", 5)]);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      gate.resolve();
      await one;
      expect(await committed).toBe(true);
      const [[, first, firstKey], [, actions, dispatchKey]] = send.mock.calls;
      controller.applyRemote("k", 2, firstKey, first);
      controller.applyRemote("k", 3, dispatchKey, actions);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      expect(controller.hasPending("k")).toBe(false);
    });

    it("sends a transaction committed during an in-flight send with the batch", async () => {
      const { gate, send } = createGatedSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0, b: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const two = controller.dispatch("k", [set("b", 2)]);
      const tx = controller.transaction("k");
      tx.add(set("a", 3));
      const committed = tx.commit();
      gate.resolve();
      expect(await Promise.all([one, two, committed])).toEqual([true, true, true]);
      expect(send.mock.calls.map(([, actions]) => actions)).toEqual([
        [set("a", 1)],
        [set("b", 2), set("a", 3)],
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 3, b: 2 } });
    });
  });

  describe("transaction", () => {
    it("accumulates and commits as a single undoable", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k", "move");
      tx.add({ type: "set", key: "a", value: 1 });
      tx.add([{ type: "set", key: "a", value: 2 }]);
      const ok = await tx.commit();
      expect(ok).toBe(true);
      expect(send).toHaveBeenCalledTimes(1);
      expect(send.mock.calls[0][1]).toHaveLength(2);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      const r = controller.prepareUndo("k");
      // Inverse computed against the pre-transaction snapshot (a:0), not
      // the last add's pre-state (a:1). Latest inverse undoes first.
      expect(r?.actions).toEqual([
        { type: "set", key: "a", value: 1 },
        { type: "set", key: "a", value: 0 },
      ]);
    });

    it("returns false from commit when no actions were added", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const ok = await controller.transaction("k").commit();
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("returns false from commit when the doc is not cached", async () => {
      const { controller, send } = setupStore();
      const tx = controller.transaction("missing");
      tx.add({ type: "set", key: "a", value: 1 });
      const ok = await tx.commit();
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("subsequent commits return false and do not re-send", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add({ type: "set", key: "a", value: 1 });
      expect(await tx.commit()).toBe(true);
      expect(await tx.commit()).toBe(false);
      expect(send).toHaveBeenCalledTimes(1);
    });

    it("add throws after the transaction is finalized", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add({ type: "set", key: "a", value: 1 });
      await tx.commit();
      expect(() => tx.add({ type: "set", key: "a", value: 2 })).toThrow();
    });

    it("leaves an add that the doc cannot hold out of the commit", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add(set("a", 1));
      expect(() => tx.add({ type: "invalid" })).toThrow("cannot hold");
      expect(await tx.commit()).toBe(true);
      expect(send).toHaveBeenCalledWith("k", [set("a", 1)], expect.any(String));
    });

    it("abort restores the pre-transaction snapshot and pushes nothing", () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add({ type: "set", key: "a", value: 1 });
      tx.add({ type: "set", key: "a", value: 2 });
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      tx.abort();
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
      expect(send).not.toHaveBeenCalled();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("abort is a no-op after commit", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add({ type: "set", key: "a", value: 5 });
      await tx.commit();
      tx.abort();
      // State stays at the committed value.
      expect(docs.get("k")).toEqual({ values: { a: 5 } });
    });

    it("rolls back doc and stack when send fails", async () => {
      const { docs, controller } = setupStore({
        send: async () => {
          throw new Error("boom");
        },
      });
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add({ type: "set", key: "a", value: 1 });
      await expect(tx.commit()).rejects.toThrow("boom");
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("keeps staged actions on top of a foreign frame that lands before the commit", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0, b: 0 });
      const tx = controller.transaction("k");
      tx.add(set("a", 1));
      controller.applyRemote("k", 1, "foreign", [set("a", 5), set("b", 5)]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 5 } });
      tx.abort();
      expect(docs.get("k")).toEqual({ values: { a: 5, b: 5 } });
    });

    it("keeps an add to an earlier transaction on top when a foreign frame lands", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0, b: 0 });
      const first = controller.transaction("k");
      const second = controller.transaction("k");
      first.add(set("a", 1));
      second.add(set("a", 2));
      first.add(set("a", 3));
      expect(docs.get("k")).toEqual({ values: { a: 3, b: 0 } });
      controller.applyRemote("k", 1, "foreign", [set("b", 5)]);
      expect(docs.get("k")).toEqual({ values: { a: 3, b: 5 } });
    });

    it("matches the Core when the echoes of two transactions on a field arrive", async () => {
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const first = controller.transaction("k");
      const second = controller.transaction("k");
      first.add(set("a", 1));
      second.add(set("a", 2));
      first.add(set("a", 3));
      const committed = [first.commit(), second.commit()];
      // The Core applied the first transaction, then the second.
      controller.applyRemote("k", 1, calls[0].dispatchKey, calls[0].actions);
      calls[0].resolve();
      await expect.poll(() => calls).toHaveLength(2);
      controller.applyRemote("k", 2, calls[1].dispatchKey, calls[1].actions);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      calls[1].resolve();
      await Promise.all(committed);
      expect(controller.hasPending("k")).toBe(false);
    });

    it("sends the staged actions under the dispatch key it held them pending with", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add(set("a", 1));
      expect(controller.hasPending("k")).toBe(true);
      await tx.commit();
      const dispatchKey = send.mock.calls[0][2];
      controller.applyRemote("k", 1, dispatchKey, [set("a", 1)]);
      expect(controller.hasPending("k")).toBe(false);
      expect(controller.prepareUndo("k")).not.toBeNull();
    });
  });

  describe("encode", () => {
    /** Lowers the key of a set, and records the values of each doc it runs against. */
    const createEncode = () =>
      vi.fn((_: Doc, a: Action): Action =>
        a.type === "set" ? { ...a, key: a.key.toLowerCase() } : a,
      );

    const valuesSeen = (encode: ReturnType<typeof createEncode>) =>
      encode.mock.calls.map(([doc]) => doc.values);

    it("encodes each dispatched action against the doc the actions before it produce", async () => {
      const encode = createEncode();
      const { docs, controller, send } = setupStore({ encode });
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("A", 1), set("B", 2)]);
      expect(send).toHaveBeenCalledWith(
        "k",
        [set("a", 1), set("b", 2)],
        expect.any(String),
      );
      expect(valuesSeen(encode)).toEqual([{ a: 0 }, { a: 1 }]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 2 } });
    });

    it("encodes the actions that the preprocess option returns", async () => {
      const { docs, controller, send } = setupStore({ encode: createEncode() });
      prime(docs, "k", { a: 3 });
      await controller.dispatch("k", [set("A", 4)], {
        preprocess: (_, acts) => [...acts, set("B", 5)],
      });
      expect(send).toHaveBeenCalledWith(
        "k",
        [set("a", 4), set("b", 5)],
        expect.any(String),
      );
    });

    it("encodes each add against the doc the adds before it produce", async () => {
      const encode = createEncode();
      const { docs, controller, send } = setupStore({ encode });
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add(set("A", 1));
      tx.add([set("B", 2), set("A", 3)]);
      await tx.commit();
      expect(send).toHaveBeenCalledWith(
        "k",
        [set("a", 1), set("b", 2), set("a", 3)],
        expect.any(String),
      );
      expect(valuesSeen(encode)).toEqual([{ a: 0 }, { a: 1 }, { a: 1, b: 2 }]);
    });

    it("sends undo and redo actions as they are stored", async () => {
      const encode = createEncode();
      const { docs, controller, send } = setupStore({ encode });
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("A", 1)]);
      await controller.undo("k");
      await controller.redo("k");
      expect(encode).toHaveBeenCalledTimes(1);
      expect(send.mock.calls.map(([, acts]) => acts)).toEqual([
        [set("a", 1)],
        [set("a", 0)],
        [set("a", 1)],
      ]);
    });
  });

  describe("settled", () => {
    /** @returns a probe that reads whether the settled promise resolved. */
    const watch = (promise: Promise<void>) => {
      let done = false;
      void promise.then(() => {
        done = true;
      });
      return async () => {
        await new Promise((resolve) => setTimeout(resolve, 0));
        return done;
      };
    };

    it("resolves at once when nothing is pending", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      expect(await watch(controller.settled("k"))()).toBe(true);
    });

    it("resolves when the echo of the last pending dispatch arrives", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      await controller.dispatch("k", [set("a", 2)]);
      const done = watch(controller.settled("k"));
      controller.applyRemote("k", 1, send.mock.calls[0][2], [set("a", 1)]);
      expect(await done()).toBe(false);
      controller.applyRemote("k", 2, send.mock.calls[1][2], [set("a", 2)]);
      expect(await done()).toBe(true);
    });

    it("resolves when the Core refuses the pending dispatch", async () => {
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      const done = watch(controller.settled("k"));
      calls[0].reject(new Error("boom"));
      await expect(dispatched).rejects.toThrow("boom");
      expect(await done()).toBe(true);
    });

    it("waits for an open transaction until it aborts", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add(set("a", 1));
      const done = watch(controller.settled("k"));
      expect(await done()).toBe(false);
      tx.abort();
      expect(await done()).toBe(true);
    });

    it("resolves when the cache resets", async () => {
      const { docs, epochs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      const done = watch(controller.settled("k"));
      epochs.notify(0);
      expect(await done()).toBe(true);
    });

    it("resolves when the doc is deleted", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      const done = watch(controller.settled("k"));
      docs.delete("k");
      expect(await done()).toBe(true);
    });
  });

  describe("applyRemote", () => {
    it("applies foreign actions and stamps targets as remote-touched", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Push a local entry first so we can verify it gets marked stale.
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.applyRemote("k", 1, "foreign-1", [
        { type: "set", key: "a", value: 99 },
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 99 } });
      expect(controller.prepareUndo("k")).toBeNull();
    });

    it("is a no-op when the doc is not cached", () => {
      const { docs, controller } = setupStore();
      controller.applyRemote("missing", 1, "foreign-1", [
        { type: "set", key: "a", value: 1 },
      ]);
      expect(docs.get("missing")).toBeUndefined();
    });

    it("ingests a create-headed frame for a doc it has never cached", () => {
      const { docs, controller } = setupStore();
      controller.applyRemote("fresh", 1, "", [
        { type: "create", doc: { values: { a: 1 } } },
        { type: "set", key: "b", value: 2 },
      ]);
      expect(docs.get("fresh")).toEqual({ values: { a: 1, b: 2 } });
    });

    it("applies a foreign create to a cached doc", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.applyRemote("k", 1, "", [
        { type: "create", doc: { values: { a: 9 } } },
        set("b", 2),
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 9, b: 2 } });
    });

    it("orders dispatch frames after the create that seeded the doc", () => {
      const { docs, controller } = setupStore();
      controller.applyRemote("fresh", 1, "", [
        { type: "create", doc: { values: { a: 1 } } },
      ]);
      controller.applyRemote("fresh", 2, "f-1", [{ type: "set", key: "a", value: 5 }]);
      expect(docs.get("fresh")).toEqual({ values: { a: 5 } });
    });

    it("drops echoes whose seq does not exceed the high-water mark", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.applyRemote("k", 5, "f-1", [{ type: "set", key: "a", value: 1 }]);
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
      // Same seq — a duplicate frame — must be dropped.
      controller.applyRemote("k", 5, "f-1", [{ type: "set", key: "a", value: 99 }]);
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
      // Older seq — a reordered or replayed frame — must also be dropped.
      controller.applyRemote("k", 4, "f-2", [{ type: "set", key: "a", value: 99 }]);
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
      // Fresher seq applies and advances the high-water mark.
      controller.applyRemote("k", 6, "f-3", [{ type: "set", key: "a", value: 2 }]);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
    });

    it("treats seq=0 as unstamped and always applies for legacy-server compat", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.applyRemote("k", 0, "f-1", [{ type: "set", key: "a", value: 1 }]);
      controller.applyRemote("k", 0, "f-2", [{ type: "set", key: "a", value: 2 }]);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
    });

    it("leaves the doc untouched when the echo of the first pending dispatch arrives", async () => {
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      const before = docs.get("k");
      controller.applyRemote("k", 1, calls[0].dispatchKey, [set("a", 1)]);
      expect(docs.get("k")).toBe(before);
      calls[0].resolve();
      await dispatched;
      expect(controller.hasPending("k")).toBe(false);
    });

    it("applies the actions that the Core adds to the echo of a dispatch", async () => {
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      controller.applyRemote("k", 1, calls[0].dispatchKey, [set("a", 1), set("b", 2)]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 2 } });
      calls[0].resolve();
      await dispatched;
      expect(controller.hasPending("k")).toBe(false);
    });

    it("does not mark targets remote-touched on own echoes", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      const dispatchKey = send.mock.calls[0][2];
      controller.applyRemote("k", 1, dispatchKey, [set("a", 1)]);
      expect(controller.prepareUndo("k")).not.toBeNull();
    });

    it("applies a foreign frame under the pending dispatches", async () => {
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0, b: 0 });
      void controller.dispatch("k", [set("a", 10)]);
      void controller.dispatch("k", [set("a", 20)]);
      expect(docs.get("k")).toEqual({ values: { a: 20, b: 0 } });
      controller.applyRemote("k", 1, calls[0].dispatchKey, [set("a", 10)]);
      expect(docs.get("k")).toEqual({ values: { a: 20, b: 0 } });
      controller.applyRemote("k", 2, "bob", [set("a", 5), set("b", 5)]);
      expect(docs.get("k")).toEqual({ values: { a: 20, b: 5 } });
      calls[0].resolve();
      await expect.poll(() => calls).toHaveLength(2);
      controller.applyRemote("k", 3, calls[1].dispatchKey, [set("a", 20)]);
      expect(docs.get("k")).toEqual({ values: { a: 20, b: 5 } });
      expect(controller.hasPending("k")).toBe(false);
    });

    it("applies an own echo in the order the Core applied it", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add(set("a", 1));
      await controller.dispatch("k", [set("a", 2)]);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
      // The Core applied the dispatch before the transaction that commits after it.
      controller.applyRemote("k", 1, send.mock.calls[0][2], [set("a", 2)]);
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
      await tx.commit();
      controller.applyRemote("k", 2, send.mock.calls[1][2], [set("a", 1)]);
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
      expect(controller.hasPending("k")).toBe(false);
    });

    it("keeps a pending edit when another writer replaces the doc before its echo", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0, b: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      // A fetch that read the doc before the Core applied the dispatch.
      docs.set("k", { values: { a: 0, b: 3 } });
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 3 } });
      const dispatchKey = send.mock.calls[0][2];
      controller.applyRemote("k", 1, dispatchKey, [set("a", 1)]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 3 } });
    });

    it("notifies only the doc with the pending dispatches when another writer replaces it", () => {
      const { send } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0, b: 0 });
      void controller.dispatch("k", [set("a", 1)]);
      const seen: Doc[] = [];
      docs.subscribe((e) => {
        if (e.variant === "set") seen.push(e.value);
      }, "k");
      docs.set("k", { values: { a: 0, b: 3 } });
      expect(seen).toEqual([{ values: { a: 1, b: 3 } }]);
    });

    it("notifies nothing when another writer replaces a doc with what it holds", () => {
      const { send } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      void controller.dispatch("k", [set("a", 1)]);
      const listener = vi.fn();
      docs.subscribe(listener, "k");
      docs.set("k", { values: { a: 0 } });
      expect(listener).not.toHaveBeenCalled();
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
    });

    it("stores a doc another writer puts in the table as is once nothing is pending", async () => {
      const { docs, controller, send } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      controller.applyRemote("k", 1, send.mock.calls[0][2], [set("a", 1)]);
      docs.set("k", { values: { a: 5 } });
      expect(docs.get("k")).toEqual({ values: { a: 5 } });
    });

    it("keeps a pending edit out of the base when another writer sets the doc", async () => {
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0, b: 0 });
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      docs.set("k", (prev) =>
        prev == null ? prev : { values: { ...prev.values, b: 3 } },
      );
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 3 } });
      calls[0].reject(new Error("boom"));
      await expect(dispatched).rejects.toThrow("boom");
      expect(docs.get("k")).toEqual({ values: { a: 0, b: 3 } });
    });

    it("restores the base when another writer's set that shows no change rolls back", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add(set("a", 1));
      const rollback = docs.set("k", { values: { a: 2 } });
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
      rollback();
      tx.abort();
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
    });

    it("shows an edit the Core saved while a failed delete hid the doc", async () => {
      let stored = { key: "k", values: { a: 0 } };
      const { send, calls } = createHeldSend();
      const { docs, controller } = setupStore({ fetch: async () => [stored], send });
      prime(docs, "k", { a: 0 });
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      const restore = docs.delete("k");
      stored = { key: "k", values: { a: 1 } };
      calls[0].resolve();
      await dispatched;
      controller.applyRemote("k", 1, calls[0].dispatchKey, [set("a", 1)]);
      restore();
      await expect.poll(() => docs.get("k")).toEqual(stored);
    });

    it("applies pending dispatches again on a doc another writer replaced", async () => {
      const { send } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      void controller.dispatch("k", [set("a", 1)]);
      docs.set("k", { values: { a: 0, b: 4 } });
      void controller.dispatch("k", [set("c", 2)]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 4, c: 2 } });
    });

    it("applies a frame whose dispatch key is not pending as foreign", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.applyRemote("k", 1, "stranger", [set("a", 7)]);
      expect(docs.get("k")).toEqual({ values: { a: 7 } });
    });
  });

  describe("epochs", () => {
    it("settles accepted dispatches whose echoes the gap may have lost", async () => {
      const { docs, epochs, controller } = setupStore({
        send: async () => [set("h", 9)],
      });
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      epochs.notify(2);
      expect(controller.hasPending("k")).toBe(false);
      expect(docs.get("k")).toEqual({ values: { a: 1, h: 9 } });
      controller.applyRemote("k", 1, "foreign", [set("b", 2)]);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 2, h: 9 } });
    });

    it("settles a dispatch sent before the epoch changed when its send resolves", async () => {
      const { send, calls } = createHeldSend();
      const { docs, epochs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      epochs.notify(2);
      expect(controller.hasPending("k")).toBe(true);
      calls[0].resolve([set("h", 9)]);
      await dispatched;
      expect(controller.hasPending("k")).toBe(false);
      expect(docs.get("k")).toEqual({ values: { a: 1, h: 9 } });
    });

    it("keeps a staged transaction pending across an epoch change", () => {
      const { docs, epochs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add(set("a", 1));
      epochs.notify(2);
      expect(controller.hasPending("k")).toBe(true);
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
    });

    it("clears the sequence high-water mark", () => {
      const { docs, epochs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.applyRemote("k", 5, "foreign", [set("a", 1)]);
      epochs.notify(2);
      controller.applyRemote("k", 1, "foreign", [set("a", 2)]);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
    });

    it("clears pending dispatches and undo stacks when the cache resets", async () => {
      const { docs, epochs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      expect(controller.hasUndo("k")).toBe(true);
      epochs.notify(0);
      expect(controller.hasUndo("k")).toBe(false);
      expect(controller.hasPending("k")).toBe(false);
    });
  });

  describe("create", () => {
    it("holds the create pending so its echo does not revert later edits", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      let dispatchKey = "";
      const res = await controller.create(["k"], async (key) => {
        dispatchKey = key;
        return "created";
      });
      expect(res).toBe("created");
      await controller.dispatch("k", [set("a", 5)]);
      controller.applyRemote("k", 1, dispatchKey, [
        { type: "create", doc: { values: { a: 0 } } },
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 5 } });
    });

    it("removes the pending create and rethrows when the send fails", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      await expect(
        controller.create(["k"], async () => {
          throw new Error("boom");
        }),
      ).rejects.toThrow("boom");
      expect(controller.hasPending("k")).toBe(false);
    });

    it("skips keys whose doc is not cached", async () => {
      const { controller } = setupStore();
      const create = vi.fn(async () => "created");
      expect(await controller.create(["missing"], create)).toBe("created");
      expect(create).toHaveBeenCalledTimes(1);
      expect(controller.hasPending("missing")).toBe(false);
    });
  });

  describe("markRemoteTouched", () => {
    it("is a no-op for empty targets", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k");
      // No throw, no state change observable to a fresh prepareUndo.
      controller.markRemoteTouched("k", []);
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("orders touches against entries by call order, not the clock", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Touch, then entry, with no delay between them: the entry is live.
      controller.markRemoteTouched("k", ["a"]);
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(controller.prepareUndo("k")).not.toBeNull();
      // Touch again immediately after: the entry is stale.
      controller.markRemoteTouched("k", ["a"]);
      expect(controller.prepareUndo("k")).toBeNull();
    });
  });

  describe("delete (cascade)", () => {
    it("drops the doc and its undo state for a single key", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      docs.delete("k");
      expect(docs.get("k")).toBeUndefined();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("drops both for an array of keys", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k1");
      prime(docs, "k2");
      controller.recordEntry(
        "k1",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.recordEntry(
        "k2",
        [{ type: "set", key: "a", value: 2 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      docs.delete(["k1", "k2"]);
      expect(docs.has("k1")).toBe(false);
      expect(docs.has("k2")).toBe(false);
      expect(controller.hasUndo("k1")).toBe(false);
      expect(controller.hasUndo("k2")).toBe(false);
    });

    it("drops pending dispatches with the doc", () => {
      const { send } = createHeldSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      void controller.dispatch("k", [set("a", 1)]);
      docs.delete("k");
      expect(controller.hasPending("k")).toBe(false);
    });

    it("drops both for a predicate match", () => {
      const { docs, controller } = setupStore();
      prime(docs, "keep", { a: 0 });
      prime(docs, "drop", { a: 0 });
      controller.recordEntry(
        "drop",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      docs.delete((_v, k) => k === "drop");
      expect(docs.get("drop")).toBeUndefined();
      expect(docs.get("keep")).toBeDefined();
      expect(controller.hasUndo("drop")).toBe(false);
    });
  });

  describe("hasUndo / hasRedo", () => {
    it("is false initially and tracks the undo and redo stacks", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      expect(controller.hasUndo("k")).toBe(false);
      expect(controller.hasRedo("k")).toBe(false);
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(controller.hasUndo("k")).toBe(true);
      expect(controller.hasRedo("k")).toBe(false);
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(false);
      expect(controller.hasRedo("k")).toBe(true);
      controller.prepareRedo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(true);
      expect(controller.hasRedo("k")).toBe(false);
    });
  });

  describe("onUndoStateChange", () => {
    it("fires on entry record and on cascade delete", () => {
      const { docs, controller } = setupStore();
      const cb = vi.fn();
      controller.onUndoStateChange(cb, "k");
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(cb).toHaveBeenCalled();
      cb.mockClear();
      docs.delete("k");
      expect(cb).toHaveBeenCalled();
    });

    it("notifies on every stack mutation, even when entries coalesce", () => {
      const { docs, controller } = setupStore({
        coalesceWindow: TimeSpan.SECOND,
      });
      const cb = vi.fn();
      controller.onUndoStateChange(cb, "k");
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 2 }],
        [{ type: "set", key: "a", value: 1 }],
        ["a"],
      );
      expect(cb).toHaveBeenCalledTimes(2);
    });

    it("stops firing after the destructor is called", () => {
      const { docs, controller } = setupStore();
      const cb = vi.fn();
      const off = controller.onUndoStateChange(cb, "k");
      prime(docs, "k", { a: 0 });
      off();
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(cb).not.toHaveBeenCalled();
    });
  });

  describe("listener", () => {
    it("applies broadcast frames from the channel to the controller", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 1 });
      const listener = controller.listener("docs_dispatch", frameZ);
      expect(listener.channel).toBe("docs_dispatch");
      void listener.onChange([
        {
          key: "k",
          dispatchKey: "remote-1",
          seq: 1,
          actions: [{ type: "set", key: "a", value: 42 }],
        },
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 42 } });
    });
  });

  describe("defaults", () => {
    it("treats every action as undoable when isUndoable is omitted", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k");
      controller.recordEntry("k", [{ type: "noop" }], [], ["a"]);
      expect(controller.hasUndo("k")).toBe(true);
    });

    it("uses 'default' as the kind when kindOf is omitted", () => {
      const { docs, controller } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(docs, "k", { a: 0 });
      // Two entries with different action types — both fall under the same
      // default kind and should coalesce when targets+window match.
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 2 }],
        [{ type: "set", key: "a", value: 1 }],
        ["a"],
      );
      // Coalesced into one entry.
      controller.prepareUndo("k")?.commit();
      expect(controller.hasUndo("k")).toBe(false);
    });
  });
});
