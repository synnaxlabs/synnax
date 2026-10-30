// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor, TimeSpan, TimeStamp } from "@synnaxlabs/x";
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
    preprocess: (s: Doc, acts: Action[]) => Action[];
    now: () => TimeStamp;
    send: SendDispatch<string, Action>;
  }> = {},
) => {
  const send = vi.fn<SendDispatch<string, Action>>(opts.send ?? (async () => {}));
  const errors: Error[] = [];
  const onError = (error: Error) => errors.push(error);
  const docs = new Table<string, Doc>({ onError });
  const controller = new Controller<string, Doc, Action>({
    store: docs,
    onError,
    reduce: reducer,
    send,
    isUndoable: opts.isUndoable,
    kindOf: opts.kindOf,
    createOf: (a) => (a.type === "create" ? a.doc : undefined),
    coalesceWindow: opts.coalesceWindow,
    stackCap: opts.stackCap,
    preprocess: opts.preprocess,
    now: opts.now,
  });
  return { errors, docs, controller, send };
};

const prime = (
  docs: Table<string, Doc>,
  key: string,
  values: Record<string, number> = {},
) => docs.set(key, { values });

describe("actions.Controller", () => {
  describe("replay", () => {
    it("returns null when the doc is not cached", () => {
      const { controller } = setupStore();
      expect(
        controller.replay("missing", [{ type: "set", key: "a", value: 1 }]),
      ).toBeNull();
    });

    it("applies the reducer to the cached doc and returns processed/inverse/targets", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 5 });
      const r = controller.replay("k", [{ type: "set", key: "a", value: 9 }]);
      expect(r).not.toBeNull();
      expect(r?.processed).toEqual([{ type: "set", key: "a", value: 9 }]);
      expect(r?.inverse).toEqual([{ type: "set", key: "a", value: 5 }]);
      expect(r?.targets).toEqual(["a"]);
      expect(r?.changed).toBe(true);
      expect(docs.get("k")).toEqual({ values: { a: 9 } });
    });

    it("reports changed=false when no action touches the state", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 5 });
      const r = controller.replay("k", [{ type: "noop" }]);
      expect(r).not.toBeNull();
      expect(r?.changed).toBe(false);
    });

    it("rollback restores the prior doc state", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 1 });
      const r = controller.replay("k", [{ type: "set", key: "a", value: 99 }]);
      expect(docs.get("k")).toEqual({ values: { a: 99 } });
      r?.rollback();
      expect(docs.get("k")).toEqual({ values: { a: 1 } });
    });

    it("runs preprocess by default and skips it under skipPreprocess", () => {
      const preprocess = vi.fn((_: Doc, acts: Action[]): Action[] => [
        ...acts,
        { type: "set", key: "extra", value: 1 },
      ]);
      const { docs, controller } = setupStore({ preprocess });
      prime(docs, "k");
      const r1 = controller.replay("k", [{ type: "set", key: "a", value: 2 }]);
      expect(r1?.processed).toHaveLength(2);
      expect(preprocess).toHaveBeenCalledTimes(1);
      preprocess.mockClear();
      const r2 = controller.replay("k", [{ type: "set", key: "a", value: 3 }], {
        skipPreprocess: true,
      });
      expect(r2?.processed).toHaveLength(1);
      expect(preprocess).not.toHaveBeenCalled();
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

    it("undo → redo → undo cycle restores state correctly at each step", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Manually drive the cycle (no substrate): apply forward, then walk
      // through prepareUndo / prepareRedo applying their actions so the
      // doc state mirrors what the substrate would do via replay.
      const entry = {
        forward: [{ type: "set" as const, key: "a", value: 1 }],
        inverse: [{ type: "set" as const, key: "a", value: 0 }],
      };
      controller.replay("k", entry.forward);
      controller.recordEntry("k", entry.forward, entry.inverse, ["a"]);
      expect(docs.get("k")?.values.a).toBe(1);
      const undo = controller.prepareUndo("k");
      controller.replay("k", undo!.actions, { skipPreprocess: true });
      undo!.commit();
      expect(docs.get("k")?.values.a).toBe(0);
      const redo = controller.prepareRedo("k");
      controller.replay("k", redo!.actions, { skipPreprocess: true });
      redo!.commit();
      expect(docs.get("k")?.values.a).toBe(1);
      const undo2 = controller.prepareUndo("k");
      controller.replay("k", undo2!.actions, { skipPreprocess: true });
      undo2!.commit();
      expect(docs.get("k")?.values.a).toBe(0);
    });
  });

  describe("send queue", () => {
    const set = (key: string, value: number): Action => ({ type: "set", key, value });

    // A send that waits on the returned gate before its first call settles.
    const createGatedSend = () => {
      const gate = Promise.withResolvers<void>();
      const send = vi.fn<SendDispatch<string, Action>>(async () => {
        if (send.mock.calls.length === 1) await gate.promise;
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

    it("skips the reduce for an undisturbed echo of a batch", async () => {
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
      expect(controller.hasOutstanding("k")).toBe(false);
      expect(docs.get("k")).toEqual({ values: { a: 1, b: 2, c: 3 } });
      expect(controller.hasUndo("k")).toBe(true);
    });

    // Fails the sends whose call numbers are listed, counting from 1.
    const createFailingSend = (...failing: number[]) => {
      const send = vi.fn<SendDispatch<string, Action>>(async () => {
        if (failing.includes(send.mock.calls.length)) throw new Error("rejected");
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

    it("rolls back every dispatch in a failed batch, newest first", async () => {
      const gate = Promise.withResolvers<void>();
      const send = vi.fn<SendDispatch<string, Action>>(async () => {
        if (send.mock.calls.length === 1) await gate.promise;
        else throw new Error("rejected");
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
      const send = vi.fn<SendDispatch<string, Action>>(async () => {});
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      await controller.dispatch("k", [set("a", 1)]);
      const gate = Promise.withResolvers<void>();
      send.mockImplementationOnce(async () => await gate.promise);
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

    it("recovers a queued transaction from a foreign frame", async () => {
      const { gate, send } = createGatedSend();
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const one = controller.dispatch("k", [set("a", 1)]);
      const tx = controller.transaction("k");
      tx.add(set("a", 2));
      const committed = tx.commit();
      controller.applyRemote("k", 1, "foreign-1", [set("a", 5)]);
      expect(docs.get("k")).toEqual({ values: { a: 5 } });
      gate.resolve();
      await one;
      expect(await committed).toBe(true);
      const [, actions, dispatchKey] = send.mock.calls[1];
      controller.applyRemote("k", 2, dispatchKey, actions);
      expect(docs.get("k")).toEqual({ values: { a: 2 } });
    });
  });

  describe("create", () => {
    const set = (key: string, value: number): Action => ({ type: "set", key, value });

    it("sends a dispatch made during the create after the create", async () => {
      const { docs, controller, send } = setupStore();
      const created = Promise.withResolvers<void>();
      const sending = Promise.withResolvers<void>();
      const done = controller.create([{ key: "k", values: { a: 0 } }], {
        send: async () => {
          sending.resolve();
          await created.promise;
        },
      });
      await sending.promise;
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      expect(docs.get("k")?.values).toEqual({ a: 1 });
      expect(send).not.toHaveBeenCalled();
      created.resolve();
      await done;
      expect(await dispatched).toBe(true);
      expect(send.mock.calls[0][1]).toEqual([set("a", 1)]);
    });

    it("rejects a dispatch made during a failed create and uncaches the doc", async () => {
      const { docs, controller, send } = setupStore();
      const created = Promise.withResolvers<void>();
      const sending = Promise.withResolvers<void>();
      const done = controller.create([{ key: "k", values: { a: 0 } }], {
        send: async () => {
          sending.resolve();
          await created.promise;
        },
      });
      await sending.promise;
      const dispatched = controller.dispatch("k", [set("a", 1)]);
      created.reject(new Error("create failed"));
      await expect(done).rejects.toThrow("create failed");
      await expect(dispatched).rejects.toThrow("create failed");
      expect(send).not.toHaveBeenCalled();
      expect(docs.get("k")).toBeUndefined();
      expect(controller.hasUndo("k")).toBe(false);
    });
    it("rolls back an edit made before a failed create over a cached doc", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const done = controller.create([{ key: "k", values: { a: 5 } }], {
        send: async () => {
          throw new Error("create failed");
        },
      });
      expect(await controller.dispatch("k", [set("a", 1)])).toBe(true);
      await expect(done).rejects.toThrow("create failed");
      expect(docs.get("k")?.values).toEqual({ a: 0 });
      expect(await controller.undo("k")).toBe(false);
      expect(docs.get("k")?.values).toEqual({ a: 0 });
    });

    it("carries a dispatch made inside onCached in the create", async () => {
      const { docs, controller, send } = setupStore();
      const create = vi.fn(async (_: Doc[]) => {});
      await controller.create([{ key: "k", values: { a: 0 } }], {
        onCached: async () => {
          expect(await controller.dispatch("k", [set("a", 1)])).toBe(true);
        },
        send: create,
      });
      expect(create.mock.calls[0][0].map(({ values }) => values)).toEqual([{ a: 1 }]);
      expect(send).not.toHaveBeenCalled();
      expect(docs.get("k")?.values).toEqual({ a: 1 });
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
      const send = vi.fn<SendDispatch<string, Action>>(async () => {
        throw new Error("boom");
      });
      const { docs, controller } = setupStore({ send });
      prime(docs, "k", { a: 0 });
      const tx = controller.transaction("k");
      tx.add({ type: "set", key: "a", value: 1 });
      await expect(tx.commit()).rejects.toThrow("boom");
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
      expect(controller.hasUndo("k")).toBe(false);
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

    it("drops a create frame for a doc it already caches", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Locally edited after creation; the late create echo must not revert.
      controller.replay("k", [{ type: "set", key: "a", value: 7 }]);
      controller.applyRemote("k", 1, "", [
        { type: "create", doc: { values: { a: 0 } } },
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 7 } });
    });

    it("applies trailing actions after dropping the create for a cached doc", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.applyRemote("k", 1, "", [
        { type: "create", doc: { values: { a: 9 } } },
        { type: "set", key: "b", value: 2 },
      ]);
      expect(docs.get("k")).toEqual({ values: { a: 0, b: 2 } });
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

    it("skips the reducer on own echoes when no foreign action interleaved", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Originator simulates local replay then registers the dispatch as
      // outstanding before the echo arrives.
      controller.registerOutstandingDispatch("k", "own-1");
      // Now the echo lands. The reducer should NOT re-run: the local replay
      // was already authoritative.
      controller.applyRemote("k", 1, "own-1", [{ type: "set", key: "a", value: 99 }]);
      // State unchanged — the local replay's effect (a=0 here, since the test
      // skipped the replay step) is preserved.
      expect(docs.get("k")).toEqual({ values: { a: 0 } });
    });

    it("does not mark targets remote-touched on own echoes", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      controller.registerOutstandingDispatch("k", "own-1");
      controller.applyRemote("k", 1, "own-1", [{ type: "set", key: "a", value: 1 }]);
      // The local undo entry survives because the echo was own.
      expect(controller.prepareUndo("k")).not.toBeNull();
    });

    it("re-applies own echoes whose outstanding window was disturbed by a foreign action", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Alice locally replays both her dispatches up front and registers
      // them as outstanding. State after both local replays is a=20.
      controller.applyRemote("k", 0, "alice-1", [{ type: "set", key: "a", value: 10 }]);
      // Reset state to what it would be after local replays:
      prime(docs, "k", { a: 20 });
      controller.registerOutstandingDispatch("k", "alice-1");
      controller.registerOutstandingDispatch("k", "alice-2");
      // Now the echoes arrive in server-seq order.
      // Alice (own) seq=1 — own and not disturbed → skip.
      controller.applyRemote("k", 1, "alice-1", [{ type: "set", key: "a", value: 10 }]);
      expect(docs.get("k")).toEqual({ values: { a: 20 } });
      // Bob (foreign) seq=2 — applies, marks all outstanding own as disturbed.
      controller.applyRemote("k", 2, "bob-1", [{ type: "set", key: "a", value: 5 }]);
      expect(docs.get("k")).toEqual({ values: { a: 5 } });
      // Alice (own) seq=3 — own but disturbed → reduce to recover.
      controller.applyRemote("k", 3, "alice-2", [{ type: "set", key: "a", value: 20 }]);
      expect(docs.get("k")).toEqual({ values: { a: 20 } });
    });

    it("re-applies own echoes when their dispatchKey is unknown (registration lost the race against the echo)", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // No registration: simulate the case where the echo arrived before the dispatch
      // promise resolved, so the originator never got to register. The substrate treats
      // this as foreign and reduces — safer than silently dropping the action.
      controller.applyRemote("k", 1, "stranger", [{ type: "set", key: "a", value: 7 }]);
      expect(docs.get("k")).toEqual({ values: { a: 7 } });
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
