// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, schematic } from "@synnaxlabs/client";
import { TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createSchem = async () => {
  const ws = await client.workspaces.create({
    name: `ws_${uuid.create()}`,
    layout: {},
  });
  return await client.schematics.create(ws.key, {
    name: `dispatch_test_${uuid.create()}`,
    nodes: [
      { key: "n1", position: { x: 0, y: 0 } },
      { key: "n2", position: { x: 10, y: 10 } },
    ],
    edges: [],
    configs: {},
  });
};

type SendFn = Flux.CreateDispatchParams<
  schematic.Key,
  schematic.Action,
  typeof Schematic.FLUX_STORE_KEY
>["send"];

type Handle = ReturnType<
  typeof Flux.createDispatch<
    schematic.Key,
    schematic.Schematic,
    schematic.Action,
    typeof Schematic.FLUX_STORE_KEY,
    Schematic.FluxSubStore
  >
>;

const makeDispatch = (sendMock: SendFn): Handle =>
  Flux.createDispatch<
    schematic.Key,
    schematic.Schematic,
    schematic.Action,
    typeof Schematic.FLUX_STORE_KEY,
    Schematic.FluxSubStore
  >({ storeKey: Schematic.FLUX_STORE_KEY, send: sendMock });

const primeCache = async (Wrapper: FC<PropsWithChildren>, key: schematic.Key) => {
  const Display = (): ReactElement => {
    Schematic.useEnsureRetrieved({ key });
    return <div data-testid="ready">ok</div>;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Wrapper>
        <Errors.SuspenseBoundary loading={<div>loading</div>}>
          <Display />
        </Errors.SuspenseBoundary>
      </Wrapper>,
    );
  });
  await waitFor(() => {
    expect(within(utils.container).queryByTestId("ready")?.textContent).toBe("ok");
  });
};

const setupHook = async <H extends object>(
  hookFn: (td: Handle, key: schematic.Key) => H,
  sendMock: SendFn = vi.fn<SendFn>(async () => {}),
): Promise<{
  result: { current: H & { store: Schematic.FluxSubStore } };
  send: SendFn;
  td: Handle;
  key: schematic.Key;
}> => {
  const Wrapper = await createAsyncSynnaxWrapper({ client });
  const schem = await createSchem();
  await primeCache(Wrapper, schem.key);
  const td = makeDispatch(sendMock);
  const { result } = renderHook(
    () => ({
      ...hookFn(td, schem.key),
      store: Flux.useStore<Schematic.FluxSubStore>(),
    }),
    { wrapper: Wrapper },
  );
  return { result, send: sendMock, td, key: schem.key };
};

const getDoc = (
  store: Schematic.FluxSubStore,
  key: schematic.Key,
): schematic.Schematic | undefined => store[Schematic.FLUX_STORE_KEY].get(key);

describe("Flux.createDispatch", () => {
  describe("dispatch", () => {
    it("applies locally, sends, and pushes onto the undo stack", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = await setupHook(
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
      expect(getDoc(result.current.store, key)?.nodes[0].position).toEqual({
        x: 99,
        y: 99,
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
    });

    it("does not push non-undoable actions onto the stack", async () => {
      const { result, key } = await setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodeMeasured({
            key: "n1",
            measured: { width: 1, height: 1 },
          }),
        });
      });
      expect(result.current.undo.canUndo).toBe(false);
    });

    it("rolls back local state when send fails", async () => {
      const send = vi.fn<SendFn>(async () => {
        throw new Error("send failed");
      });
      const { result, key } = await setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      const before = getDoc(result.current.store, key);
      let ok = true;
      await act(async () => {
        ok = await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 99, y: 99 } }),
        });
      });
      expect(ok).toBe(false);
      expect(getDoc(result.current.store, key)).toEqual(before);
      expect(result.current.undo.canUndo).toBe(false);
    });

    it("sends an array of actions in a single send call", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = await setupHook(
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
      expect(send.mock.calls[0][0].actions).toHaveLength(2);
      const doc = getDoc(result.current.store, key);
      expect(doc?.nodes[0].position).toEqual({ x: 1, y: 1 });
      expect(doc?.nodes[1].position).toEqual({ x: 2, y: 2 });
    });

    it("returns true without sending when given an empty actions array", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = await setupHook(
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

    it("returns false from dispatchAsync when the doc is not cached", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const td = makeDispatch(send);
      // Don't prime the cache — dispatch on an unknown key.
      const { result } = renderHook(() => td.useDispatch(), { wrapper: Wrapper });
      let ok = true;
      await act(async () => {
        ok = await result.current.dispatchAsync({
          key: "unknown",
          actions: schematic.setNodePosition({ key: "n", position: { x: 0, y: 0 } }),
        });
      });
      expect(ok).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("dispatch (sync) fires the action without awaiting", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = await setupHook(
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
        expect(getDoc(result.current.store, key)?.nodes[0].position).toEqual({
          x: 4,
          y: 4,
        });
      });
    });
  });

  describe("undo", () => {
    it("reverts the last user dispatch and pushes a redo entry", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = await setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
          redo: td.useRedo({ key: k }),
        }),
        send,
      );
      const before = getDoc(result.current.store, key);
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
        expect(getDoc(result.current.store, key)?.nodes[0].position).toEqual(
          before?.nodes[0].position,
        );
        expect(result.current.redo.canRedo).toBe(true);
      });
    });

    it("returns the entry to the undo stack when the inverse send fails", async () => {
      let calls = 0;
      const send = vi.fn<SendFn>(async () => {
        if (++calls === 2) throw new Error("inverse send failed");
      });
      const { result, key } = await setupHook(
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
      const after = getDoc(result.current.store, key);
      act(() => result.current.undo.undo());
      await waitFor(() => expect(send).toHaveBeenCalledTimes(2));
      expect(getDoc(result.current.store, key)).toEqual(after);
      expect(result.current.undo.canUndo).toBe(true);
      expect(result.current.redo.canRedo).toBe(false);
    });

    it("auto-advances past entries invalidated by remote touches", async () => {
      const { result, key } = await setupHook((td, k) => ({
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
        result.current.store[Schematic.FLUX_STORE_KEY].markRemoteTouched(
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
      const { result, key } = await setupHook((td, k) => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: k }),
        redo: td.useRedo({ key: k }),
      }));
      const before = getDoc(result.current.store, key);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } }),
        });
      });
      const afterDispatch = getDoc(result.current.store, key);
      act(() => result.current.undo.undo());
      await waitFor(() => {
        expect(getDoc(result.current.store, key)?.nodes[0].position).toEqual(
          before?.nodes[0].position,
        );
        expect(result.current.redo.canRedo).toBe(true);
      });
      act(() => result.current.redo.redo());
      // Redo must restore the post-dispatch state, not leave it at the
      // undone state. (This caught the swap bug in prepareUndo/prepareRedo.)
      await waitFor(() => {
        expect(getDoc(result.current.store, key)?.nodes[0].position).toEqual(
          afterDispatch?.nodes[0].position,
        );
        expect(result.current.undo.canUndo).toBe(true);
      });
    });

    it("clears the redo stack on a new user dispatch", async () => {
      const { result, key } = await setupHook((td, k) => ({
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
      const { result, key } = await setupHook(
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
        result.current.store[Schematic.FLUX_STORE_KEY].markRemoteTouched(
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
      const { result, key } = await setupHook(
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
      const undoneState = getDoc(result.current.store, key);
      act(() => result.current.redo.redo());
      await waitFor(() => expect(send).toHaveBeenCalledTimes(3));
      expect(getDoc(result.current.store, key)).toEqual(undoneState);
      expect(result.current.redo.canRedo).toBe(true);
    });
  });

  describe("coalescing", () => {
    it("merges same-kind dispatches inside the coalesce window", async () => {
      const { result, key } = await setupHook((td, k) => ({
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
      const { result, key, send } = await setupHook((td, k) => ({
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
          actions: schematic.setConfig({ key: "n1", config: { label: "x" } }),
        });
      });
      // Two distinct entries: one undo leaves canUndo true, a second clears it.
      act(() => result.current.undo.undo());
      await waitFor(() => expect(send).toHaveBeenCalledTimes(3));
      expect(result.current.undo.canUndo).toBe(true);
      act(() => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });

    it("restores state when undoing two coalesced setConfig dispatches", async () => {
      const { result, key } = await setupHook((td, k) => ({
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
      const baseline = getDoc(result.current.store, key)?.configs;
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
      await waitFor(() =>
        expect(getDoc(result.current.store, key)?.configs).toEqual(baseline),
      );
    });
  });

  describe("transactions", () => {
    it("commits accumulated actions as one undoable", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result, key } = await setupHook(
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
      const { result, key } = await setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      const initial = getDoc(result.current.store, key);
      act(() => {
        const tx = result.current.dispatch.beginTransaction({ key, kind: "move" });
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 9, y: 9 } })]);
        tx.abort();
      });
      expect(send).not.toHaveBeenCalled();
      expect(result.current.undo.canUndo).toBe(false);
      expect(getDoc(result.current.store, key)).toEqual(initial);
    });

    it("restores the pre-transaction snapshot when commit's send fails", async () => {
      const send = vi.fn<SendFn>(async () => {
        throw new Error("commit failed");
      });
      const { result, key } = await setupHook(
        (td, k) => ({
          dispatch: td.useDispatch(),
          undo: td.useUndo({ key: k }),
        }),
        send,
      );
      const initial = getDoc(result.current.store, key);
      await act(async () => {
        const tx = result.current.dispatch.beginTransaction({ key, kind: "move" });
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } })]);
        tx.add([schematic.setNodePosition({ key: "n1", position: { x: 2, y: 2 } })]);
        await tx.commit();
      });
      expect(getDoc(result.current.store, key)).toEqual(initial);
      expect(result.current.undo.canUndo).toBe(false);
    });
  });

  describe("cascade", () => {
    it("drops undo state when the document is deleted", async () => {
      const { result, key } = await setupHook((td, k) => ({
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
        result.current.store[Schematic.FLUX_STORE_KEY].delete(key);
      });
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
      expect(getDoc(result.current.store, key)).toBeUndefined();
    });
  });

  describe("empty stacks", () => {
    it("undo is a no-op when nothing has been dispatched", async () => {
      const send = vi.fn<SendFn>(async () => {});
      const { result } = await setupHook(
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
      const { result } = await setupHook(
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
      const Wrapper = await createAsyncSynnaxWrapper({ client });
      const a = await createSchem();
      const b = await createSchem();
      await primeCache(Wrapper, a.key);
      await primeCache(Wrapper, b.key);
      const send = vi.fn<SendFn>(async () => {});
      const td = makeDispatch(send);
      const { result } = renderHook(
        () => ({
          dispatch: td.useDispatch(),
          undoA: td.useUndo({ key: a.key }),
          undoB: td.useUndo({ key: b.key }),
        }),
        { wrapper: Wrapper },
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: a.key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 5, y: 5 } }),
        });
      });
      await waitFor(() => expect(result.current.undoA.canUndo).toBe(true));
      // Dispatching on `a` must not enable undo on `b`.
      expect(result.current.undoB.canUndo).toBe(false);
    });
  });
});
