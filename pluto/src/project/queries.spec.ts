// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, project, schematic } from "@synnaxlabs/client";
import { createProxiedTestClient, createTestClient } from "@synnaxlabs/client/testutil";
import { id, TimeRange } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { Project } from "@/project";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("list", () => {
    it("should keep the matching projects in name order across a rename", async () => {
      const b = await client.projects.create({ name: "b" });
      const a = await client.projects.create({ name: "a" });
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.projects, {
            name: "projects",
            query: { where: { key: { in: [a.key, b.key] } }, order: { field: "name" } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([a.key, b.key]);
      });
      await client.projects.rename(a.key, "c");
      await waitFor(() => expect(result.current.data).toEqual([b.key, a.key]));
      expect(result.current.getItem(a.key)?.name).toEqual("c");
    });
  });

  describe("use", () => {
    it("should retrieve a single project by key", async () => {
      const testProject = await client.projects.create({
        name: "singleProject",
      });

      const { result } = await renderHookSuspended(
        () => Project.use({ key: testProject.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current?.key).toEqual(testProject.key);
      expect(result.current?.name).toEqual("singleProject");
    });

    it("should handle retrieve with valid project key", async () => {
      const project = await client.projects.create({
        name: "validProject",
      });

      const { result } = await renderHookSuspended(
        () => Project.use({ key: project.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current).not.toBeNull();
      expect(result.current?.key).toEqual(project.key);
    });
  });

  describe("useRename", () => {
    it("should correctly rename a project", async () => {
      const proj = await client.projects.create({
        name: `testProject-${id.create()}`,
      });

      const newName = `newName-${id.create()}`;
      const { result } = await renderHookSuspended(
        () => ({
          retrieve: Project.use({ key: proj.key }),
          rename: Project.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.rename.updateAsync({ key: proj.key, name: newName });
      });
      await waitFor(() => expect(result.current.retrieve?.name).toEqual(newName));
    });

    it("should hold the new name in the cache once the rename resolves", async () => {
      const proj = await client.projects.create({
        name: `testProject-${id.create()}`,
      });
      const name = `newName-${id.create()}`;
      const { result } = renderHook(() => Project.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: proj.key, name });
      });
      expect(client.projects.watch(proj.key).getCached()).toMatchObject({ name });
    });
  });

  describe("useForm", () => {
    it("should create a project, then rename it on a second save", async () => {
      const { result } = renderHook(() => Project.useForm({ query: null }), {
        wrapper,
      });
      const name = `formProject-${id.create()}`;
      act(() => result.current.form.set(Project.formPath.name, name));
      await act(async () => {
        await result.current.saveAsync();
      });
      const key = result.current.form.value().key;
      expect(key).toBeDefined();
      expect((await client.projects.retrieve(key!)).name).toEqual(name);
      const newName = `renamedProject-${id.create()}`;
      act(() => result.current.form.set(Project.formPath.name, newName));
      await act(async () => {
        await result.current.saveAsync();
      });
      expect(result.current.form.value().key).toEqual(key);
      await waitFor(async () =>
        expect((await client.projects.retrieve(key!)).name).toEqual(newName),
      );
    });

    it("should rename an existing project", async () => {
      const proj = await client.projects.create({ name: `testProject-${id.create()}` });
      const { result } = await renderHookSuspended(
        () => Project.useForm({ query: { key: proj.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const newName = `newName-${id.create()}`;
      act(() => result.current.form.set(Project.formPath.name, newName));
      await act(async () => {
        await result.current.saveAsync();
      });
      await waitFor(async () =>
        expect((await client.projects.retrieve(proj.key)).name).toEqual(newName),
      );
    });

    it("should save a name that the client cache still holds", async () => {
      const { proxy, client: proxied } = await createProxiedTestClient();
      const proj = await client.projects.create({ name: `testProject-${id.create()}` });
      const { result } = await renderHookSuspended(
        () => Project.useForm({ query: { key: proj.key } }),
        { wrapper: await createAsyncSynnaxWrapper({ client: proxied }) },
      );
      // The cache stops hearing the Core, so it keeps the name it read.
      expect(proxy.blackholeStreams()).toBeGreaterThan(0);
      act(() => result.current.form.set(Project.formPath.name, "renamed"));
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      act(() => result.current.form.set(Project.formPath.name, proj.name));
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      const stored = await createTestClient().projects.retrieve(proj.key);
      expect(stored.name).toEqual(proj.name);
    });
  });

  describe("useDelete", () => {
    it("should correctly delete a project", async () => {
      const proj = await client.projects.create({
        name: "testProject",
      });

      const { result } = renderHook(() => Project.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(proj.key);
      });
      await waitFor(async () => {
        await expect(client.projects.retrieve(proj.key)).rejects.toThrow(NotFoundError);
      });
    });
  });

  describe("retrieveSiblings", () => {
    it("should return siblings filtered by a single type", async () => {
      const proj = await client.projects.create({ name: "single_type_ws" });
      const s1 = await client.schematics.create({
        project: proj.key,
        name: "A Schematic",
      });
      const l1 = await client.logs.create({ project: proj.key, name: "My Log" });
      await client.lineplots.create({ project: proj.key, name: "My Plot" });

      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(s1.key),
        types: ["log"],
      });
      const keys = siblings.map((p) => p.key);
      expect(keys).toContain(l1.key);
      expect(keys).toHaveLength(1);
    });

    it("should return siblings filtered by multiple types", async () => {
      const proj = await client.projects.create({ name: "multi_type_ws" });
      const s1 = await client.schematics.create({
        project: proj.key,
        name: "Source Schematic",
      });
      const lp = await client.lineplots.create({ project: proj.key, name: "A Plot" });
      const t1 = await client.tables.create({ project: proj.key, name: "A Table" });
      const l1 = await client.logs.create({ project: proj.key, name: "A Log" });

      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(s1.key),
        types: ["lineplot", "table"],
      });
      const keys = siblings.map((p) => p.key);
      expect(keys).toContain(lp.key);
      expect(keys).toContain(t1.key);
      expect(keys).not.toContain(l1.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should return all visualization types except the source type", async () => {
      const proj = await client.projects.create({
        name: "all_but_schematic_ws",
      });
      const s1 = await client.schematics.create({
        project: proj.key,
        name: "Current Schematic",
      });
      const s2 = await client.schematics.create({
        project: proj.key,
        name: "Other Schematic",
      });
      const lp = await client.lineplots.create({ project: proj.key, name: "Plot" });
      const t1 = await client.tables.create({ project: proj.key, name: "Table" });
      const l1 = await client.logs.create({ project: proj.key, name: "Log" });

      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(s1.key),
        types: ["lineplot", "table", "log"],
      });
      const keys = siblings.map((p) => p.key);
      expect(keys).toContain(lp.key);
      expect(keys).toContain(t1.key);
      expect(keys).toContain(l1.key);
      expect(keys).not.toContain(s1.key);
      expect(keys).not.toContain(s2.key);
    });

    it("should exclude the source resource from results", async () => {
      const proj = await client.projects.create({ name: "exclude_ws" });
      const s1 = await client.schematics.create({ project: proj.key, name: "Self" });
      const s2 = await client.schematics.create({ project: proj.key, name: "Other" });

      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(s1.key),
        types: ["schematic"],
      });
      const keys = siblings.map((p) => p.key);
      expect(keys).not.toContain(s1.key);
      expect(keys).toContain(s2.key);
    });

    it("should return empty for a snapshot that belongs to a range", async () => {
      const proj = await client.projects.create({ name: "snapshot_ws" });
      const s1 = await client.schematics.create({ project: proj.key, name: "Source" });
      await client.schematics.create({ project: proj.key, name: "Sibling" });
      const rng = await client.ranges.create({
        name: "Snapshot Range",
        timeRange: new TimeRange(1n, 1000n),
      });
      const snap = await client.schematics.copy({
        key: s1.key,
        name: "Snapshot",
        snapshot: true,
        range: rng.key,
      });
      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(snap.key),
        types: ["schematic"],
      });
      expect(siblings).toEqual([]);
    });

    it("should return empty when resourceID is not provided", async () =>
      expect(await Project.retrieveSiblings(client, { types: ["schematic"] })).toEqual(
        [],
      ));

    it("should reject a resource that is not a visualization", async () =>
      await expect(
        Project.retrieveSiblings(client, {
          resourceID: { type: "channel", key: "1" },
          types: ["schematic"],
        }),
      ).rejects.toThrow("a channel does not belong to a project"));

    it("should find siblings inside groups", async () => {
      const proj = await client.projects.create({ name: "grouped_ws" });
      const s1 = await client.schematics.create({
        project: proj.key,
        name: "Top Level",
      });
      const s2 = await client.schematics.create({
        project: proj.key,
        name: "In Group",
      });
      const g = await client.groups.create({
        parent: project.resourceID(proj.key),
        name: "My Group",
      });
      await client.batch.send(client.schematics.setGroupRequest(s2.key, g.key));

      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(s1.key),
        types: ["schematic"],
      });
      const keys = siblings.map((p) => p.key);
      expect(keys).toContain(s2.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should find siblings in deeply nested groups", async () => {
      const proj = await client.projects.create({ name: "deep_nested_ws" });
      const s1 = await client.schematics.create({
        project: proj.key,
        name: "Top Level",
      });
      const s2 = await client.schematics.create({
        project: proj.key,
        name: "Deeply Nested",
      });
      const outerGroup = await client.groups.create({
        parent: project.resourceID(proj.key),
        name: "Outer Group",
      });
      const innerGroup = await client.groups.create({
        parent: group.resourceID(outerGroup.key),
        name: "Inner Group",
      });
      await client.batch.send(
        client.schematics.setGroupRequest(s2.key, innerGroup.key),
      );

      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(s1.key),
        types: ["schematic"],
      });
      const keys = siblings.map((p) => p.key);
      expect(keys).toContain(s2.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should scope results to the source resource's project", async () => {
      const p1 = await client.projects.create({ name: "scope_p_1" });
      const p2 = await client.projects.create({ name: "scope_p_2" });
      const s1 = await client.schematics.create({
        project: p1.key,
        name: "P1 Schematic",
      });
      await client.schematics.create({ project: p2.key, name: "P2 Schematic" });
      const lp1 = await client.lineplots.create({ project: p1.key, name: "P1 Plot" });
      await client.lineplots.create({ project: p2.key, name: "P2 Plot" });

      const siblings = await Project.retrieveSiblings(client, {
        resourceID: schematic.resourceID(s1.key),
        types: ["lineplot"],
      });
      const keys = siblings.map((p) => p.key);
      expect(keys).toContain(lp1.key);
      expect(keys).toHaveLength(1);
    });

    describe("nested group visibility", () => {
      // TestSpace structure:
      //   Schematic A (top level)
      //   Group 1
      //     Schematic B
      //     Group 2
      //       Schematic C
      //       Schematic D
      //       Group E
      //         Schematic E
      //
      // Mirrored TestSpace (separate project, same structure):
      //   Schematic A Mirrored (top level)
      //   Group 1 Mirrored
      //     Schematic B Mirrored
      //     Group 2 Mirrored
      //       Schematic C Mirrored
      //       Schematic D Mirrored
      //       Group E Mirrored
      //         Schematic E Mirrored

      let sA: schematic.Schematic,
        sB: schematic.Schematic,
        sC: schematic.Schematic,
        sD: schematic.Schematic,
        sE: schematic.Schematic;
      let sAm: schematic.Schematic,
        sBm: schematic.Schematic,
        sCm: schematic.Schematic,
        sDm: schematic.Schematic,
        sEm: schematic.Schematic;

      beforeEach(async () => {
        const proj = await client.projects.create({ name: "TestSpace" });
        sA = await client.schematics.create({ project: proj.key, name: "Schematic A" });
        sB = await client.schematics.create({ project: proj.key, name: "Schematic B" });
        sC = await client.schematics.create({ project: proj.key, name: "Schematic C" });
        sD = await client.schematics.create({ project: proj.key, name: "Schematic D" });
        sE = await client.schematics.create({ project: proj.key, name: "Schematic E" });

        const g1 = await client.groups.create({
          parent: project.resourceID(proj.key),
          name: "Group 1",
        });
        await client.batch.send(client.schematics.setGroupRequest(sB.key, g1.key));

        const g2 = await client.groups.create({
          parent: group.resourceID(g1.key),
          name: "Group 2",
        });
        await client.batch.send(client.schematics.setGroupRequest(sC.key, g2.key));
        await client.batch.send(client.schematics.setGroupRequest(sD.key, g2.key));

        const gE = await client.groups.create({
          parent: group.resourceID(g2.key),
          name: "Group E",
        });
        await client.batch.send(client.schematics.setGroupRequest(sE.key, gE.key));

        const mproj = await client.projects.create({
          name: "Mirrored TestSpace",
        });
        sAm = await client.schematics.create({
          project: mproj.key,
          name: "Schematic A Mirrored",
        });
        sBm = await client.schematics.create({
          project: mproj.key,
          name: "Schematic B Mirrored",
        });
        sCm = await client.schematics.create({
          project: mproj.key,
          name: "Schematic C Mirrored",
        });
        sDm = await client.schematics.create({
          project: mproj.key,
          name: "Schematic D Mirrored",
        });
        sEm = await client.schematics.create({
          project: mproj.key,
          name: "Schematic E Mirrored",
        });

        const mg1 = await client.groups.create({
          parent: project.resourceID(mproj.key),
          name: "Group 1 Mirrored",
        });
        await client.batch.send(client.schematics.setGroupRequest(sBm.key, mg1.key));

        const mg2 = await client.groups.create({
          parent: group.resourceID(mg1.key),
          name: "Group 2 Mirrored",
        });
        await client.batch.send(client.schematics.setGroupRequest(sCm.key, mg2.key));
        await client.batch.send(client.schematics.setGroupRequest(sDm.key, mg2.key));

        const mgE = await client.groups.create({
          parent: group.resourceID(mg2.key),
          name: "Group E Mirrored",
        });
        await client.batch.send(client.schematics.setGroupRequest(sEm.key, mgE.key));
      });

      const expectSiblingsFromSource = async (
        source: schematic.Schematic,
        expectedSiblings: schematic.Schematic[],
        unexpectedKeys: string[],
      ): Promise<void> => {
        const siblings = await Project.retrieveSiblings(client, {
          resourceID: schematic.resourceID(source.key),
          types: ["schematic"],
        });
        const keys = siblings.map((p) => p.key);
        for (const s of expectedSiblings) expect(keys).toContain(s.key);
        expect(keys).not.toContain(source.key);
        for (const k of unexpectedKeys) expect(keys).not.toContain(k);
      };

      it("top-level schematic A sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sA,
          [sB, sC, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("grouped schematic B sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sB,
          [sA, sC, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("deeply nested schematic C sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sC,
          [sA, sB, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("deeply nested schematic D sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sD,
          [sA, sB, sC, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("most deeply nested schematic E sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sE,
          [sA, sB, sC, sD],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("mirrored schematic A sees only mirrored schematics", async () => {
        await expectSiblingsFromSource(
          sAm,
          [sBm, sCm, sDm, sEm],
          [sA.key, sB.key, sC.key, sD.key, sE.key],
        );
      });

      it("mirrored deeply nested schematic E sees only mirrored schematics", async () => {
        await expectSiblingsFromSource(
          sEm,
          [sAm, sBm, sCm, sDm],
          [sA.key, sB.key, sC.key, sD.key, sE.key],
        );
      });
    });
  });
});
