// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test, vi } from "vitest";

import { NotFoundError } from "@/index";
import { lineplot } from "@/lineplot";
import { query } from "@/query";
import { createTestClient, expectDeleted } from "@/testutil";

const client = createTestClient();

describe("LinePlot", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create(proj.key, { name: "Line Plot" });
      expect(linePlot.name).toEqual("Line Plot");
      expect(linePlot.key).not.toEqual(uuid.ZERO);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({
        name: "Line Plot",
      });
      const linePlot = await client.lineplots.create(proj.key, { name: "Line Plot" });
      await client.lineplots.rename(linePlot.key, "Line Plot2");
      const res = await client.lineplots.retrieve(linePlot.key);
      expect(res.name).toEqual("Line Plot2");
    });
  });
  describe("project", () => {
    test("retrieve the lineplots of a project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const other = await client.projects.create({ name: "Line Plot" });
      const v = await client.lineplots.create(proj.key, { name: "Line Plot" });
      expect(v.project).toEqual(proj.key);
      const owned = await client.lineplots.retrieve({ projects: [proj.key] });
      expect(owned.map(({ key }) => key)).toEqual([v.key]);
      const unowned = await client.lineplots.retrieve({ projects: [other.key] });
      expect(unowned).toEqual([]);
    });

    test("move to another project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const dest = await client.projects.create({ name: "Line Plot" });
      const v = await client.lineplots.create(proj.key, { name: "Line Plot" });
      await client.lineplots.setProject(v.key, dest.key);
      const res = await client.lineplots.retrieve({ keys: [v.key] });
      expect(res[0].project).toEqual(dest.key);
    });

    test("delete with its project", async () => {
      const proj = await client.projects.create({ name: "Line Plot" });
      const v = await client.lineplots.create(proj.key, { name: "Line Plot" });
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
      const linePlot = await client.lineplots.create(proj.key, { name: "Line Plot" });
      await client.lineplots.delete(linePlot.key);
      await expect(client.lineplots.retrieve(linePlot.key)).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});

const seedPlot = async (): Promise<lineplot.LinePlot> => {
  const project = await client.projects.create({ name: `lp-${id.create()}` });
  return await client.lineplots.create(project.key, { name: `plot-${id.create()}` });
};

const cachedName = (key: lineplot.LinePlot["key"]): string | undefined => {
  const cached = client.lineplots.getCached(key);
  return query.isLive(cached) ? cached.name : undefined;
};

describe("store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.connect();
    const plot = await seedPlot();
    await client.lineplots.delete(plot.key);
    await expect
      .poll(() => query.Deleted.matches(client.lineplots.getCached(plot.key)))
      .toBe(true);
    const cached = expectDeleted(client.lineplots.getCached(plot.key));
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
        .poll(() => query.isLive(remote.lineplots.getCached(plot.key)))
        .toBe(true);
      const cached = remote.lineplots.getCached(plot.key);
      if (query.isLive(cached)) expect(cached.name).toEqual(plot.name);
    } finally {
      await remote.close();
    }
  });

  it("reduces broadcast dispatch frames into the cached document", async () => {
    await client.connect();
    const plot = await seedPlot();
    const name = `renamed-${id.create()}`;
    await client.lineplots.dispatch(plot.key, [lineplot.rename({ name })]);
    await expect.poll(() => cachedName(plot.key)).toBe(name);
  });
});
