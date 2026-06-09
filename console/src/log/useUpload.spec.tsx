// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  combineReducers,
  configureStore,
  type Reducer,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { createTestClient, log, type Synnax, type workspace } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Flux, Pluto, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { color, id, uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { Log } from "@/log";
import { LAYOUT_TYPE } from "@/log/layout";
import { type PendingUpload } from "@/log/types";
import { useAutoUpload } from "@/log/useUpload";
import { Workspace } from "@/workspace";

const client: Synnax = createTestClient();

interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Log.SLICE_NAME]: Log.SliceState;
  [Workspace.SLICE_NAME]: Workspace.SliceState;
}

const rootReducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [Log.SLICE_NAME]: Log.reducer,
  [Workspace.SLICE_NAME]: Workspace.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

type RootStore = ReturnType<typeof configureStore<RootState>>;

interface SeedArgs {
  logKey: string;
  layoutName: string;
  pendingUpload?: PendingUpload;
  activeWorkspace?: workspace.Workspace;
}

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: RootStore;
}

const stripLayout = (ws: workspace.Workspace): Omit<workspace.Workspace, "layout"> => {
  const { layout: _layout, ...rest } = ws;
  return rest;
};

const buildHarness = async ({
  logKey,
  layoutName,
  pendingUpload,
  activeWorkspace,
}: SeedArgs): Promise<Harness> => {
  const fluxClient = new Flux.Client({
    client,
    storeConfig: Pluto.FLUX_STORE_CONFIG,
    handleError: () => {},
    handleAsyncError: async () => {},
  });
  await fluxClient.awaitInitialized();
  const preloadedState: RootState = {
    [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
    [Layout.SLICE_NAME]: {
      ...Layout.ZERO_SLICE_STATE,
      layouts: {
        ...Layout.ZERO_SLICE_STATE.layouts,
        [logKey]: {
          windowKey: "main",
          key: logKey,
          type: LAYOUT_TYPE,
          name: layoutName,
          location: "mosaic",
        },
      },
    },
    [Log.SLICE_NAME]: {
      ...Log.ZERO_SLICE_STATE,
      logs: { [logKey]: { ...Log.ZERO_STATE, key: logKey, pendingUpload } },
    },
    [Workspace.SLICE_NAME]: {
      ...Workspace.ZERO_SLICE_STATE,
      active: activeWorkspace != null ? stripLayout(activeWorkspace) : null,
    },
  };
  const store = configureStore({ reducer: rootReducer, preloadedState });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <Status.Aggregator>
        <PSynnax.TestProvider client={client}>
          <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
        </PSynnax.TestProvider>
      </Status.Aggregator>
    </Provider>
  );
  return { wrapper: Wrapper, store };
};

const buildPendingUpload = (override: Partial<PendingUpload> = {}): PendingUpload =>
  log.logZ.omit({ name: true }).parse({
    key: override.key ?? uuid.create(),
    channels: [log.channelEntryZ.parse({ channel: 1, color: color.ZERO })],
    timestampPrecision: 2,
    ...override,
  });

const pendingOf = (store: RootStore, key: string): PendingUpload | undefined =>
  store.getState()[Log.SLICE_NAME].logs[key].pendingUpload;

describe("useAutoUpload", () => {
  let workspaceA: workspace.Workspace;

  beforeEach(async () => {
    workspaceA = await client.workspaces.create({
      name: `ws-${id.create()}`,
      layout: {},
    });
  });

  it("returns true when there is no pending upload and does not call the server", async () => {
    const logKey = uuid.create();
    const { wrapper } = await buildHarness({
      logKey,
      layoutName: "Already Uploaded",
      activeWorkspace: workspaceA,
    });
    const { result } = renderHook(() => useAutoUpload(logKey), { wrapper });
    expect(result.current).toBe(true);
    await expect(client.logs.retrieve({ key: logKey })).rejects.toThrow();
  });

  it("uploads the pending payload using the layout name, then returns true", async () => {
    const logKey = uuid.create();
    const pendingUpload = buildPendingUpload({ key: logKey });
    const { wrapper, store } = await buildHarness({
      logKey,
      layoutName: "Live Layout Name",
      pendingUpload,
      activeWorkspace: workspaceA,
    });
    const { result } = renderHook(() => useAutoUpload(logKey), { wrapper });
    expect(result.current).toBe(false);
    await waitFor(() => expect(result.current).toBe(true));
    const uploaded = await client.logs.retrieve({ key: logKey });
    expect(uploaded.name).toEqual("Live Layout Name");
    expect(uploaded.channels.map((e) => e.channel)).toEqual([1]);
    expect(uploaded.timestampPrecision).toEqual(2);
    expect(pendingOf(store, logKey)).toBeUndefined();
  });

  it("uploads without a workspace when none is active", async () => {
    const logKey = uuid.create();
    const pendingUpload = buildPendingUpload({ key: logKey });
    const { wrapper, store } = await buildHarness({
      logKey,
      layoutName: "Workspaceless",
      pendingUpload,
    });
    const { result } = renderHook(() => useAutoUpload(logKey), { wrapper });
    await waitFor(() => expect(result.current).toBe(true));
    const uploaded = await client.logs.retrieve({ key: logKey });
    expect(uploaded.name).toEqual("Workspaceless");
    expect(uploaded.channels.map((e) => e.channel)).toEqual([1]);
    expect(pendingOf(store, logKey)).toBeUndefined();
  });

  it("does not re-upload on subsequent renders once pendingUpload is cleared", async () => {
    const logKey = uuid.create();
    const pendingUpload = buildPendingUpload({ key: logKey });
    const { wrapper, store } = await buildHarness({
      logKey,
      layoutName: "Single Upload",
      pendingUpload,
      activeWorkspace: workspaceA,
    });
    const { result, rerender } = renderHook(() => useAutoUpload(logKey), { wrapper });
    await waitFor(() => expect(result.current).toBe(true));
    const first = await client.logs.retrieve({ key: logKey });
    rerender();
    rerender();
    expect(pendingOf(store, logKey)).toBeUndefined();
    const second = await client.logs.retrieve({ key: logKey });
    expect(second).toEqual(first);
  });
});
