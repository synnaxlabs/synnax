// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache, dispatch, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { array, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, type Mock, vi } from "vitest";

import { Flux } from "@/flux";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

type SendFn = (actions: schematic.Action[]) => Promise<void>;
type SendMock = Mock<SendFn>;

type Domain = dispatch.Domain<schematic.Key, schematic.Schematic, schematic.Action>;

type Handle = ReturnType<
  typeof Flux.createDispatch<schematic.Key, schematic.Schematic, schematic.Action>
>;

// The hooks subscribe in their own scope so writes made through the domain's
// default scope still notify them.
const HOOK_SCOPE = "hooks";

interface Harness {
  domain: Domain;
  docs: cache.ScopedUnaryStore<schematic.Key, schematic.Schematic>;
  controller: dispatch.Controller<schematic.Key, schematic.Schematic, schematic.Action>;
}

// Binds the production dispatch controller to a controllable network send,
// exposed through the same Domain surface client.schematics implements.
const createHarness = (send: SendFn): Harness => {
  const docs = new cache.ScopedUnaryStore<schematic.Key, schematic.Schematic>(
    cache.consoleErrorHandler,
  );
  const controller = new dispatch.Controller<
    schematic.Key,
    schematic.Schematic,
    schematic.Action
  >({
    store: docs,
    handleError: cache.consoleErrorHandler,
    reduce: schematic.reduceAll,
    kindOf: schematic.kindOf,
  });
  const sendDispatch: dispatch.SendDispatch<schematic.Action> = async (actions) =>
    await send(actions);
  const domain: Domain = {
    dispatch: async (key, actions, opts) =>
      await controller.dispatch(
        "",
        key,
        array.toArray(actions),
        sendDispatch,
        opts?.preprocess,
      ),
    undo: async (key) => await controller.undo("", key, sendDispatch),
    redo: async (key) => await controller.redo("", key, sendDispatch),
    hasUndo: (key) => controller.hasUndo(key),
    hasRedo: (key) => controller.hasRedo(key),
    onUndoStateChange: (callback, key) =>
      controller.onUndoStateChange(HOOK_SCOPE, callback, key),
    beginTransaction: (key, kind) =>
      controller.transaction("", key, sendDispatch, kind),
  };
  return { domain, docs, controller };
};

const createDoc = (key: schematic.Key): schematic.Schematic =>
  schematic.schematicZ.parse({
    key,
    name: `dispatch_test_${uuid.create()}`,
    nodes: [
      { key: "n1", position: { x: 0, y: 0 } },
      { key: "n2", position: { x: 10, y: 10 } },
    ],
    edges: [],
    configs: {},
  });

const makeDispatch = (domain: Domain): Handle =>
  Flux.createDispatch<schematic.Key, schematic.Schematic, schematic.Action>({
    domain: () => domain,
  });

interface SetupResult<H extends object> {
  result: { current: H };
  send: SendMock;
  td: Handle;
  key: schematic.Key;
  docs: Harness["docs"];
  controller: Harness["controller"];
}

const setupHook = <H extends object>(
  hookFn: (td: Handle, key: schematic.Key) => H,
  sendMock: SendMock = vi.fn<SendFn>(async () => {}),
): SetupResult<H> => {
  const { domain, docs, controller } = createHarness(sendMock);
  const key = uuid.create();
  docs.set("seed", key, createDoc(key));
  const td = makeDispatch(domain);
  const { result } = renderHook(() => hookFn(td, key), { wrapper });
  return { result, send: sendMock, td, key, docs, controller };
};

const getDoc = (
  docs: Harness["docs"],
  key: schematic.Key,
): schematic.Schematic | undefined => docs.get(key);

describe("Flux.createDispatch", () => {
  describe("dispatch", () => {
    it("applies locally, sends, and pushes onto the undo stack", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      expect(result.current.undo.canUndo).toBe(false);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 99, y: 99 } }),
        });
      });
      expect(send).toHaveBeenCalledTimes(1);
      expect(getDoc(docs, key)?.nodes[0].position).toEqual({ x: 99, y: 99 });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
    });

    it("rolls back local state when send fails", async () => {
      const send = vi.fn<SendFn>(async () => {
        throw new Error("send failed");
      });
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      const before = getDoc(docs, key);
      let ok = true;
      await act(async () => {
        ok = await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 99, y: 99 } }),
        });
      });
      expect(ok).toBe(false);
      expect(getDoc(docs, key)).toEqual(before);
      expect(result.current.undo.canUndo).toBe(false);
    });

    it("sends an array of actions in a single send call", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: [
            schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } }),
            schematic.setNodePosition({ key: "n2", position: { x: 2, y: 2 } }),
          ],
        });
      });
      expect(send).toHaveBeenCalledTimes(1);
      expect(send.mock.calls[0][0]).toHaveLength(2);
      const doc = getDoc(docs, key);
      expect(doc?.nodes[0].position).toEqual({ x: 1, y: 1 });
      expect(doc?.nodes[1].position).toEqual({ x: 2, y: 2 });
    });

    it("returns true without sending when given an empty actions array", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      let ok = false;
      await act(async () => {
        ok = await result.current.dispatch.dispatchAsync({ key, actions: [] });
      });
      expect(ok).toBe(true);
      expect(send).not.toHaveBeenCalled();
      expect(result.current.undo.canUndo).toBe(false);
    });

    it("returns true without sending when the replay leaves state unchanged", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      const before = getDoc(docs, key);
      let ok = false;
      await act(async () => {
        ok = await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({
            key: "nonexistent",
            position: { x: 1, y: 1 },
          }),
        });
      });
      expect(ok).toBe(true);
      expect(send).not.toHaveBeenCalled();
      expect(getDoc(docs, key)).toBe(before);
      expect(result.current.undo.canUndo).toBe(false);
    });

    it("returns false from dispatchAsync when the doc is not cached", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { domain } = createHarness(send);
      const td = makeDispatch(domain);
      // Don't seed the store — dispatch on an unknown key.
      const { result } = renderHook(() => td.useDispatch(), { wrapper });
      let ok = true;
      await act(async () => {
        ok = await result.current.dispatchAsync({
          key: uuid.create(),
          actions: schematic.setNodePosition({ key: "n", position: { x: 0, y: 0 } }),
        });
      });
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("dispatch (sync) fires the action without awaiting", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      act(() => {
        result.current.dispatch.dispatch({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 4, y: 4 } }),
        });
      });
      await waitFor(() => {
        expect(send).toHaveBeenCalledTimes(1);
        expect(getDoc(docs, key)?.nodes[0].position).toEqual({ x: 4, y: 4 });
      });
    });
  });

  describe("undo", () => {
    it("reverts the last user dispatch and pushes a redo entry", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
          redo: td.useRedo({ key: k }),
        }),
        send,
      );
      const before = getDoc(docs, key);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 50, y: 60 } }),
        });
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      act(() => result.current.undo.undo());
      await waitFor(() => {
        expect(send).toHaveBeenCalledTimes(2);
        expect(getDoc(docs, key)?.nodes[0].position).toEqual(before?.nodes[0].position);
        expect(result.current.redo.canRedo).toBe(true);
      });
    });

    it("returns the entry to the undo stack when the inverse send fails", async () => {
      let calls = 0;
      const send = vi.fn<SendFn>(async () => {
        if (++calls === 2) throw new Error("inverse send failed");
      });
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
          redo: td.useRedo({ key: k }),
        }),
        send,
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 5, y: 6 } }),
        });
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      const after = getDoc(docs, key);
      act(() => result.current.undo.undo());
      await waitFor(() => expect(send).toHaveBeenCalledTimes(2));
      expect(getDoc(docs, key)).toEqual(after);
      expect(result.current.undo.canUndo).toBe(true);
      expect(result.current.redo.canRedo).toBe(false);
    });

    it("auto-advances past entries invalidated by remote touches", async () => {
      const { result, key, controller } = setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 7, y: 8 } }),
        });
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      // Stamp n1 strictly after the entry's ts so the only entry is stale;
      // undo should drop it without sending.
      act(() => {
        controller.markRemoteTouched(
          "",
          key,
          ["n1"],
          TimeStamp.now().add(TimeSpan.SECOND),
        );
      });
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });
  });

  describe("redo", () => {
    it("re-applies the original forward and restores the post-dispatch state", async () => {
      const { result, key, docs } = setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
        redo: td.useRedo({ key: k }),
      }));
      const before = getDoc(docs, key);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } }),
        });
      });
      const afterDispatch = getDoc(docs, key);
      act(() => result.current.undo.undo());
      await waitFor(() => {
        expect(getDoc(docs, key)?.nodes[0].position).toEqual(before?.nodes[0].position);
        expect(result.current.redo.canRedo).toBe(true);
      });
      act(() => result.current.redo.redo());
      // Redo must restore the post-dispatch state, not leave it at the
      // undone state. (This caught the swap bug in prepareUndo/prepareRedo.)
      await waitFor(() => {
        expect(getDoc(docs, key)?.nodes[0].position).toEqual(
          afterDispatch?.nodes[0].position,
        );
        expect(result.current.undo.canUndo).toBe(true);
      });
    });

    it("clears the redo stack on a new user dispatch", async () => {
      const { result, key } = setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
        redo: td.useRedo({ key: k }),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } }),
        });
      });
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.redo.canRedo).toBe(true));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 7, y: 7 } }),
        });
      });
      await waitFor(() => expect(result.current.redo.canRedo).toBe(false));
    });

    it("auto-advances past redo entries invalidated by remote touches", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key, controller } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
          redo: td.useRedo({ key: k }),
        }),
        send,
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 7, y: 8 } }),
        });
      });
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.redo.canRedo).toBe(true));
      // Stamp n1 strictly after the entry's ts so the only redo entry is
      // stale; redo should drop it without sending.
      act(() => {
        controller.markRemoteTouched(
          "",
          key,
          ["n1"],
          TimeStamp.now().add(TimeSpan.SECOND),
        );
      });
      const callsBefore = send.mock.calls.length;
      act(() => result.current.redo.redo());
      await waitFor(() => expect(result.current.redo.canRedo).toBe(false));
      expect(send).toHaveBeenCalledTimes(callsBefore);
    });

    it("returns the entry to the redo stack when the redo send fails", async () => {
      let calls = 0;
      const send = vi.fn<SendFn>(async () => {
        if (calls++ === 2) throw new Error("redo send failed");
      });
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
          redo: td.useRedo({ key: k }),
        }),
        send,
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 4, y: 4 } }),
        });
      });
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.redo.canRedo).toBe(true));
      const undoneState = getDoc(docs, key);
      act(() => result.current.redo.redo());
      await waitFor(() => expect(send).toHaveBeenCalledTimes(3));
      expect(getDoc(docs, key)).toEqual(undoneState);
      expect(result.current.redo.canRedo).toBe(true);
    });
  });

  describe("coalescing", () => {
    it("merges same-kind dispatches inside the coalesce window", async () => {
      const { result, key } = setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
      }));
      for (const x of [10, 20, 30])
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key,
            actions: schematic.setNodePosition({ key: "n1", position: { x, y: 0 } }),
          });
        });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });

    it("does not merge across kind boundaries", async () => {
      const { result, key, send } = setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } }),
        });
      });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.rename({ name: "renamed" }),
        });
      });
      act(() => result.current.undo.undo());
      await waitFor(() => expect(send).toHaveBeenCalledTimes(3));
      expect(result.current.undo.canUndo).toBe(true);
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });

    it("restores state when undoing two coalesced setConfig dispatches", async () => {
      const { result, key, docs } = setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
      }));
      // Establish an existing config so the next two setConfigs hit the
      // "existing != null" branch and capture non-empty inverses. setNode's
      // kind differs from "set_config", so this entry won't coalesce with
      // the burst that follows.
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNode({
            node: { key: "n1", position: { x: 0, y: 0 } },
            config: { label: "original" },
          }),
        });
      });
      const baseline = getDoc(docs, key)?.configs;
      // Two rapid same-kind dispatches → coalesce inside the 500ms window
      // into a single entry whose merged inverse is [inv2, inv1]. Before the
      // snapshot() fix in client/ts/src/schematic/actions.ts the second
      // inverse action threw inside reduceAll ("current expects a draft")
      // and undo silently did nothing.
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setConfig({ key: "n1", config: { label: "first" } }),
        });
      });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setConfig({ key: "n1", config: { label: "second" } }),
        });
      });
      act(() => result.current.undo.undo());
      await waitFor(() => expect(getDoc(docs, key)?.configs).toEqual(baseline));
    });
  });

  describe("transactions", () => {
    it("commits accumulated actions as one undoable", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      await act(async () => {
        const tx = result.current.dispatch.beginTransaction({ key, kind: "move" });
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } })]);
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 2, y: 2 } })]);
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 3, y: 3 } })]);
        await tx.commit();
      });
      expect(send).toHaveBeenCalledTimes(1);
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });

    it("aborts without sending or pushing", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      const initial = getDoc(docs, key);
      act(() => {
        const tx = result.current.dispatch.beginTransaction({ key, kind: "move" });
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 9, y: 9 } })]);
        tx.abort();
      });
      expect(send).not.toHaveBeenCalled();
      expect(result.current.undo.canUndo).toBe(false);
      expect(getDoc(docs, key)).toEqual(initial);
    });

    it("restores the pre-transaction snapshot when commit's send fails", async () => {
      const send = vi.fn<SendFn>(async () => {
        throw new Error("commit failed");
      });
      const { result, key, docs } = setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      const initial = getDoc(docs, key);
      let ok = true;
      await act(async () => {
        const tx = result.current.dispatch.beginTransaction({ key, kind: "move" });
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } })]);
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 2, y: 2 } })]);
        ok = await tx.commit();
      });
      expect(ok).toBe(false);
      expect(getDoc(docs, key)).toEqual(initial);
      expect(result.current.undo.canUndo).toBe(false);
    });
  });

  describe("cascade", () => {
    it("drops undo state when the document is deleted", async () => {
      const { result, key, docs } = setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } }),
        });
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      act(() => {
        docs.delete("remote", key);
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
      expect(getDoc(docs, key)).toBeUndefined();
    });
  });

  describe("empty stacks", () => {
    it("undo is a no-op when nothing has been dispatched", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result } = setupHook(
        (td, k) => ({
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      expect(result.current.undo.canUndo).toBe(false);
      act(() => result.current.undo.undo());
      expect(send).not.toHaveBeenCalled();
    });

    it("redo is a no-op when nothing has been undone", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result } = setupHook(
        (td, k) => ({
          redo: td.useRedo({ key: k }),
        }),
        send,
      );
      expect(result.current.redo.canRedo).toBe(false);
      act(() => result.current.redo.redo());
      expect(send).not.toHaveBeenCalled();
    });
  });

  describe("multi-key isolation", () => {
    it("undo state for one document does not affect another", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { domain, docs } = createHarness(send);
      const a = uuid.create();
      const b = uuid.create();
      docs.set("seed", a, createDoc(a));
      docs.set("seed", b, createDoc(b));
      const td = makeDispatch(domain);
      const { result } = renderHook(
        () => ({
          dispatch: td.useDispatch(),
          undoA: td.useUndo({ key: a }),
          undoB: td.useUndo({ key: b }),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: a,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 5, y: 5 } }),
        });
      });
      await waitFor(() => expect(result.current.undoA.canUndo).toBe(true));
      // Dispatching on `a` must not enable undo on `b`.
      expect(result.current.undoB.canUndo).toBe(false);
    });
  });

  describe("live schematic domain", () => {
    const live = Flux.createDispatch<
      schematic.Key,
      schematic.Schematic,
      schematic.Action
    >({ domain: (client) => client.schematics });

    // Reads server truth: an unsubscribed retrieve always refetches.
    const raw = createTestClient();

    const createLiveSchem = async (): Promise<schematic.Schematic> => {
      const proj = await client.projects.create({
        name: `dispatch_${uuid.create()}`,
        layout: {},
      });
      return await client.schematics.create(proj.key, {
        name: `dispatch_live_${uuid.create()}`,
        nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
        edges: [],
        configs: {},
      });
    };

    const serverPosition = async (key: schematic.Key) =>
      (await raw.schematics.retrieve({ key })).nodes[0].position;

    it("dispatches through the hooks and persists to the cluster", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createLiveSchem();
      const { result } = renderHook(() => live.useDispatch(), { wrapper: Wrapper });
      let ok = false;
      await act(async () => {
        ok = await result.current.dispatchAsync({
          key: schem.key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 7, y: 8 } }),
        });
      });
      expect(ok).toBe(true);
      expect(await serverPosition(schem.key)).toEqual({ x: 7, y: 8 });
      expect(client.schematics.hasUndo(schem.key)).toBe(true);
    }, 15000);

    it("undoes and redoes a dispatch against the cluster", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createLiveSchem();
      const { result } = renderHook(
        () => ({
          dispatch: live.useDispatch(),
          undo: live.useUndo({ key: schem.key }),
          redo: live.useRedo({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: schem.key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 5, y: 6 } }),
        });
      });
      act(() => result.current.undo.undo());
      await expect
        .poll(async () => await serverPosition(schem.key), { timeout: 5000 })
        .toEqual({ x: 0, y: 0 });
      expect(client.schematics.hasRedo(schem.key)).toBe(true);
      act(() => result.current.redo.redo());
      await expect
        .poll(async () => await serverPosition(schem.key), { timeout: 5000 })
        .toEqual({ x: 5, y: 6 });
    }, 15000);

    it("returns false and rolls back when the cluster rejects a dispatch", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createLiveSchem();
      // Snapshots reject dispatches server-side, giving a deterministic
      // send failure through the production domain.
      const snap = await client.schematics.copy({
        key: schem.key,
        name: `snap_${uuid.create()}`,
        snapshot: true,
      });
      const { result } = renderHook(() => live.useDispatch(), { wrapper: Wrapper });
      let ok = true;
      await act(async () => {
        ok = await result.current.dispatchAsync({
          key: snap.key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 3, y: 3 } }),
        });
      });
      expect(ok).toBe(false);
      expect(client.schematics.hasUndo(snap.key)).toBe(false);
      expect(await serverPosition(snap.key)).toEqual({ x: 0, y: 0 });
    }, 15000);

    it("drops undo state when the schematic is deleted", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createLiveSchem();
      const { result } = renderHook(
        () => ({
          dispatch: live.useDispatch(),
          undo: live.useUndo({ key: schem.key }),
        }),
        { wrapper: Wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: schem.key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } }),
        });
      });
      expect(client.schematics.hasUndo(schem.key)).toBe(true);
      await act(async () => await client.schematics.delete(schem.key));
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
      expect(client.schematics.hasUndo(schem.key)).toBe(false);
    }, 15000);

    it("commits a transaction as one undoable against the cluster", async () => {
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const schem = await createLiveSchem();
      const { result } = renderHook(() => live.useDispatch(), { wrapper: Wrapper });
      let ok = false;
      await act(async () => {
        const tx = result.current.beginTransaction({ key: schem.key, kind: "move" });
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } })]);
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 2, y: 2 } })]);
        ok = await tx.commit();
      });
      expect(ok).toBe(true);
      expect(await serverPosition(schem.key)).toEqual({ x: 2, y: 2 });
      expect(client.schematics.hasUndo(schem.key)).toBe(true);
      await act(async () => {
        await client.schematics.undo(schem.key);
      });
      expect(await serverPosition(schem.key)).toEqual({ x: 0, y: 0 });
      expect(client.schematics.hasUndo(schem.key)).toBe(false);
    }, 15000);
  });
});
