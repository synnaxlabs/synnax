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

import { Schematic } from "@/platform/schematic";
import { Session } from "@/session";
import { createConsoleWrapper, waitForPlacedLayout } from "@/testutil";

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
    name: `proj_${id.create().replace(/-/g, "_")}`,
    layout: Session.Layout.ZERO_SLICE_STATE,
  });

describe("schematic useCreate", () => {
  let projectA: project.Project;
  let projectB: project.Project;

  beforeEach(async () => {
    projectA = await newProject();
    projectB = await newProject();
  });

  describe("project resolution", () => {
    it("prefers the prop project over the active project", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(
        () => Schematic.useCreate({ project: projectB.key }),
        { wrapper },
      );
      await act(async () => {
        result.current({ name: "ProvidedProject" });
      });
      const placedKey = await waitForPlacedLayout(store, Schematic.LAYOUT_TYPE);
      const retrieved = await client.schematics.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ProvidedProject");
      expect(Session.Project.selectSelected(store.getState())).toEqual(projectB.key);
    });

    it("falls back to the active project when no prop is given", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => Schematic.useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "ActiveProject" });
      });
      const placedKey = await waitForPlacedLayout(store, Schematic.LAYOUT_TYPE);
      const retrieved = await client.schematics.retrieve({ key: placedKey });
      expect(retrieved.name).toEqual("ActiveProject");
      expect(Session.Project.selectSelected(store.getState())).toEqual(projectA.key);
    });
  });

  describe("layout placement", () => {
    it("places the layout with editable=true after the server returns", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => Schematic.useCreate({}), { wrapper });
      await act(async () => {
        result.current({ name: "Editable" });
      });
      const placedKey = await waitForPlacedLayout(store, Schematic.LAYOUT_TYPE);
      const state = store.getState();
      expect(Session.Schematic.selectEditable({ state, key: placedKey })).toBe(true);
      expect(Session.Layout.select(state, placedKey)?.name).toEqual("Editable");
      expect(Session.Layout.selectType(state, placedKey)).toEqual(
        Schematic.LAYOUT_TYPE,
      );
    });

    it("defaults the layout name to 'Schematic' when init does not provide one", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => Schematic.useCreate({}), { wrapper });
      await act(async () => {
        result.current();
      });
      const placedKey = await waitForPlacedLayout(store, Schematic.LAYOUT_TYPE);
      expect(Session.Layout.select(store.getState(), placedKey)?.name).toEqual(
        "Schematic",
      );
    });

    it("uses the caller-provided key for both the server schematic and the layout", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const { result } = renderHook(() => Schematic.useCreate({}), { wrapper });
      const callerKey = uuid.create();
      await act(async () => {
        result.current({ key: callerKey, name: "WithKey" });
      });
      await waitFor(() => {
        expect(Session.Layout.select(store.getState(), callerKey)).toBeDefined();
      });
      const retrieved = await client.schematics.retrieve({ key: callerKey });
      expect(retrieved.key).toEqual(callerKey);
      expect(retrieved.name).toEqual("WithKey");
    });
  });

  describe("project switching", () => {
    it("does not flip the active project when created in the active one", async () => {
      const { wrapper, store } = await buildHarness({ activeProject: projectA });
      const beforeActive = Session.Project.selectSelected(store.getState());
      const { result } = renderHook(
        () => Schematic.useCreate({ project: projectA.key }),
        { wrapper },
      );
      await act(async () => {
        result.current({ name: "SameProject" });
      });
      await waitForPlacedLayout(store, Schematic.LAYOUT_TYPE);
      expect(Session.Project.selectSelected(store.getState())).toBe(beforeActive);
    });
  });
});
