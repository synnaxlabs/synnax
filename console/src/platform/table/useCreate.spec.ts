// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type project, type Synnax } from "@synnaxlabs/client";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Table } from "@/platform/table";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const client: Synnax = createTestClient();

interface BuildHarnessArgs {
  activeProject?: project.Project;
}

const buildHarness = async ({ activeProject }: BuildHarnessArgs = {}) =>
  await createConsoleWrapper({
    client,
    preloadedState: {
      [Session.Project.SLICE_NAME]: {
        ...Session.Project.ZERO_SLICE_STATE,
        selected: activeProject?.key,
      },
    },
  });

const newProject = async (): Promise<project.Project> =>
  await client.projects.create({
    name: `proj-${id.create()}`,
    layout: Session.Layout.ZERO_SLICE_STATE,
  });

const findPlacedTableLayout = (store: TestStore) =>
  Session.Layout.selectByFilter(store.getState(), (l) => l.type === Table.LAYOUT_TYPE);

const waitForPlacedLayout = async (store: TestStore): Promise<string> => {
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
      const { result } = renderHook(() => Table.useCreate({ project: projectB.key }), {
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
      const { result } = renderHook(() => Table.useCreate({}), { wrapper });
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
      const { result } = renderHook(() => Table.useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "Editable" });
      });
      const placedKey = await waitForPlacedLayout(store);
      const state = store.getState();
      expect(Session.Table.selectEditable({ state, key: placedKey })).toBe(true);
      expect(Session.Layout.select(state, placedKey)?.name).toEqual("Editable");
      expect(Session.Layout.selectType(state, placedKey)).toEqual(Table.LAYOUT_TYPE);
    });

    it("defaults the layout name to 'Table' when init does not provide one", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => Table.useCreate({}), { wrapper });
      await act(async () => {
        result.current();
      });
      const placedKey = await waitForPlacedLayout(store);
      expect(Session.Layout.select(store.getState(), placedKey)?.name).toEqual("Table");
    });

    it("uses the caller-provided key for both the server table and the layout", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => Table.useCreate({}), { wrapper });
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
      const { result } = renderHook(() => Table.useCreate({ project: projectA.key }), {
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
