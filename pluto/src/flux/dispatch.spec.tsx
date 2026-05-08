// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, schematic } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

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
    ...schematic.ZERO_NEW,
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
  schematic.Schematic,
  schematic.Action,
  Schematic.FluxSubStore,
  typeof Schematic.FLUX_STORE_KEY
>["send"];

const makeDispatch = (sendMock: SendFn, coalesceMs = 100) =>
  Flux.createDispatch<
    schematic.Key,
    schematic.Schematic,
    schematic.Action,
    Schematic.FluxSubStore,
    typeof Schematic.FLUX_STORE_KEY
  >({
    name: "schematic-test",
    storeKey: Schematic.FLUX_STORE_KEY,
    reduce: schematic.reduceAll,
    send: sendMock,
    isUndoable: schematic.isUndoable,
    kindOf: (actions) => (actions.length === 1 ? actions[0].type : "transaction"),
    coalesceMs,
  });

const primeCache = async (Wrapper: FC<PropsWithChildren>, key: schematic.Key) => {
  const Display = (): ReactElement => {
    Schematic.useEnsureRetrieved({ key });
    return <div data-testid="ready">ok</div>;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Wrapper>
        <Flux.Suspense loading={<div>loading</div>}>
          <Display />
        </Flux.Suspense>
      </Wrapper>,
    );
  });
  await waitFor(() => {
    expect(utils.queryByTestId("ready")?.textContent).toBe("ok");
  });
};

describe("Flux.createDispatch", () => {
  it("dispatches actions, applies them locally, and pushes to the undo stack", async () => {
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    const schem = await createSchem();
    await primeCache(Wrapper, schem.key);

    const send = vi.fn<SendFn>(async () => {});
    const td = makeDispatch(send);

    const { result } = renderHook(
      () => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: schem.key }),
      }),
      { wrapper: Wrapper },
    );

    expect(result.current.undo.canUndo).toBe(false);

    await act(async () => {
      await result.current.dispatch.dispatchAsync({
        key: schem.key,
        actions: schematic.setNodePosition({
          key: "n1",
          position: { x: 99, y: 99 },
        }),
      });
    });

    expect(send).toHaveBeenCalledTimes(1);
    expect(result.current.undo.canUndo).toBe(true);
  });

  it("undoes a user action by dispatching the inverse", async () => {
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    const schem = await createSchem();
    await primeCache(Wrapper, schem.key);

    const send = vi.fn<SendFn>(async () => {});
    const td = makeDispatch(send);

    const { result } = renderHook(
      () => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: schem.key }),
        redo: td.useRedo({ key: schem.key }),
      }),
      { wrapper: Wrapper },
    );

    await act(async () => {
      await result.current.dispatch.dispatchAsync({
        key: schem.key,
        actions: schematic.setNodePosition({
          key: "n1",
          position: { x: 50, y: 60 },
        }),
      });
    });
    expect(send).toHaveBeenCalledTimes(1);
    expect(result.current.undo.canUndo).toBe(true);

    await act(async () => {
      result.current.undo.undo();
      await new Promise((r) => setTimeout(r, 30));
    });

    await waitFor(() => expect(result.current.redo.canRedo).toBe(true));
    expect(send).toHaveBeenCalledTimes(2);
  });

  it("redoes after undo and clears redo on new user action", async () => {
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    const schem = await createSchem();
    await primeCache(Wrapper, schem.key);

    const send = vi.fn<SendFn>(async () => {});
    const td = makeDispatch(send);

    const { result } = renderHook(
      () => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: schem.key }),
        redo: td.useRedo({ key: schem.key }),
      }),
      { wrapper: Wrapper },
    );

    await act(async () => {
      await result.current.dispatch.dispatchAsync({
        key: schem.key,
        actions: schematic.setNodePosition({
          key: "n1",
          position: { x: 1, y: 1 },
        }),
      });
    });

    await act(async () => {
      result.current.undo.undo();
      await new Promise((r) => setTimeout(r, 30));
    });
    await waitFor(() => expect(result.current.redo.canRedo).toBe(true));

    await act(async () => {
      result.current.redo.redo();
      await new Promise((r) => setTimeout(r, 30));
    });
    await waitFor(() => expect(result.current.undo.canUndo).toBe(true));

    await act(async () => {
      await result.current.dispatch.dispatchAsync({
        key: schem.key,
        actions: schematic.setNodePosition({
          key: "n1",
          position: { x: 7, y: 7 },
        }),
      });
    });

    expect(result.current.redo.canRedo).toBe(false);
  });

  it("coalesces same-kind dispatches within the window into one undo entry", async () => {
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    const schem = await createSchem();
    await primeCache(Wrapper, schem.key);

    const send = vi.fn<SendFn>(async () => {});
    const td = makeDispatch(send, /* coalesceMs */ 1000);

    const { result } = renderHook(
      () => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: schem.key }),
      }),
      { wrapper: Wrapper },
    );

    for (const x of [10, 20, 30])
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: schem.key,
          actions: schematic.setNodePosition({
            key: "n1",
            position: { x, y: 0 },
          }),
        });
      });

    expect(result.current.undo.canUndo).toBe(true);

    await act(async () => {
      result.current.undo.undo();
      await new Promise((r) => setTimeout(r, 30));
    });

    await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
  });

  it("commits a transaction as a single undoable unit", async () => {
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    const schem = await createSchem();
    await primeCache(Wrapper, schem.key);

    const send = vi.fn<SendFn>(async () => {});
    const td = makeDispatch(send);

    const { result } = renderHook(
      () => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: schem.key }),
      }),
      { wrapper: Wrapper },
    );

    await act(async () => {
      const tx = result.current.dispatch.beginTransaction({
        key: schem.key,
        kind: "move",
      });
      tx.add([schematic.setNodePosition({ key: "n1", position: { x: 1, y: 1 } })]);
      tx.add([schematic.setNodePosition({ key: "n1", position: { x: 2, y: 2 } })]);
      tx.add([schematic.setNodePosition({ key: "n1", position: { x: 3, y: 3 } })]);
      await tx.commit();
    });

    expect(send).toHaveBeenCalledTimes(1);
    expect(result.current.undo.canUndo).toBe(true);

    await act(async () => {
      result.current.undo.undo();
      await new Promise((r) => setTimeout(r, 30));
    });
    await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
  });

  it("aborts a transaction without sending or pushing", async () => {
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    const schem = await createSchem();
    await primeCache(Wrapper, schem.key);

    const send = vi.fn<SendFn>(async () => {});
    const td = makeDispatch(send);

    const { result } = renderHook(
      () => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: schem.key }),
      }),
      { wrapper: Wrapper },
    );

    act(() => {
      const tx = result.current.dispatch.beginTransaction({
        key: schem.key,
        kind: "move",
      });
      tx.add([schematic.setNodePosition({ key: "n1", position: { x: 9, y: 9 } })]);
      tx.abort();
    });

    expect(send).not.toHaveBeenCalled();
    expect(result.current.undo.canUndo).toBe(false);
  });

  it("auto-advances past stale entries on undo", async () => {
    const Wrapper = await createAsyncSynnaxWrapper({ client });
    const schem = await createSchem();
    await primeCache(Wrapper, schem.key);

    const send = vi.fn<SendFn>(async () => {});
    const td = makeDispatch(send);

    const { result } = renderHook(
      () => ({
        dispatch: td.useDispatch(),
        undo: td.useUndo({ key: schem.key }),
      }),
      { wrapper: Wrapper },
    );

    await act(async () => {
      await result.current.dispatch.dispatchAsync({
        key: schem.key,
        actions: schematic.setNodePosition({
          key: "n1",
          position: { x: 1, y: 1 },
        }),
      });
    });
    await act(async () => {
      await result.current.dispatch.dispatchAsync({
        key: schem.key,
        actions: schematic.setNodePosition({
          key: "n2",
          position: { x: 2, y: 2 },
        }),
      });
    });

    await new Promise((r) => setTimeout(r, 5));
    td.notifyRemoteActions(schem.key, [
      schematic.setNodePosition({ key: "n2", position: { x: 99, y: 99 } }),
    ]);

    const sendCallsBefore = send.mock.calls.length;

    await act(async () => {
      result.current.undo.undo();
      await new Promise((r) => setTimeout(r, 30));
    });

    await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    expect(send.mock.calls.length - sendCallsBefore).toBe(1);
  });
});
