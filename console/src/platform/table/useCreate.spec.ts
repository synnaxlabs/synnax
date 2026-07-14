// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Table } from "@/platform/table";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client: Synnax = createTestClient();

interface BuildHarnessParams {
  activeProject?: project.Project;
}

const buildHarness = async ({ activeProject }: BuildHarnessParams = {}) =>
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
  await client.projects.create({ name: `proj-${id.create()}`, layout: {} });

const renderCreate = (
  harness: Awaited<ReturnType<typeof buildHarness>>,
  props: Parameters<typeof Table.useCreate>[0] = {},
) => renderHook(() => Table.useCreate(props), { wrapper: harness.wrapper });

describe("useCreate", () => {
  let projectA: project.Project;
  let projectB: project.Project;

  beforeEach(async () => {
    projectA = await newProject();
    projectB = await newProject();
  });

  describe("project resolution", () => {
    it("prefers the prop project over the active project", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness, { project: projectB.key });
      const key = uuid.create();
      await act(async () => {
        result.current({ key, name: "ProvidedProject" });
      });
      await waitFor(async () =>
        expect((await client.tables.retrieve({ key })).name).toEqual("ProvidedProject"),
      );
      expect(Session.Project.selectSelected(harness.store.getState())).toEqual(
        projectB.key,
      );
    });

    it("falls back to the active project when no prop is given", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const key = uuid.create();
      await act(async () => {
        result.current({ key, name: "ActiveProject" });
      });
      await waitFor(async () =>
        expect((await client.tables.retrieve({ key })).name).toEqual("ActiveProject"),
      );
      expect(Session.Project.selectSelected(harness.store.getState())).toEqual(
        projectA.key,
      );
    });
  });

  describe("session seeding", () => {
    it("seeds the table session state with editable=true", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const key = uuid.create();
      await act(async () => {
        result.current({ key, name: "Editable" });
      });
      await waitFor(() =>
        expect(
          Session.Table.selectEditable({ state: harness.store.getState(), key }),
        ).toBe(true),
      );
    });

    it("defaults the table name to 'Table' when init omits one", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const key = uuid.create();
      await act(async () => {
        result.current({ key });
      });
      await waitFor(async () =>
        expect((await client.tables.retrieve({ key })).name).toEqual("Table"),
      );
    });

    it("uses the caller-provided key for the server table", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const callerKey = uuid.create();
      await act(async () => {
        result.current({ key: callerKey, name: "WithKey" });
      });
      const retrieved = await waitFor(
        async () => await client.tables.retrieve({ key: callerKey }),
      );
      expect(retrieved.key).toEqual(callerKey);
      expect(retrieved.name).toEqual("WithKey");
    });
  });

  describe("project switching", () => {
    it("does not flip the active project when the table is created in the active one", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const beforeActive = Session.Project.selectSelected(harness.store.getState());
      const { result } = renderCreate(harness, { project: projectA.key });
      const key = uuid.create();
      await act(async () => {
        result.current({ key, name: "SameProject" });
      });
      await waitFor(async () => await client.tables.retrieve({ key }));
      expect(Session.Project.selectSelected(harness.store.getState())).toBe(
        beforeActive,
      );
    });
  });
});
