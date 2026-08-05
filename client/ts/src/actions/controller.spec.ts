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

import { Controller, type Frame, type Reducer, type Send } from "@/actions/controller";
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

const setupStore = (
  opts: Partial<{
    isUndoable: (a: Action) => boolean;
    kindOf: (acts: Action[]) => string;
    coalesceWindow: TimeSpan;
    stackCap: number;
    preprocess: (s: Doc, acts: Action[]) => Action[];
  }> = {},
) => {
  const errors: Error[] = [];
  const onError = (error: Error) => errors.push(error);
  const docs = new Table<string, Doc>({ onError });
  const controller = new Controller<string, Doc, Action>({
    store: docs,
    onError,
    reduce: reducer,
    isUndoable: opts.isUndoable,
    kindOf: opts.kindOf,
    createOf: (a) => (a.type === "create" ? a.doc : undefined),
    coalesceWindow: opts.coalesceWindow,
    stackCap: opts.stackCap,
    preprocess: opts.preprocess,
  });
  return { errors, docs, controller };
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

    it("does not merge when the prior entry is older than the window", async () => {
      const { docs, controller } = setupStore({
        coalesceWindow: TimeSpan.NANOSECOND,
      });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "move");
      // Force the next entry's TimeStamp.now() to be after the 1ns window.
      await sleep.sleep(TimeSpan.milliseconds(2));
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

    it("trims to stackCap by dropping the oldest", async () => {
      const { docs, controller } = setupStore({
        coalesceWindow: TimeSpan.NANOSECOND,
        stackCap: 2,
      });
      prime(docs, "k", { a: 0 });
      push(controller, "k", "a", 1, "k1");
      await sleep.sleep(TimeSpan.milliseconds(2));
      push(controller, "k", "a", 2, "k2");
      await sleep.sleep(TimeSpan.milliseconds(2));
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
      // Mark "b" as remote-touched at a future ts → top entry is stale.
      controller.markRemoteTouched("k", ["b"], TimeStamp.now().add(TimeSpan.SECOND));
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
      controller.markRemoteTouched("k", ["a"], TimeStamp.now().add(TimeSpan.SECOND));
      expect(controller.prepareUndo("k")).toBeNull();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("ignores a remote touch older than the entry", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      // Stamp a touch in the past first.
      controller.markRemoteTouched("k", ["a"], TimeStamp.now().sub(TimeSpan.MINUTE));
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      // Entry's ts is after the touch — it is not stale.
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
      // Mark "a" as remote-touched at a future ts → top of redo is stale.
      controller.markRemoteTouched("k", ["a"], TimeStamp.now().add(TimeSpan.SECOND));
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
      controller.markRemoteTouched("k", ["a"], TimeStamp.now().add(TimeSpan.SECOND));
      expect(controller.prepareRedo("k")).toBeNull();
      expect(controller.hasRedo("k")).toBe(false);
    });

    it("ignores a remote touch older than the redo entry", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      controller.markRemoteTouched("k", ["a"], TimeStamp.now().sub(TimeSpan.MINUTE));
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

  describe("beginTransaction", () => {
    it("accumulates and commits as a single undoable", async () => {
      const { docs, controller } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(docs, "k", { a: 0 });
      const tx = controller.beginTransaction("k", send, "move");
      tx.add({ type: "set", key: "a", value: 1 });
      tx.add([{ type: "set", key: "a", value: 2 }]);
      const ok = await tx.commit();
      expect(ok).toBe(true);
      expect(send).toHaveBeenCalledTimes(1);
      expect(send.mock.calls[0][0]).toHaveLength(2);
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
      const { docs, controller } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(docs, "k", { a: 0 });
      const ok = await controller.beginTransaction("k", send).commit();
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
      expect(controller.hasUndo("k")).toBe(false);
    });

    it("returns false from commit when the doc is not cached", async () => {
      const { controller } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      const tx = controller.beginTransaction("missing", send);
      tx.add({ type: "set", key: "a", value: 1 });
      const ok = await tx.commit();
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("subsequent commits return false and do not re-send", async () => {
      const { docs, controller } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(docs, "k", { a: 0 });
      const tx = controller.beginTransaction("k", send);
      tx.add({ type: "set", key: "a", value: 1 });
      expect(await tx.commit()).toBe(true);
      expect(await tx.commit()).toBe(false);
      expect(send).toHaveBeenCalledTimes(1);
    });

    it("add throws after the transaction is finalized", async () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const tx = controller.beginTransaction(
        "k",
        vi.fn<Send<Action>>(async () => {}),
      );
      tx.add({ type: "set", key: "a", value: 1 });
      await tx.commit();
      expect(() => tx.add({ type: "set", key: "a", value: 2 })).toThrow();
    });

    it("abort restores the pre-transaction snapshot and pushes nothing", () => {
      const { docs, controller } = setupStore();
      const send = vi.fn<Send<Action>>(async () => {});
      prime(docs, "k", { a: 0 });
      const tx = controller.beginTransaction("k", send);
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
      const tx = controller.beginTransaction(
        "k",
        vi.fn<Send<Action>>(async () => {}),
      );
      tx.add({ type: "set", key: "a", value: 5 });
      await tx.commit();
      tx.abort();
      // State stays at the committed value.
      expect(docs.get("k")).toEqual({ values: { a: 5 } });
    });

    it("rolls back doc and stack when send fails", async () => {
      const { docs, controller } = setupStore();
      const send = vi.fn(async () => {
        throw new Error("boom");
      });
      prime(docs, "k", { a: 0 });
      const tx = controller.beginTransaction("k", send);
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
      // Force a strictly-after timestamp on the remote stamp.
      await sleep.sleep(TimeSpan.milliseconds(2));
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
      await sleep.sleep(TimeSpan.milliseconds(2));
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
      // No registration: simulate the case where the echo arrived before the
      // dispatch promise resolved, so the originator never got to register.
      // The substrate treats this as foreign and reduces — safer than
      // silently dropping the action.
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

    it("uses the supplied timestamp", () => {
      const { docs, controller } = setupStore();
      prime(docs, "k", { a: 0 });
      const ts = TimeStamp.now();
      controller.recordEntry(
        "k",
        [{ type: "set", key: "a", value: 1 }],
        [{ type: "set", key: "a", value: 0 }],
        ["a"],
      );
      // Stamp at a ts before the entry → entry survives.
      controller.markRemoteTouched("k", ["a"], ts.sub(TimeSpan.SECOND));
      expect(controller.prepareUndo("k")).not.toBeNull();
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
      void listener.onChange({
        key: "k",
        dispatchKey: "remote-1",
        seq: 1,
        actions: [{ type: "set", key: "a", value: 42 }],
      });
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
