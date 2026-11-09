// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { arc } from "@/arc";
import { NotFoundError, type Synnax, ValidationError } from "@/index";
import { query } from "@/query";
import { createTestClient, waitForStreamLive } from "@/testutil";

const client = createTestClient();

const uniqueName = (): string => `d_${id.create().replace(/-/g, "_")}`;

const createArc = async (): Promise<arc.Arc> =>
  await client.arcs.create({
    name: uniqueName(),
    type: "automation",
    source: { mode: "text" },
  });

describe("arc.directory", () => {
  it("should retrieve the root directory", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    expect(root.name).toBe("Arcs");
    expect(root.parent).toBeUndefined();
  });

  it("should create a directory in its parent", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    expect(d.parent).toBe(root.key);
    const children = await client.arcs.directories.retrieve({ parents: [root.key] });
    expect(children.map((c) => c.key)).toContain(d.key);
  });

  it("should cache a directory with a key before the Core answers", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const key = uuid.create();
    const name = uniqueName();
    const created = client.arcs.directories.create({ parent: root.key, key, name });
    expect(client.arcs.directories.watch(key).getCached()).toEqual({
      key,
      name,
      parent: root.key,
    });
    await created;
  });

  it("should drop the cached directory when the Core rejects the create", async () => {
    const key = uuid.create();
    const parent = uuid.create();
    const created = client.arcs.directories.create({ parent, key, name: uniqueName() });
    await expect(created).rejects.toThrow(`parent: arc_directory ${parent}`);
    expect(query.isLive(client.arcs.directories.watch(key).getCached())).toBe(false);
  });

  it("should rename a directory", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const name = uniqueName();
    await client.arcs.directories.rename(d.key, name);
    const [res] = await client.arcs.directories.retrieve([d.key]);
    expect(res.name).toBe(name);
  });

  it("should retrieve a directory by name", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const res = await client.arcs.directories.retrieve(d.name);
    expect(res.key).toBe(d.key);
  });

  it("should delete a directory", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    await client.arcs.directories.delete(d.key);
    await expect(client.arcs.directories.retrieve([d.key])).rejects.toSatisfy(
      NotFoundError.matches,
    );
  });

  it("should put a new Arc in the root directory", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const a = await createArc();
    const arcs = await client.arcs.retrieve({ directories: [root.key] });
    expect(arcs.map((r) => r.key)).toContain(a.key);
  });

  it("should move an Arc into a directory", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const a = await createArc();
    await client.arcs.directories.move(
      arc.directory.resourceID(d.key),
      arc.resourceID(a.key),
    );
    const res = await client.arcs.retrieve({ key: a.key });
    expect(res.directory).toBe(d.key);
  });

  it("should move an Arc and a directory in one call", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const [dest, moved] = await Promise.all(
      [uniqueName(), uniqueName()].map(
        async (name) =>
          await client.arcs.directories.create({ parent: root.key, name }),
      ),
    );
    const a = await createArc();
    await client.arcs.directories.move(
      arc.directory.resourceID(dest.key),
      arc.resourceID(a.key),
      arc.directory.resourceID(moved.key),
    );
    expect((await client.arcs.retrieve({ key: a.key })).directory).toBe(dest.key);
    const [res] = await createTestClient().arcs.directories.retrieve([moved.key]);
    expect(res.parent).toBe(dest.key);
  });

  it("should move nothing when one entry cannot move", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const dest = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const name = uniqueName();
    await client.arcs.directories.create({ parent: dest.key, name });
    const clash = await client.arcs.directories.create({ parent: root.key, name });
    const a = await createArc();
    await expect(
      client.arcs.directories.move(
        arc.directory.resourceID(dest.key),
        arc.resourceID(a.key),
        arc.directory.resourceID(clash.key),
      ),
    ).rejects.toSatisfy(ValidationError.matches);
    const res = await createTestClient().arcs.retrieve({ key: a.key });
    expect(res.directory).toBe(root.key);
  });

  describe("cache of another client", () => {
    const watchDirectory = async (): Promise<{
      watcher: Synnax;
      root: arc.directory.Directory;
      dir: arc.directory.Directory;
    }> => {
      const root = await client.arcs.directories.retrieveRoot();
      const dir = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      const watcher = createTestClient();
      await watcher.connect();
      await waitForStreamLive(watcher.connection);
      await watcher.arcs.directories.retrieve([dir.key]);
      return { watcher, root, dir };
    };
    const cachedOf = (
      watcher: Synnax,
      key: arc.directory.Key,
    ): arc.directory.Directory | undefined => {
      const cached = watcher.arcs.directories.watch(key).getCached();
      return query.isLive(cached) ? cached : undefined;
    };

    it("should take a rename", async () => {
      const { watcher, dir } = await watchDirectory();
      const name = uniqueName();
      await client.arcs.directories.rename(dir.key, name);
      await expect.poll(() => cachedOf(watcher, dir.key)?.name).toBe(name);
    });

    it("should take a move", async () => {
      const { watcher, root, dir } = await watchDirectory();
      const dest = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      await client.arcs.directories.move(
        arc.directory.resourceID(dest.key),
        arc.directory.resourceID(dir.key),
      );
      await expect.poll(() => cachedOf(watcher, dir.key)?.parent).toBe(dest.key);
    });

    it("should take a delete", async () => {
      const { watcher, dir } = await watchDirectory();
      await client.arcs.directories.delete(dir.key);
      await expect
        .poll(() =>
          query.Deleted.matches(watcher.arcs.directories.watch(dir.key).getCached()),
        )
        .toBe(true);
    });
  });
});
