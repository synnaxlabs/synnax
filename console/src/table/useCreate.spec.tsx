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
import { createTestClient, type Synnax, type workspace } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Flux, Pluto, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { Table } from "@/table";
import { LAYOUT_TYPE } from "@/table/layout";
import { selectEditable } from "@/table/selectors";
import { useCreate } from "@/table/useCreate";
import { Workspace } from "@/workspace";

const client: Synnax = createTestClient();

interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Workspace.SLICE_NAME]: Workspace.SliceState;
}

const rootReducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Workspace.SLICE_NAME]: Workspace.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

type RootStore = ReturnType<typeof configureStore<RootState>>;

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: RootStore;
}

interface BuildHarnessArgs {
  activeWorkspace?: workspace.Workspace;
}

const stripLayout = (ws: workspace.Workspace): Omit<workspace.Workspace, "layout"> => {
  const { layout: _, ...rest } = ws;
  return rest;
};

const buildHarness = async ({
  activeWorkspace,
}: BuildHarnessArgs = {}): Promise<Harness> => {
  const fluxClient = new Flux.Client({
    client,
    storeConfig: Pluto.FLUX_STORE_CONFIG,
    handleError: () => {},
    handleAsyncError: async () => {},
  });
  await fluxClient.awaitInitialized();
  const preloadedState: RootState = {
    [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
    [Layout.SLICE_NAME]: Layout.ZERO_SLICE_STATE,
    [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
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

const newWorkspace = async (): Promise<workspace.Workspace> =>
  await client.workspaces.create({ name: `ws-${id.create()}`, layout: {} });

const findPlacedTableLayout = (store: RootStore) =>
  Object.values(store.getState()[Layout.SLICE_NAME].layouts).find(
    (l) => l.type === LAYOUT_TYPE,
  );

const waitForPlacedLayout = async (store: RootStore): Promise<string> => {
  let key: string | undefined;
  await waitFor(() => {
    const placed = findPlacedTableLayout(store);
    expect(placed).toBeDefined();
    key = placed!.key;
  });
  return key!;
};

describe("useCreate", () => {
  let workspaceA: workspace.Workspace;
  let workspaceB: workspace.Workspace;

  beforeEach(async () => {
    workspaceA = await newWorkspace();
    workspaceB = await newWorkspace();
  });

  describe("workspace resolution", () => {
    it("prefers the prop workspace over the active workspace", async () => {
      const { wrapper, store } = await buildHarness({ activeWorkspace: workspaceA });
      const { result } = renderHook(() => useCreate({ workspace: workspaceB.key }), {
        wrapper,
      });
      await act(async () => {
        result.current({ name: "ProvidedWS" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ProvidedWS");
      expect(store.getState()[Workspace.SLICE_NAME].active?.key).toEqual(
        workspaceB.key,
      );
    });

    it("falls back to the active workspace when no prop is given", async () => {
      const { wrapper, store } = await buildHarness({ activeWorkspace: workspaceA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "ActiveWS" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ActiveWS");
      expect(store.getState()[Workspace.SLICE_NAME].active?.key).toEqual(
        workspaceA.key,
      );
    });

    it("creates a workspace-less table when neither prop nor active workspace is set", async () => {
      const { wrapper, store } = await buildHarness();
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "Loose" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("Loose");
      expect(store.getState()[Workspace.SLICE_NAME].active).toBeNull();
    });
  });

  describe("layout placement", () => {
    it("places the layout with editable=true after the server returns", async () => {
      const { wrapper, store } = await buildHarness({ activeWorkspace: workspaceA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "Editable" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const state = store.getState();
      expect(selectEditable(state, placedKey)).toBe(true);
      expect(state[Layout.SLICE_NAME].layouts[placedKey].name).toEqual("Editable");
      expect(state[Layout.SLICE_NAME].layouts[placedKey].type).toEqual(LAYOUT_TYPE);
    });

    it("defaults the layout name to 'Table' when init does not provide one", async () => {
      const { wrapper, store } = await buildHarness({ activeWorkspace: workspaceA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current();
      });
      const placedKey = await waitForPlacedLayout(store);
      expect(store.getState()[Layout.SLICE_NAME].layouts[placedKey].name).toEqual(
        "Table",
      );
    });

    it("uses the caller-provided key for both the server table and the layout", async () => {
      const { wrapper, store } = await buildHarness({ activeWorkspace: workspaceA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      const callerKey = uuid.create();
      await act(async () => {
        result.current({ key: callerKey, name: "WithKey" });
      });
      await waitFor(() => {
        expect(store.getState()[Layout.SLICE_NAME].layouts[callerKey]).toBeDefined();
      });
      const retrieved = await client.tables.retrieve({ key: callerKey });
      expect(retrieved.key).toEqual(callerKey);
      expect(retrieved.name).toEqual("WithKey");
    });
  });

  describe("workspace switching", () => {
    it("does not flip the active workspace when the table is created in the active one", async () => {
      const { wrapper, store } = await buildHarness({ activeWorkspace: workspaceA });
      const beforeActive = store.getState()[Workspace.SLICE_NAME].active;
      const { result } = renderHook(() => useCreate({ workspace: workspaceA.key }), {
        wrapper,
      });
      await act(async () => {
        result.current({ name: "SameWS" });
      });
      await waitForPlacedLayout(store);
      expect(store.getState()[Workspace.SLICE_NAME].active).toBe(beforeActive);
    });
  });
});
