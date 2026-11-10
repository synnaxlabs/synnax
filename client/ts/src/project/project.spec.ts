// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { NotFoundError } from "@/index";
import { log } from "@/log";
import { project } from "@/project";
import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("Project", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
      });
      expect(proj.name).toEqual("Schematic");
      expect(proj.key).not.toEqual(uuid.ZERO);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
      });
      await client.projects.rename(proj.key, "Schematic2");
      const res = await client.projects.retrieve(proj.key);
      expect(res.name).toEqual("Schematic2");
    });
  });
  describe("dispatch", () => {
    test("should apply the action to the project", async () => {
      const proj = await client.projects.create({ name: "Schematic" });
      const name = `project-${id.create()}`;
      await client.projects.dispatch(
        proj.key,
        project.set({ path: ["name"], value: name }),
      );
      expect((await createTestClient().projects.retrieve(proj.key)).name).toEqual(name);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
      });
      await client.projects.delete(proj.key);
      await expect(client.projects.retrieve(proj.key)).rejects.toThrow();
    });
  });
  describe("retrieve", () => {
    test("retrieve projects by search term", async () => {
      const prefix = `searchable-project-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.projects.create(names.map((name) => ({ name })));
      await expect
        .poll(async () => {
          const results = await client.projects.retrieve({ searchTerm: prefix });
          return results.map((p) => p.name).sort();
        })
        .toEqual(names);
    });
    test("throw a NotFoundError for a missing key in a key list", async () => {
      const proj = await client.projects.create({ name: "survivor" });
      const missing = uuid.create();
      await expect(client.projects.retrieve([proj.key, missing])).rejects.toThrow(
        new NotFoundError(`project with key ${missing} not found`),
      );
    });
    test("omit missing keys from a filter on keys", async () => {
      const proj = await client.projects.create({ name: "survivor" });
      const res = await client.projects.retrieve({ key: [proj.key, uuid.create()] });
      expect(res.map(({ key }) => key)).toEqual([proj.key]);
    });
  });
  describe("export", () => {
    // Zip entry names are stored uncompressed, so the raw archive names the bundle's
    // files without the spec needing a zip reader.
    const download = async (key: string): Promise<string> => {
      const stream = await client.projects.export(key, { encoding: "JSON" });
      return new TextDecoder().decode(await new Response(stream).arrayBuffer());
    };

    test("export documents and panels as a zip bundle", async () => {
      const proj = await client.projects.create({ name: `export-${id.create()}` });
      const l = await client.logs.create({ project: proj.key, name: "Metrics Log" });
      await client.panels.create({
        key: uuid.create(),
        name: "Controls",
        project: proj.key,
        root: {
          variant: "leaf",
          tabs: [
            {
              key: uuid.create(),
              variant: "resource",
              resource: log.resourceID(l.key),
            },
          ],
        },
      });
      const archive = await download(proj.key);
      expect(archive.startsWith("PK")).toBe(true);
      expect(archive).toContain("manifest.json");
      expect(archive).toContain("Metrics Log.json");
      expect(archive).toContain("Controls.json");
    });

    test("grouped members export into directories", async () => {
      const proj = await client.projects.create({ name: `export-${id.create()}` });
      const g = await client.groups.create({
        parent: project.resourceID(proj.key),
        name: "Propulsion",
      });
      const l = await client.logs.create({ project: proj.key, name: "Pressure" });
      await client.batch.send(client.logs.setGroupRequest(l.key, g.key));
      const archive = await download(proj.key);
      expect(archive).toContain("Propulsion/Pressure.json");
    });

    test("suffixes the second of two members taking one file name", async () => {
      const proj = await client.projects.create({ name: `export-${id.create()}` });
      await client.logs.create({ project: proj.key, name: "Pressure" });
      await client.logs.create({ project: proj.key, name: "pressure" });
      const archive = await download(proj.key);
      expect(archive).toContain("Pressure.json");
      expect(archive).toContain("pressure (1).json");
    });
  });
  describe("import", () => {
    test("imports an exported bundle back as a fresh project", async () => {
      const src = await client.projects.create({ name: `import-${id.create()}` });
      const l = await client.logs.create({ project: src.key, name: "Metrics" });
      await client.panels.create({
        key: uuid.create(),
        name: "Controls",
        project: src.key,
        root: {
          variant: "leaf",
          tabs: [
            {
              key: uuid.create(),
              variant: "resource",
              resource: log.resourceID(l.key),
            },
          ],
        },
      });
      const stream = await client.projects.export(src.key, { encoding: "JSON" });
      const bundle = new Uint8Array(await new Response(stream).arrayBuffer());
      const imported = await client.projects.import(bundle, {
        fileName: `${src.name}.zip`,
      });
      expect(imported.key).not.toEqual(src.key);
      expect(imported.name).toEqual(src.name);
      const logs = await client.logs.retrieve({ projects: [imported.key] });
      const panels = await client.panels.retrieve({ projects: [imported.key] });
      expect([...logs, ...panels].map(({ name }) => name).sort()).toEqual([
        "Controls",
        "Metrics",
      ]);
    });

    test("recreates group directories as groups", async () => {
      const src = await client.projects.create({ name: `import-${id.create()}` });
      const g = await client.groups.create({
        parent: project.resourceID(src.key),
        name: "Propulsion",
      });
      const l = await client.logs.create({ project: src.key, name: "Pressure" });
      await client.batch.send(client.logs.setGroupRequest(l.key, g.key));
      const stream = await client.projects.export(src.key, { encoding: "JSON" });
      const bundle = new Uint8Array(await new Response(stream).arrayBuffer());
      const imported = await client.projects.import(bundle, {
        fileName: `${src.name}.zip`,
      });
      const groups = await client.groups.retrieve({
        parents: [project.resourceID(imported.key)],
      });
      expect(groups.map(({ name }) => name)).toEqual(["Propulsion"]);
      const logs = await client.logs.retrieve({ groups: [groups[0].key] });
      expect(logs.map(({ name }) => name)).toEqual(["Pressure"]);
    });

    test("rejects a bundle without a manifest", async () => {
      // The 22-byte end-of-central-directory record alone: a valid, empty archive.
      const emptyArchive = new Uint8Array(22);
      emptyArchive.set([0x50, 0x4b, 0x05, 0x06]);
      await expect(
        client.projects.import(emptyArchive, { fileName: "Empty.zip" }),
      ).rejects.toThrow("bundle holds no manifest.json");
    });
  });
});
