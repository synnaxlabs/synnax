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

import { Layout } from "@/layout";
import { Project } from "@/project";
import { Table } from "@/table";
import { LAYOUT_TYPE } from "@/table/layout";
import { selectEditable } from "@/table/selectors";
import { useCreate } from "@/table/useCreate";

const client: Synnax = createTestClient();

interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Project.SLICE_NAME]: Project.SliceState;
}

const rootReducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Project.SLICE_NAME]: Project.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

type RootStore = ReturnType<typeof configureStore<RootState>>;

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: RootStore;
}

interface BuildHarnessArgs {
  activeProject?: project.Project;
}

const stripLayout = (proj: project.Project): Omit<project.Project, "layout"> => {
  const { layout: _, ...rest } = proj;
  return rest;
};

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
    [Layout.SLICE_NAME]: Layout.ZERO_SLICE_STATE,
    [Table.SLICE_NAME]: Table.ZERO_SLICE_STATE,
    [Project.SLICE_NAME]: {
      ...Project.ZERO_SLICE_STATE,
      active: activeProject != null ? stripLayout(activeProject) : null,
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
  Layout.selectByFilter(store.getState(), (l) => l.type === LAYOUT_TYPE);

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
        result.current({ name: "ProvidedWS" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ProvidedWS");
      expect(Project.selectActiveKey(store.getState())).toEqual(projectB.key);
    });

    it("falls back to the active project when no prop is given", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "ActiveWS" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ActiveWS");
      expect(Project.selectActiveKey(store.getState())).toEqual(projectA.key);
    });

    it("creates a project-less table when neither prop nor active project is set", async () => {
      const { wrapper, store } = await buildHarness();
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "Loose" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const retrieved = await client.tables.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("Loose");
      expect(Project.selectActive(store.getState())).toBeNull();
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
      expect(selectEditable(state, placedKey)).toBe(true);
      expect(Layout.select(state, placedKey)?.name).toEqual("Editable");
      expect(Layout.selectType(state, placedKey)).toEqual(LAYOUT_TYPE);
    });

    it("defaults the layout name to 'Table' when init does not provide one", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      await act(async () => {
        result.current();
      });
      const placedKey = await waitForPlacedLayout(store);
      expect(Layout.select(store.getState(), placedKey)?.name).toEqual("Table");
    });

    it("uses the caller-provided key for both the server table and the layout", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => useCreate({}), { wrapper });
      const callerKey = uuid.create();
      await act(async () => {
        result.current({ key: callerKey, name: "WithKey" });
      });
      await waitFor(() => {
        expect(Layout.select(store.getState(), callerKey)).toBeDefined();
      });
      const retrieved = await client.tables.retrieve({ key: callerKey });
      expect(retrieved.key).toEqual(callerKey);
      expect(retrieved.name).toEqual("WithKey");
    });
  });

  describe("project switching", () => {
    it("does not flip the active project when the table is created in the active one", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const beforeActive = Project.selectActive(store.getState());
      const { result } = renderHook(() => useCreate({ project: projectA.key }), {
        wrapper,
      });
      await act(async () => {
        result.current({ name: "SameWS" });
      });
      await waitForPlacedLayout(store);
      expect(Project.selectActive(store.getState())).toBe(beforeActive);
    });
  });
});
