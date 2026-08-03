// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  lineplot,
  type project,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { LinePlot } from "@/platform/lineplot";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab } from "@/testutil";

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
  await client.projects.create({
    name: `proj_${id.create().replace(/-/g, "_")}`,
    layout: {},
  });

const renderCreate = (
  harness: Awaited<ReturnType<typeof buildHarness>>,
  props: Parameters<typeof LinePlot.useCreate>[0] = {},
) => renderHook(() => LinePlot.useCreate(props), { wrapper: harness.wrapper });

const expectParent = async (key: string, projectKey: project.Key) =>
  await waitFor(async () => {
    const parents = await client.ontology.retrieveParents(lineplot.ontologyID(key));
    expect(
      parents.some((p) => p.id.type === "project" && p.id.key === projectKey),
    ).toBe(true);
  });

describe("lineplot useCreate", () => {
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
      await expectParent(key, projectB.key);
    });

    it("falls back to the active project when no prop is given", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const key = uuid.create();
      await act(async () => {
        result.current({ key, name: "ActiveProject" });
      });
      await expectParent(key, projectA.key);
    });

    it("does not change the active project when creating in another one", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness, { project: projectB.key });
      const key = uuid.create();
      await act(async () => {
        result.current({ key, name: "OtherProject" });
      });
      await expectParent(key, projectB.key);
      expect(Session.Project.selectSelected(harness.store.getState())).toEqual(
        projectA.key,
      );
    });
  });

  describe("without a project", () => {
    it("throws when no params, prop, or active project is available", async () => {
      const harness = await buildHarness();
      const { result } = renderCreate(harness);
      expect(() => result.current({ key: uuid.create() })).toThrow(UnexpectedError);
    });
  });

  describe("tab management", () => {
    it("opens a resource tab for the created line plot", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const key = uuid.create();
      await act(async () => {
        result.current({ key, name: "Opened" });
      });
      const tab = await resolveFocusedTab(
        harness.store,
        client,
        (t) => t.variant === "resource" && t.resource.key === key,
      );
      expect(tab.variant).toEqual("resource");
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.type).toEqual(lineplot.ontologyID(key).type);
    });
  });

  describe("init defaults", () => {
    it("defaults the line plot name to 'Line Plot' when init omits one", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const key = uuid.create();
      await act(async () => {
        result.current({ key });
      });
      await waitFor(async () =>
        expect((await client.lineplots.retrieve({ key })).name).toEqual("Line Plot"),
      );
    });

    it("uses the caller-provided key for the server line plot", async () => {
      const harness = await buildHarness({ activeProject: projectA });
      const { result } = renderCreate(harness);
      const callerKey = uuid.create();
      await act(async () => {
        result.current({ key: callerKey, name: "WithKey" });
      });
      const retrieved = await waitFor(
        async () => await client.lineplots.retrieve({ key: callerKey }),
      );
      expect(retrieved.key).toEqual(callerKey);
      expect(retrieved.name).toEqual("WithKey");
    });
  });
});
