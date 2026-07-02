// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, log, type project, type Synnax } from "@synnaxlabs/client";
import { Log as PLog } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { Project } from "@/platform/project";
import { activeState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const client: Synnax = createTestClient();

const placedLog = (store: TestStore): Session.Layout.State | undefined =>
  Session.Layout.selectByFilter(store.getState(), (l) => l.type === Log.LAYOUT_TYPE);

const projectParents = async (logKey: string): Promise<string[]> =>
  (await client.ontology.retrieveParents(log.ontologyID(logKey))).map((r) => r.id.key);

describe("createUseCreate", () => {
  let projectA: project.Project;
  let projectB: project.Project;

  beforeEach(async () => {
    projectA = await client.projects.create({
      name: `proj-a-${id.create()}`,
      layout: Session.Layout.ZERO_SLICE_STATE,
    });
    projectB = await client.projects.create({
      name: `proj-b-${id.create()}`,
      layout: Session.Layout.ZERO_SLICE_STATE,
    });
  });

  const buildUseCreate = (
    args?: Partial<
      Parameters<typeof Project.createUseCreate<PLog.CreateParams, log.Log>>[0]
    >,
  ) =>
    Project.createUseCreate<PLog.CreateParams, log.Log>({
      useCreate: PLog.useCreate,
      createSessionState: Log.create,
      toCreateParams: ({ overrides, project }) => ({
        name: "Log",
        ...overrides,
        project,
      }),
      ...args,
    });

  it("creates the record under the active project with the default name and places its layout", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: activeState(projectA) },
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
    expect(await projectParents(layout.key)).toEqual([projectA.key]);
    // The record was created in the already-active project, so the switch is a
    // no-op and the active project stays put.
    expect(Session.Project.selectSelected(store.getState())).toBe(projectA.key);
  });

  it("prefers caller init over defaults, and passes defaults through for unspecified fields", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: activeState(projectA) },
    });
    const useCreate = buildUseCreate({
      toCreateParams: ({ overrides, project }) => ({
        name: "Default Log",
        hideChannelNames: true,
        ...overrides,
        project,
      }),
    });
    const { result } = renderHook(() => useCreate({}), { wrapper });

    act(() => {
      result.current({ name: "Init Log" });
    });

    await waitFor(() => expect(placedLog(store)).toBeDefined());
    const created = await client.logs.retrieve({ key: placedLog(store)!.key });
    expect(created.name).toBe("Init Log");
    expect(created.hideChannelNames).toBe(true);
  });

  it("creates the record under the project passed to the hook over the active one", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: activeState(projectA) },
    });
    const useCreate = buildUseCreate();
    const { result } = renderHook(() => useCreate({ project: projectB.key }), {
      wrapper,
    });

    act(() => {
      result.current();
    });

    await waitFor(() => expect(placedLog(store)).toBeDefined());
    expect(await projectParents(placedLog(store)!.key)).toEqual([projectB.key]);
  });

  it("switches the active project to the project the record was created in", async () => {
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: activeState(projectA) },
    });
    const useCreate = buildUseCreate();
    const { result } = renderHook(() => useCreate({ project: projectB.key }), {
      wrapper,
    });

    act(() => {
      result.current();
    });

    await waitFor(() =>
      expect(Session.Project.selectSelected(store.getState())).toBe(projectB.key),
    );
  });
});
