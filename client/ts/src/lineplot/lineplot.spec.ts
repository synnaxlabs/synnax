// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test, vi } from "vitest";

import { NotFoundError } from "@/index";
import { lineplot } from "@/lineplot";
import { project } from "@/project";
import { query } from "@/query";
import { createTestClient, expectDeleted, expectLive } from "@/testutil";

const client = createTestClient();

describe("LinePlot", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create({
        project: proj.key,
        name: "Line Plot",
      });
      expect(linePlot.name).toEqual("Line Plot");
      expect(linePlot.key).not.toEqual(uuid.ZERO);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create({
        project: proj.key,
        name: "Line Plot",
      });
      await client.lineplots.rename(linePlot.key, "Line Plot2");
      const res = await client.lineplots.retrieve(linePlot.key);
      expect(res.name).toEqual("Line Plot2");
    });
  });
  describe("send", () => {
    test("apply actions without an undo entry", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const linePlot = await client.lineplots.create({
        project: proj.key,
        name: "Line Plot",
      });
      await client.lineplots.send(
        linePlot.key,
        lineplot.set({ path: ["name"], value: "Sent" }),
      );
      const res = await client.lineplots.retrieve(linePlot.key);
      expect(res.name).toEqual("Sent");
      expect(client.lineplots.hasUndo(linePlot.key)).toBe(false);
    });
  });
  describe("project", () => {
    test("retrieve the lineplots of a project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const other = await client.projects.create({ name: "Line Plot" });
      const v = await client.lineplots.create({ project: proj.key, name: "Line Plot" });
      expect(v.project).toEqual(proj.key);
      const owned = await client.lineplots.retrieve({ projects: [proj.key] });
      expect(owned.map(({ key }) => key)).toEqual([v.key]);
      const unowned = await client.lineplots.retrieve({ projects: [other.key] });
      expect(unowned).toEqual([]);
    });

    test("move to another project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const dest = await client.projects.create({ name: "Line Plot" });
      const v = await client.lineplots.create({ project: proj.key, name: "Line Plot" });
      await client.batch.send(client.lineplots.setProjectRequest(v.key, dest.key));
      const res = await client.lineplots.retrieve([v.key]);
      expect(res[0].project).toEqual(dest.key);
    });

    test("move out of its group into another project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const dest = await client.projects.create({ name: "Line Plot" });
      const g = await client.groups.create({
        parent: project.resourceID(proj.key),
        name: "Line Plot",
      });
      const v = await client.lineplots.create({ project: proj.key, name: "Line Plot" });
      await client.batch.send(client.lineplots.setGroupRequest(v.key, g.key));
      await client.batch.send(client.lineplots.setProjectRequest(v.key, dest.key));
      const [res] = await client.lineplots.retrieve({ projects: [dest.key] });
      expect(res.key).toEqual(v.key);
      expect(res.group).toBeUndefined();
    });

    test("move out of its group into the top of its project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const g = await client.groups.create({
        parent: project.resourceID(proj.key),
        name: "Line Plot",
      });
      const v = await client.lineplots.create({ project: proj.key, name: "Line Plot" });
      await client.batch.send(client.lineplots.setGroupRequest(v.key, g.key));
      await client.batch.send(client.lineplots.setProjectRequest(v.key, proj.key));
      const [res] = await client.lineplots.retrieve([v.key]);
      expect(res.project).toEqual(proj.key);
      expect(res.group).toBeUndefined();
    });

    test("move into the project of its group", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const dest = await client.projects.create({ name: "Line Plot" });
      const g = await client.groups.create({
        parent: project.resourceID(dest.key),
        name: "Line Plot",
      });
      const v = await client.lineplots.create({ project: proj.key, name: "Line Plot" });
      await client.batch.send(client.lineplots.setGroupRequest(v.key, g.key));
      const [res] = await client.lineplots.retrieve({ projects: [dest.key] });
      expect(res.key).toEqual(v.key);
      expect(res.group).toEqual(g.key);
    });

    test("delete with its project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const v = await client.lineplots.create({ project: proj.key, name: "Line Plot" });
      await client.projects.delete(proj.key);
      await vi.waitFor(
        async () =>
          await expect(client.lineplots.retrieve(v.key)).rejects.toThrow(NotFoundError),
      );
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create({
        project: proj.key,
        name: "Line Plot",
      });
      await client.lineplots.delete(linePlot.key);
      await expect(client.lineplots.retrieve(linePlot.key)).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});

const seedPlot = async (): Promise<lineplot.LinePlot> => {
  const proj = await client.projects.create({ name: `lp-${id.create()}` });
  return await client.lineplots.create({
    project: proj.key,
    name: `plot-${id.create()}`,
  });
};

const cachedName = (key: lineplot.LinePlot["key"]): string | undefined => {
  const cached = client.lineplots.watch(key).getCached();
  return query.isLive(cached) ? cached.name : undefined;
};

describe("store", () => {
  it("reduces a group move from the Core into the cached document", async () => {
    await client.connect();
    const proj = await client.projects.create({ name: "Line Plot" });
    const plot = await client.lineplots.create({
      project: proj.key,
      name: "Line Plot",
    });
    const g = await client.groups.create({
      parent: project.resourceID(proj.key),
      name: "Line Plot",
    });
    await client.batch.send(client.lineplots.setGroupRequest(plot.key, g.key));
    await expect
      .poll(() => expectLive(client.lineplots.watch(plot.key).getCached()).group)
      .toBe(g.key);
  });

  it("caches the project of a group move as soon as the batch resolves", async () => {
    // A client with no change stream gets no echo of the move.
    const unstreamed = createTestClient({ cache: false });
    const proj = await unstreamed.projects.create({ name: "Line Plot" });
    const dest = await unstreamed.projects.create({ name: "Line Plot" });
    const g = await unstreamed.groups.create({
      parent: project.resourceID(dest.key),
      name: "Line Plot",
    });
    const plot = await unstreamed.lineplots.create({
      project: proj.key,
      name: "Line Plot",
    });
    await unstreamed.batch.send(unstreamed.lineplots.setGroupRequest(plot.key, g.key));
    const cached = expectLive(unstreamed.lineplots.watch(plot.key).getCached());
    expect(cached.group).toEqual(g.key);
    expect(cached.project).toEqual(dest.key);
  });

  it("tombstones deletes from live delete signals", async () => {
    await client.connect();
    const plot = await seedPlot();
    await client.lineplots.delete(plot.key);
    await expect
      .poll(() => query.Deleted.matches(client.lineplots.watch(plot.key).getCached()))
      .toBe(true);
    const cached = expectDeleted(client.lineplots.watch(plot.key).getCached());
    expect(cached.corpse.name).toEqual(plot.name);
  });

  // Regression: a document created by another client must materialize in this
  // client's cache from the create broadcast alone, with no retrieve. A panel
  // tab referencing it can otherwise render before the doc is fetchable.
  it("ingests remotely created documents from create broadcast frames", async () => {
    const remote = createTestClient();
    try {
      await remote.connect();
      await client.connect();
      const plot = await seedPlot();
      await expect
        .poll(() => query.isLive(remote.lineplots.watch(plot.key).getCached()))
        .toBe(true);
      const cached = remote.lineplots.watch(plot.key).getCached();
      if (query.isLive(cached)) expect(cached.name).toEqual(plot.name);
    } finally {
      await remote.close();
    }
  });

  it("drops a deleted channel from the cached document", async () => {
    await client.connect();
    const ch = await client.channels.create({
      name: `lineplot_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const proj = await client.projects.create({ name: `lp-${id.create()}` });
    const plot = await client.lineplots.create({
      project: proj.key,
      name: `plot-${id.create()}`,
      channels: { y1: [ch.key] },
    });
    expect(plot.lines).toHaveLength(1);
    await client.channels.delete(ch.key);
    await expect
      .poll(() => expectLive(client.lineplots.watch(plot.key).getCached()).channels.y1)
      .toEqual([]);
    expect(expectLive(client.lineplots.watch(plot.key).getCached()).lines).toEqual([]);
  });

  it("reduces broadcast dispatch frames into the cached document", async () => {
    await client.connect();
    const plot = await seedPlot();
    const name = `renamed-${id.create()}`;
    await client.lineplots.dispatch(plot.key, [
      lineplot.set({ path: ["name"], value: name }),
    ]);
    await expect.poll(() => cachedName(plot.key)).toBe(name);
  });
});
