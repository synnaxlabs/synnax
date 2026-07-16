// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import z from "zod";

import { type cache } from "@/cache";
import { ScopedUnaryStore } from "@/cache/store";
import {
  Controller,
  type Frame,
  type Handle,
  type Reducer,
  type Send,
} from "@/dispatch/dispatch";

interface Doc {
  values: Record<string, number>;
}

type Action =
  | { type: "set"; key: string; value: number }
  | { type: "tag"; key: string; tag: string }
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

type CombinedStore = cache.UnaryStore<string, Doc> & Handle<string, Doc, Action>;

const setupStore = (
  opts: Partial<{
    isUndoable: (a: Action) => boolean;
    kindOf: (acts: Action[]) => string;
    coalesceWindow: TimeSpan;
    stackCap: number;
    preprocess: (s: Doc, acts: Action[]) => Action[];
  }> = {},
) => {
  const handleError = vi.fn((excOrFn: unknown) => {
    if (typeof excOrFn === "function") void excOrFn();
  });
  const docs = new ScopedUnaryStore<string, Doc>(handleError);
  const controller = new Controller<string, Doc, Action>({
    store: docs,
    handleError,
    reduce: reducer,
    isUndoable: opts.isUndoable,
    kindOf: opts.kindOf,
    coalesceWindow: opts.coalesceWindow,
    stackCap: opts.stackCap,
    preprocess: opts.preprocess,
  });
  const scope = (s: string): CombinedStore => ({
    ...(docs.scope(s) as cache.UnaryStore<string, Doc>),
    ...controller.scope(s),
  });
  return { handleError, scope, store: scope("primary"), controller };
};

const prime = (
  store: CombinedStore,
  key: string,
  values: Record<string, number> = {},
) => store.set(key, { values });

describe("dispatch.Controller", () => {
  describe("replay", () => {
    it("returns null when the doc is not cached", () => {
      const { store } = setupStore();
      expect(store.replay("missing", [{ type: "set", key: "a", value: 1 }])).toBeNull();
    });

    it("applies the reducer to the cached doc and returns processed/inverse/targets", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 5 });
      const r = store.replay("k", [{ type: "set", key: "a", value: 9 }]);
      expect(r).not.toBeNull();
      expect(r?.processed).toEqual([{ type: "set", key: "a", value: 9 }]);
      expect(r?.inverse).toEqual([{ type: "set", key: "a", value: 5 }]);
      expect(r?.targets).toEqual(["a"]);
      expect(r?.changed).toBe(true);
      expect(store.get("k")).toEqual({ values: { a: 9 } });
    });

    it("reports changed=false when no action touches the state", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 5 });
      const r = store.replay("k", [{ type: "noop" }]);
      expect(r).not.toBeNull();
      expect(r?.changed).toBe(false);
    });

    it("rollback restores the prior doc state", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 1 });
      const r = store.replay("k", [{ type: "set", key: "a", value: 99 }]);
      expect(store.get("k")).toEqual({ values: { a: 99 } });
      r?.rollback();
      expect(store.get("k")).toEqual({ values: { a: 1 } });
    });

    it("runs preprocess by default and skips it under skipPreprocess", () => {
      const preprocess = vi.fn((_: Doc, acts: Action[]): Action[] => [
        ...acts,
        { type: "set", key: "extra", value: 1 },
      ]);
      const { store } = setupStore({ preprocess });
      prime(store, "k");
      const r1 = store.replay("k", [{ type: "set", key: "a", value: 2 }]);
      expect(r1?.processed).toHaveLength(2);
      expect(preprocess).toHaveBeenCalledTimes(1);
      preprocess.mockClear();
      const r2 = store.replay("k", [{ type: "set", key: "a", value: 3 }], {
        skipPreprocess: true,
      });
      expect(r2?.processed).toHaveLength(1);
      expect(preprocess).not.toHaveBeenCalled();
    });
  });

  describe("recordEntry", () => {
    it("is a no-op when targets is empty", () => {
      const { store } = setupStore();
      prime(store, "k");
      store.recordEntry("k", [{ type: "set", key: "a", value: 1 }], [], []);
      expect(store.hasUndo("k")).toBe(false);
    });

    it("is a no-op when all forward actions are non-undoable", () => {
      const { store } = setupStore({ isUndoable: (a) => a.type !== "noop" });
      prime(store, "k");
      store.recordEntry("k", [{ type: "noop" }], [], ["a"]);
      expect(store.hasUndo("k")).toBe(false);
    });

    it("derives kind via kindOf", () => {
      const kindOf = vi.fn((acts: Action[]) => acts[0].type);
      const { store } = setupStore({ kindOf });
      prime(store, "k");
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(kindOf).toHaveBeenCalledTimes(1);
    });

    it("uses kindOverride when supplied without calling kindOf", () => {
      const kindOf = vi.fn(() => "auto");
      const { store } = setupStore({ kindOf });
      prime(store, "k");
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
        "manual",
      );
      expect(kindOf).not.toHaveBeenCalled();
    });

    it("clears the redo stack on a new entry", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      const undo = store.prepareUndo("k");
      undo?.commit();
      expect(store.hasRedo("k")).toBe(true);
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 2 }],
        [{ type: "set", key: "a", value: 1 }],
        ["a"],
      );
      expect(store.hasRedo("k")).toBe(false);
    });
  });

  describe("coalescing", () => {
    const push = (
      store: CombinedStore,
      key: string,
      target: string,
      value: number,
      kind: string,
    ) =>
      store.recordEntry(
        key,
        [{ type: "set", key: target, value }],
        [{ type: "set", key: target, value: value - 1 }],
        [target],
        kind,
      );

    it("merges entries with same kind and same targets within the window", () => {
      const { store } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(store, "k", { a: 0 });
      push(store, "k", "a", 1, "move");
      push(store, "k", "a", 2, "move");
      push(store, "k", "a", 3, "move");
      // One coalesced entry — single undo clears the stack.
      const r = store.prepareUndo("k");
      r?.commit();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("does not merge across kinds", () => {
      const { store } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(store, "k", { a: 0 });
      push(store, "k", "a", 1, "move");
      push(store, "k", "a", 2, "config");
      // Two entries — two undos required.
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(true);
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("does not merge when targets differ", () => {
      const { store } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(store, "k", { a: 0, b: 0 });
      push(store, "k", "a", 1, "move");
      push(store, "k", "b", 1, "move");
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(true);
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("does not merge when the prior entry is older than the window", async () => {
      const { store } = setupStore({ coalesceWindow: TimeSpan.NANOSECOND });
      prime(store, "k", { a: 0 });
      push(store, "k", "a", 1, "move");
      // Force the next entry's TimeStamp.now() to be after the 1ns window.
      await sleep.sleep(TimeSpan.milliseconds(2));
      push(store, "k", "a", 2, "move");
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(true);
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("preserves stack ordering when merging — newest inverse undoes first", () => {
      const { store } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(store, "k", { a: 0 });
      push(store, "k", "a", 1, "move");
      push(store, "k", "a", 2, "move");
      const r = store.prepareUndo("k");
      // The merged inverse should restore to the pre-first-push state.
      // First inverse runs the latest's inverse (a:1), then the prior's (a:0).
      expect(r?.actions[r.actions.length - 1]).toEqual({
        type: "set",
        key: "a",
        value: 0,
      });
    });

    it("trims to stackCap by dropping the oldest", async () => {
      const { store } = setupStore({
        coalesceWindow: TimeSpan.NANOSECOND,
        stackCap: 2,
      });
      prime(store, "k", { a: 0 });
      push(store, "k", "a", 1, "k1");
      await sleep.sleep(TimeSpan.milliseconds(2));
      push(store, "k", "a", 2, "k2");
      await sleep.sleep(TimeSpan.milliseconds(2));
      push(store, "k", "a", 3, "k3");
      store.prepareUndo("k")?.commit();
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(false);
    });
  });

  describe("prepareUndo", () => {
    it("returns null when the stack is empty", () => {
      const { store } = setupStore();
      prime(store, "k");
      expect(store.prepareUndo("k")).toBeNull();
    });

    it("returns the top entry's inverse and pushes a redo on commit", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      const r = store.prepareUndo("k");
      expect(r?.actions).toEqual([{ type: "set", key: "a", value: 0 }]);
      r?.commit();
      expect(store.hasUndo("k")).toBe(false);
      expect(store.hasRedo("k")).toBe(true);
    });

    it("skips a stale top entry (target touched after the entry's ts)", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0, b: 0 });
      // Two entries, oldest first.
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.recordEntry(
        "k",
        [{ type: "set", key: "b", value: 1 }],
        [{ type: "set", key: "b", value: 0 }],
        ["b"],
      );
      // Mark "b" as remote-touched at a future ts → top entry is stale.
      store.markRemoteTouched("k", ["b"], TimeStamp.now().add(TimeSpan.SECOND));
      const r = store.prepareUndo("k");
      // Walks past the stale "b" entry and returns "a"'s inverse.
      expect(r?.actions).toEqual([{ type: "set", key: "a", value: 0 }]);
    });

    it("returns null and drops stale entries when the entire stack is stale", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.markRemoteTouched("k", ["a"], TimeStamp.now().add(TimeSpan.SECOND));
      expect(store.prepareUndo("k")).toBeNull();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("ignores a remote touch older than the entry", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      // Stamp a touch in the past first.
      store.markRemoteTouched("k", ["a"], TimeStamp.now().sub(TimeSpan.MINUTE));
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      // Entry's ts is after the touch — it is not stale.
      expect(store.prepareUndo("k")?.actions).toEqual([
        { type: "set", key: "a", value: 0 },
      ]);
    });
  });

  describe("prepareRedo", () => {
    it("returns null when the redo stack is empty", () => {
      const { store } = setupStore();
      prime(store, "k");
      expect(store.prepareRedo("k")).toBeNull();
    });

    it("returns the original forward to re-apply, and pushes back to undo on commit", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.prepareUndo("k")?.commit();
      const r = store.prepareRedo("k");
      // Redo must re-apply the original forward, not the inverse.
      expect(r?.actions).toEqual([{ type: "set", key: "a", value: 1 }]);
      r?.commit();
      expect(store.hasRedo("k")).toBe(false);
      expect(store.hasUndo("k")).toBe(true);
    });

    it("skips a stale top entry (target touched after the entry's ts)", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0, b: 0 });
      // Two entries pushed then undone, oldest first into redo.
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.recordEntry(
        "k",
        [{ type: "set", key: "b", value: 1 }],
        [{ type: "set", key: "b", value: 0 }],
        ["b"],
      );
      store.prepareUndo("k")?.commit();
      store.prepareUndo("k")?.commit();
      // Mark "a" as remote-touched at a future ts → top of redo is stale.
      store.markRemoteTouched("k", ["a"], TimeStamp.now().add(TimeSpan.SECOND));
      const r = store.prepareRedo("k");
      // Walks past the stale "a" entry and returns "b"'s forward.
      expect(r?.actions).toEqual([{ type: "set", key: "b", value: 1 }]);
    });

    it("returns null and drops stale entries when the entire redo stack is stale", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.prepareUndo("k")?.commit();
      store.markRemoteTouched("k", ["a"], TimeStamp.now().add(TimeSpan.SECOND));
      expect(store.prepareRedo("k")).toBeNull();
      expect(store.hasRedo("k")).toBe(false);
    });

    it("ignores a remote touch older than the redo entry", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.markRemoteTouched("k", ["a"], TimeStamp.now().sub(TimeSpan.MINUTE));
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.prepareUndo("k")?.commit();
      expect(store.prepareRedo("k")?.actions).toEqual([
        { type: "set", key: "a", value: 1 },
      ]);
    });

    it("undo → redo → undo cycle restores state correctly at each step", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      // Manually drive the cycle (no substrate): apply forward, then walk
      // through prepareUndo / prepareRedo applying their actions so the
      // doc state mirrors what the substrate would do via replay.
      const entry = {
        forward: [{ type: "set" as const, key: "a", value: 1 }],
        inverse: [{ type: "set" as const, key: "a", value: 0 }],
      };
      store.replay("k", entry.forward);
      store.recordEntry("k", entry.forward, entry.inverse, ["a"]);
      expect(store.get("k")?.values.a).toBe(1);
      const undo = store.prepareUndo("k");
      store.replay("k", undo!.actions, { skipPreprocess: true });
      undo!.commit();
      expect(store.get("k")?.values.a).toBe(0);
      const redo = store.prepareRedo("k");
      store.replay("k", redo!.actions, { skipPreprocess: true });
      redo!.commit();
      expect(store.get("k")?.values.a).toBe(1);
      const undo2 = store.prepareUndo("k");
      store.replay("k", undo2!.actions, { skipPreprocess: true });
      undo2!.commit();
      expect(store.get("k")?.values.a).toBe(0);
    });
  });

  describe("beginTransaction", () => {
    it("accumulates and commits as a single undoable", async () => {
      const { store } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(store, "k", { a: 0 });
      const tx = store.beginTransaction("k", send, "move");
      tx.add({ type: "set", key: "a", value: 1 });
      tx.add([{ type: "set", key: "a", value: 2 }]);
      const ok = await tx.commit();
      expect(ok).toBe(true);
      expect(send).toHaveBeenCalledTimes(1);
      expect(send.mock.calls[0][0]).toHaveLength(2);
      expect(store.get("k")).toEqual({ values: { a: 2 } });
      const r = store.prepareUndo("k");
      // Inverse computed against the pre-transaction snapshot (a:0), not
      // the last add's pre-state (a:1). Latest inverse undoes first.
      expect(r?.actions).toEqual([
        { type: "set", key: "a", value: 1 },
        { type: "set", key: "a", value: 0 },
      ]);
    });

    it("returns false from commit when no actions were added", async () => {
      const { store } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(store, "k", { a: 0 });
      const ok = await store.beginTransaction("k", send).commit();
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("returns false from commit when the doc is not cached", async () => {
      const { store } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      const tx = store.beginTransaction("missing", send);
      tx.add({ type: "set", key: "a", value: 1 });
      const ok = await tx.commit();
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("subsequent commits return false and do not re-send", async () => {
      const { store } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(store, "k", { a: 0 });
      const tx = store.beginTransaction("k", send);
      tx.add({ type: "set", key: "a", value: 1 });
      expect(await tx.commit()).toBe(true);
      expect(await tx.commit()).toBe(false);
      expect(send).toHaveBeenCalledTimes(1);
    });

    it("add throws after the transaction is finalized", async () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      const tx = store.beginTransaction(
        "k",
        vi.fn<Send<Action>>(async () => {}),
      );
      tx.add({ type: "set", key: "a", value: 1 });
      await tx.commit();
      expect(() => tx.add({ type: "set", key: "a", value: 2 })).toThrow();
    });

    it("abort restores the pre-transaction snapshot and pushes nothing", () => {
      const { store } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(store, "k", { a: 0 });
      const tx = store.beginTransaction("k", send);
      tx.add({ type: "set", key: "a", value: 1 });
      tx.add({ type: "set", key: "a", value: 2 });
      expect(store.get("k")).toEqual({ values: { a: 2 } });
      tx.abort();
      expect(store.get("k")).toEqual({ values: { a: 0 } });
      expect(send).not.toHaveBeenCalled();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("abort is a no-op after commit", async () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      const tx = store.beginTransaction(
        "k",
        vi.fn<Send<Action>>(async () => {}),
      );
      tx.add({ type: "set", key: "a", value: 5 });
      await tx.commit();
      tx.abort();
      // State stays at the committed value.
      expect(store.get("k")).toEqual({ values: { a: 5 } });
    });

    it("rolls back doc and stack when send fails", async () => {
      const { store } = setupStore();
      const send = vi.fn(async () => {
        throw new Error("boom");
      });
      prime(store, "k", { a: 0 });
      const tx = store.beginTransaction("k", send);
      tx.add({ type: "set", key: "a", value: 1 });
      const ok = await tx.commit();
      expect(ok).toBe(false);
      expect(store.get("k")).toEqual({ values: { a: 0 } });
      expect(store.hasUndo("k")).toBe(false);
    });
  });

  describe("applyRemote", () => {
    it("applies foreign actions and stamps targets as remote-touched", async () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      // Push a local entry first so we can verify it gets marked stale.
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      // Force a strictly-after timestamp on the remote stamp.
      await sleep.sleep(TimeSpan.milliseconds(2));
      store.applyRemote("k", 1, "foreign-1", [{ type: "set", key: "a", value: 99 }]);
      expect(store.get("k")).toEqual({ values: { a: 99 } });
      expect(store.prepareUndo("k")).toBeNull();
    });

    it("is a no-op when the doc is not cached", () => {
      const { store } = setupStore();
      store.applyRemote("missing", 1, "foreign-1", [
        { type: "set", key: "a", value: 1 },
      ]);
      expect(store.get("missing")).toBeUndefined();
    });

    it("drops echoes whose seq does not exceed the high-water mark", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.applyRemote("k", 5, "f-1", [{ type: "set", key: "a", value: 1 }]);
      expect(store.get("k")).toEqual({ values: { a: 1 } });
      // Same seq — a duplicate frame — must be dropped.
      store.applyRemote("k", 5, "f-1", [{ type: "set", key: "a", value: 99 }]);
      expect(store.get("k")).toEqual({ values: { a: 1 } });
      // Older seq — a reordered or replayed frame — must also be dropped.
      store.applyRemote("k", 4, "f-2", [{ type: "set", key: "a", value: 99 }]);
      expect(store.get("k")).toEqual({ values: { a: 1 } });
      // Fresher seq applies and advances the high-water mark.
      store.applyRemote("k", 6, "f-3", [{ type: "set", key: "a", value: 2 }]);
      expect(store.get("k")).toEqual({ values: { a: 2 } });
    });

    it("treats seq=0 as unstamped and always applies for legacy-server compat", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.applyRemote("k", 0, "f-1", [{ type: "set", key: "a", value: 1 }]);
      store.applyRemote("k", 0, "f-2", [{ type: "set", key: "a", value: 2 }]);
      expect(store.get("k")).toEqual({ values: { a: 2 } });
    });

    it("skips the reducer on own echoes when no foreign action interleaved", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      // Originator simulates local replay then registers the dispatch as
      // outstanding before the echo arrives.
      store.registerOutstandingDispatch("k", "own-1");
      // Now the echo lands. The reducer should NOT re-run: the local replay
      // was already authoritative.
      store.applyRemote("k", 1, "own-1", [{ type: "set", key: "a", value: 99 }]);
      // State unchanged — the local replay's effect (a=0 here, since the test
      // skipped the replay step) is preserved.
      expect(store.get("k")).toEqual({ values: { a: 0 } });
    });

    it("does not mark targets remote-touched on own echoes", async () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      await sleep.sleep(TimeSpan.milliseconds(2));
      store.registerOutstandingDispatch("k", "own-1");
      store.applyRemote("k", 1, "own-1", [{ type: "set", key: "a", value: 1 }]);
      // The local undo entry survives because the echo was own.
      expect(store.prepareUndo("k")).not.toBeNull();
    });

    it("re-applies own echoes whose outstanding window was disturbed by a foreign action", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      // Alice locally replays both her dispatches up front and registers
      // them as outstanding. State after both local replays is a=20.
      store.applyRemote("k", 0, "alice-1", [{ type: "set", key: "a", value: 10 }]);
      // Reset state to what it would be after local replays:
      prime(store, "k", { a: 20 });
      store.registerOutstandingDispatch("k", "alice-1");
      store.registerOutstandingDispatch("k", "alice-2");
      // Now the echoes arrive in server-seq order.
      // Alice (own) seq=1 — own and not disturbed → skip.
      store.applyRemote("k", 1, "alice-1", [{ type: "set", key: "a", value: 10 }]);
      expect(store.get("k")).toEqual({ values: { a: 20 } });
      // Bob (foreign) seq=2 — applies, marks all outstanding own as disturbed.
      store.applyRemote("k", 2, "bob-1", [{ type: "set", key: "a", value: 5 }]);
      expect(store.get("k")).toEqual({ values: { a: 5 } });
      // Alice (own) seq=3 — own but disturbed → reduce to recover.
      store.applyRemote("k", 3, "alice-2", [{ type: "set", key: "a", value: 20 }]);
      expect(store.get("k")).toEqual({ values: { a: 20 } });
    });

    it("re-applies own echoes when their dispatchKey is unknown (registration lost the race against the echo)", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      // No registration: simulate the case where the echo arrived before the
      // dispatch promise resolved, so the originator never got to register.
      // The substrate treats this as foreign and reduces — safer than
      // silently dropping the action.
      store.applyRemote("k", 1, "stranger", [{ type: "set", key: "a", value: 7 }]);
      expect(store.get("k")).toEqual({ values: { a: 7 } });
    });
  });

  describe("markRemoteTouched", () => {
    it("is a no-op for empty targets", () => {
      const { store } = setupStore();
      prime(store, "k");
      // No throw, no state change observable to a fresh prepareUndo.
      store.markRemoteTouched("k", []);
      expect(store.hasUndo("k")).toBe(false);
    });

    it("uses the supplied timestamp", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      const ts = TimeStamp.now();
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      // Stamp at a ts before the entry → entry survives.
      store.markRemoteTouched("k", ["a"], ts.sub(TimeSpan.SECOND));
      expect(store.prepareUndo("k")).not.toBeNull();
    });
  });

  describe("delete (cascade)", () => {
    it("drops the doc and its undo state for a single key", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.delete("k");
      expect(store.get("k")).toBeUndefined();
      expect(store.hasUndo("k")).toBe(false);
    });

    it("drops both for an array of keys", () => {
      const { store } = setupStore();
      prime(store, "k1");
      prime(store, "k2");
      store.recordEntry(
        "k1",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.recordEntry(
        "k2",
        [{ type: "set", key: "a", value: 2 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.delete(["k1", "k2"]);
      expect(store.has(["k1", "k2"])).toBe(false);
      expect(store.hasUndo("k1")).toBe(false);
      expect(store.hasUndo("k2")).toBe(false);
    });

    it("drops both for a predicate match", () => {
      const { store } = setupStore();
      prime(store, "keep", { a: 0 });
      prime(store, "drop", { a: 0 });
      store.recordEntry(
        "drop",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.delete((_v, k) => k === "drop");
      expect(store.get("drop")).toBeUndefined();
      expect(store.get("keep")).toBeDefined();
      expect(store.hasUndo("drop")).toBe(false);
    });
  });

  describe("hasUndo / hasRedo", () => {
    it("is false initially and tracks the undo and redo stacks", () => {
      const { store } = setupStore();
      prime(store, "k", { a: 0 });
      expect(store.hasUndo("k")).toBe(false);
      expect(store.hasRedo("k")).toBe(false);
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(store.hasUndo("k")).toBe(true);
      expect(store.hasRedo("k")).toBe(false);
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(false);
      expect(store.hasRedo("k")).toBe(true);
      store.prepareRedo("k")?.commit();
      expect(store.hasUndo("k")).toBe(true);
      expect(store.hasRedo("k")).toBe(false);
    });
  });

  describe("onUndoStateChange", () => {
    // Notification semantics mirror ScopedUnaryStore: a scope's writes do not
    // notify its own listeners (the writer already knows). Cross-scope
    // observers receive the notification — that's the React reactivity path.
    it("fires on cross-scope entry record and on cascade delete", () => {
      const { store, scope } = setupStore();
      const observer = scope("observer");
      const cb = vi.fn();
      observer.onUndoStateChange(cb, "k");
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(cb).toHaveBeenCalled();
      cb.mockClear();
      store.delete("k");
      expect(cb).toHaveBeenCalled();
    });

    it("does not notify the writing scope (self-notify is suppressed)", () => {
      const { store } = setupStore();
      const cb = vi.fn();
      store.onUndoStateChange(cb, "k");
      prime(store, "k", { a: 0 });
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      expect(cb).not.toHaveBeenCalled();
    });

    it("stops firing after the destructor is called", () => {
      const { store, scope } = setupStore();
      const observer = scope("observer");
      const cb = vi.fn();
      const off = observer.onUndoStateChange(cb, "k");
      prime(store, "k", { a: 0 });
      off();
      store.recordEntry(
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
      const { store, controller } = setupStore();
      prime(store, "k", { a: 1 });
      const listener = controller.listener("docs_dispatch", frameZ);
      expect(listener.channel).toBe("docs_dispatch");
      void listener.onChange({
        changed: {
          key: "k",
          dispatchKey: "remote-1",
          seq: 1,
          actions: [{ type: "set", key: "a", value: 42 }],
        },
        store: {},
      });
      expect(store.get("k")).toEqual({ values: { a: 42 } });
    });
  });

  describe("defaults", () => {
    it("treats every action as undoable when isUndoable is omitted", () => {
      const { store } = setupStore();
      prime(store, "k");
      store.recordEntry("k", [{ type: "noop" }], [], ["a"]);
      expect(store.hasUndo("k")).toBe(true);
    });

    it("uses 'default' as the kind when kindOf is omitted", () => {
      const { store } = setupStore({ coalesceWindow: TimeSpan.SECOND });
      prime(store, "k", { a: 0 });
      // Two entries with different action types — both fall under the same
      // default kind and should coalesce when targets+window match.
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      store.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 2 }],
        [{ type: "set", key: "a", value: 1 }],
        ["a"],
      );
      // Coalesced into one entry.
      store.prepareUndo("k")?.commit();
      expect(store.hasUndo("k")).toBe(false);
    });
  });
});
