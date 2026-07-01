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
import { createTestClient, type project, type Synnax } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Flux, Pluto, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { LAYOUT_TYPE } from "@/component/table/layout";
import { useCreate } from "@/component/table/useCreate";
import { Session } from "@/session";

const client: Synnax = createTestClient();

interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Session.Layout.SLICE_NAME]: Session.Layout.SliceState;
  [Session.Table.SLICE_NAME]: Session.Table.SliceState;
  [Session.Project.SLICE_NAME]: Session.Project.SliceState;
}

const rootReducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Session.Layout.SLICE_NAME]: Session.Layout.reducer,
  [Session.Table.SLICE_NAME]: Session.Table.reducer,
  [Session.Project.SLICE_NAME]: Session.Project.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

type RootStore = ReturnType<typeof configureStore<RootState>>;

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: RootStore;
}

interface BuildHarnessArgs {
  activeProject?: project.Project;
}

const buildHarness = async ({
  activeProject,
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
    [Session.Layout.SLICE_NAME]: Session.Layout.ZERO_SLICE_STATE,
    [Session.Table.SLICE_NAME]: Session.Table.ZERO_SLICE_STATE,
    [Session.Project.SLICE_NAME]: {
      ...Session.Project.ZERO_SLICE_STATE,
      selected: activeProject?.key,
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

const newProject = async (): Promise<project.Project> =>
  await client.projects.create({ name: `proj-${id.create()}`, layout: {} });

const findPlacedTableLayout = (store: RootStore) =>
  Session.Layout.selectByFilter(store.getState(), (l) => l.type === LAYOUT_TYPE);

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
  let projectA: project.Project;
  let projectB: project.Project;

  beforeEach(async () => {
    projectA = await newProject();
    projectB = await newProject();
  });

  describe("project resolution", () => {
    it("prefers the prop project over the active project", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({ project: projectB.key }), {
        wrapper,
      });
      await act(async () => {
        result.current({ name: "ProvidedProject" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ProvidedProject");
      expect(Session.Project.selectSelected(store.getState())).toEqual(projectB.key);
    });

    it("falls back to the active project when no prop is given", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "ActiveProject" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ActiveProject");
      expect(Session.Project.selectSelected(store.getState())).toEqual(projectA.key);
    });
  });

  describe("layout placement", () => {
    it("places the layout with editable=true after the server returns", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "Editable" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const state = store.getState();
      expect(Session.Table.selectEditable({ state, key: placedKey })).toBe(true);
      expect(Session.Layout.select(state, placedKey)?.name).toEqual("Editable");
      expect(Session.Layout.selectType(state, placedKey)).toEqual(LAYOUT_TYPE);
    });

    it("defaults the layout name to 'Table' when init does not provide one", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current();
      });
      const placedKey = await waitForPlacedLayout(store);
      expect(Session.Layout.select(store.getState(), placedKey)?.name).toEqual("Table");
    });

    it("uses the caller-provided key for both the server table and the layout", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      const callerKey = uuid.create();
      await act(async () => {
        result.current({ key: callerKey, name: "WithKey" });
      });
      await waitFor(() => {
        expect(Session.Layout.select(store.getState(), callerKey)).toBeDefined();
      });
      const retrieved = await client.tables.retrieve({ key: callerKey });
      expect(retrieved.key).toEqual(callerKey);
      expect(retrieved.name).toEqual("WithKey");
    });
  });

  describe("project switching", () => {
    it("does not flip the active project when the table is created in the active one", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const beforeActive = Session.Project.selectSelected(store.getState());
      const { result } = renderHook(() => useCreate({ project: projectA.key }), {
        wrapper,
      });
      await act(async () => {
        result.current({ name: "SameProject" });
      });
      await waitForPlacedLayout(store);
      expect(Session.Project.selectSelected(store.getState())).toBe(beforeActive);
    });
  });
});
