// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type EnhancedStore } from "@reduxjs/toolkit";
import { createTestClient, log, type Synnax, type workspace } from "@synnaxlabs/client";
import { Log as PLog } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { Log } from "@/log";
import { createConsoleWrapper } from "@/testUtils";
import { Workspace } from "@/workspace";

const client: Synnax = createTestClient();

const stripLayout = (ws: workspace.Workspace): Workspace.Workspace => {
  const { layout: _, ...rest } = ws;
  return rest;
};

const activeState = (ws: workspace.Workspace): Workspace.SliceState => ({
  ...Workspace.ZERO_SLICE_STATE,
  active: stripLayout(ws),
});

const placedLog = (store: EnhancedStore): Layout.State | undefined =>
  Layout.selectByFilter(store.getState(), (l) => l.type === Log.LAYOUT_TYPE);

const workspaceParents = async (logKey: string): Promise<string[]> =>
  (await client.ontology.retrieveParents(log.ontologyID(logKey))).map((r) => r.id.key);

describe("createUseCreate", () => {
  let workspaceA: workspace.Workspace;
  let workspaceB: workspace.Workspace;

  beforeEach(async () => {
    workspaceA = await client.workspaces.create({
      name: `ws-a-${id.create()}`,
      layout: {},
    });
    workspaceB = await client.workspaces.create({
      name: `ws-b-${id.create()}`,
      layout: {},
    });
  });

  const buildUseCreate = (
    args?: Partial<
      Parameters<typeof Workspace.createUseCreate<PLog.CreateParams, log.Log>>[0]
    >,
  ) =>
    Workspace.createUseCreate<PLog.CreateParams, log.Log>({
      useCreate: PLog.useCreate,
      createSessionState: Log.create,
      toCreateParams: ({ overrides, workspace }) => ({
        name: "Log",
        ...overrides,
        workspace,
      }),
      ...args,
    });

  it("creates the record under the active workspace with the default name and places its layout", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Workspace.SLICE_NAME]: activeState(workspaceA) },
    });
    const useCreate = buildUseCreate();
    const { result } = renderHook(() => useCreate({}), { wrapper });

    act(() => {
      result.current();
    });

    await waitFor(() => expect(placedLog(store)).toBeDefined());
    const layout = placedLog(store)!;
    expect(layout.name).toBe("Log");

    const created = await client.logs.retrieve({ key: layout.key });
    expect(created.name).toBe("Log");
    expect(await workspaceParents(layout.key)).toEqual([workspaceA.key]);
    // The record was created in the already-active workspace, so the switch is a
    // no-op and the active workspace stays put.
    expect(Workspace.selectActiveKey(store.getState())).toBe(workspaceA.key);
  });

  it("prefers caller init over defaults, and passes defaults through for unspecified fields", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Workspace.SLICE_NAME]: activeState(workspaceA) },
    });
    const useCreate = buildUseCreate({
      toCreateParams: ({ overrides, workspace }) => ({
        name: "Default Log",
        showChannelNames: false,
        ...overrides,
        workspace,
      }),
    });
    const { result } = renderHook(() => useCreate({}), { wrapper });

    act(() => {
      result.current({ name: "Init Log" });
    });

    await waitFor(() => expect(placedLog(store)).toBeDefined());
    const created = await client.logs.retrieve({ key: placedLog(store)!.key });
    expect(created.name).toBe("Init Log");
    expect(created.showChannelNames).toBe(false);
  });

  it("creates the record under the workspace passed to the hook over the active one", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Workspace.SLICE_NAME]: activeState(workspaceA) },
    });
    const useCreate = buildUseCreate();
    const { result } = renderHook(() => useCreate({ workspace: workspaceB.key }), {
      wrapper,
    });

    act(() => {
      result.current();
    });

    await waitFor(() => expect(placedLog(store)).toBeDefined());
    expect(await workspaceParents(placedLog(store)!.key)).toEqual([workspaceB.key]);
  });

  it("creates the record outside any workspace when none is active or provided", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const useCreate = buildUseCreate();
    const { result } = renderHook(() => useCreate({}), { wrapper });

    act(() => {
      result.current();
    });

    await waitFor(() => expect(placedLog(store)).toBeDefined());
    expect(await workspaceParents(placedLog(store)!.key)).toEqual([]);
  });

  it("switches the active workspace to the workspace the record was created in", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Workspace.SLICE_NAME]: activeState(workspaceA) },
    });
    const useCreate = buildUseCreate();
    const { result } = renderHook(() => useCreate({ workspace: workspaceB.key }), {
      wrapper,
    });

    act(() => {
      result.current();
    });

    await waitFor(() =>
      expect(Workspace.selectActiveKey(store.getState())).toBe(workspaceB.key),
    );
  });
});
