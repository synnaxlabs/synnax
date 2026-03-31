// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError, project, workspace } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Project } from "@/project";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of project keys", async () => {
      const p1 = await client.projects.create({ name: "project1" });
      const p2 = await client.projects.create({ name: "project2" });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(p1.key);
      expect(result.current.data).toContain(p2.key);
    });

    it("should get individual projects using getItem", async () => {
      const testProject = await client.projects.create({
        name: "testProject",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrieved = result.current.getItem(testProject.key);
      expect(retrieved?.key).toEqual(testProject.key);
      expect(retrieved?.name).toEqual("testProject");
    });

    it("should handle pagination with limit and offset", async () => {
      for (let i = 0; i < 5; i++)
        await client.projects.create({ name: `paginationProject${i}` });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a project is created", async () => {
      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newProject = await client.projects.create({
        name: "newProject",
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newProject.key);
      });
    });

    it("should update the list when a project is renamed", async () => {
      const testProject = await client.projects.create({
        name: "originalName",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.getItem(testProject.key)?.name).toEqual("originalName");

      await client.projects.rename(testProject.key, "renamedProject");

      await waitFor(() => {
        expect(result.current.getItem(testProject.key)?.name).toEqual("renamedProject");
      });
    });

    it("should remove project from list when deleted", async () => {
      const testProject = await client.projects.create({
        name: "toDeleteProject",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(testProject.key);

      await client.projects.delete(testProject.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testProject.key);
      });
    });

    it("should handle multiple project updates simultaneously", async () => {
      const p1 = await client.projects.create({ name: "multiUpdate1" });
      const p2 = await client.projects.create({ name: "multiUpdate2" });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await Promise.all([
        client.projects.rename(p1.key, "updated1"),
        client.projects.rename(p2.key, "updated2"),
      ]);

      await waitFor(() => {
        expect(result.current.getItem(p1.key)?.name).toEqual("updated1");
        expect(result.current.getItem(p2.key)?.name).toEqual("updated2");
      });
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single project by key", async () => {
      const testProject = await client.projects.create({
        name: "singleProject",
      });

      const { result } = renderHook(
        () => Project.useRetrieve({ key: testProject.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data?.key).toEqual(testProject.key);
      expect(result.current.data?.name).toEqual("singleProject");
    });

    it("should handle retrieve with valid project key", async () => {
      const p = await client.projects.create({ name: "validProject" });

      const { result } = renderHook(() => Project.useRetrieve({ key: p.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.key).toEqual(p.key);
    });
  });

  describe("useRename", () => {
    it("should correctly rename a project", async () => {
      const p = await client.projects.create({
        name: `testProject-${id.create()}`,
      });

      const newName = `newName-${id.create()}`;
      const { result } = renderHook(
        () => ({
          retrieve: Project.useRetrieve({ key: p.key }),
          rename: Project.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.rename.updateAsync({ key: p.key, name: newName });
      });
      await waitFor(() => expect(result.current.retrieve.data?.name).toEqual(newName));
    });
  });

  describe("useRetrieveGroupID", () => {
    it("should correctly retrieve group ID", async () => {
      const { result } = renderHook(() => Project.useRetrieveGroupID({}), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.type).toEqual("group");
        expect(result.current.data?.key).not.toBeFalsy();
      });
    });
  });

  describe("useDelete", () => {
    it("should correctly delete a project", async () => {
      const p = await client.projects.create({ name: "testProject" });

      const { result } = renderHook(() => Project.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(p.key);
      });
      await waitFor(async () => {
        await expect(client.projects.retrieve(p.key)).rejects.toThrow(NotFoundError);
      });
    });
  });

  describe("useListWorkspaces", () => {
    it("should return workspaces that are children of a project", async () => {
      const p = await client.projects.create({ name: "testProject" });
      const ws = await client.workspaces.create({
        name: "testWorkspace",
        layout: {},
      });
      await client.ontology.addChildren(
        project.ontologyID(p.key),
        workspace.ontologyID(ws.key),
      );

      const { result } = renderHook(
        () => Project.useListWorkspaces(),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ parent: p.key });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(ws.key);
    });

    it("should return empty list when project has no workspaces", async () => {
      const p = await client.projects.create({ name: "emptyProject" });

      const { result } = renderHook(
        () => Project.useListWorkspaces(),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ parent: p.key });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(0);
    });
  });
});
