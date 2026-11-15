// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

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

  describe("name", () => {
    const rejection = 'Name "bad name" must start with a letter or underscore';

    it("should reject a create whose name is not an identifier", async () => {
      const root = await client.arcs.directories.retrieveRoot();
      const key = uuid.create();
      const created = client.arcs.directories.create({
        parent: root.key,
        key,
        name: "bad name",
      });
      await expect(created).rejects.toSatisfy(ValidationError.matches);
      await expect(created).rejects.toThrow(rejection);
      expect(query.isLive(client.arcs.directories.watch(key).getCached())).toBe(false);
    });

    it("should reject a batched create whose name is not an identifier", async () => {
      const root = await client.arcs.directories.retrieveRoot();
      expect(() =>
        client.arcs.directories.createRequest({ parent: root.key, name: "bad name" }),
      ).toThrow(rejection);
    });

    it("should reject a rename to a name that is not an identifier", async () => {
      const root = await client.arcs.directories.retrieveRoot();
      const d = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      const renamed = client.arcs.directories.rename(d.key, "bad name");
      await expect(renamed).rejects.toSatisfy(ValidationError.matches);
      await expect(renamed).rejects.toThrow(rejection);
      const [res] = await client.arcs.directories.retrieve([d.key]);
      expect(res.name).toBe(d.name);
    });
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

  it("should apply a dispatched action to a directory", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const name = uniqueName();
    await client.arcs.directories.dispatch(
      d.key,
      arc.directory.set({ path: ["name"], value: name }),
    );
    const [res] = await createTestClient().arcs.directories.retrieve([d.key]);
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

  describe("batch", () => {
    it("should create a directory and move an Arc into it", async () => {
      const root = await client.arcs.directories.retrieveRoot();
      const a = await createArc();
      const key = uuid.create();
      const { directories } = client.arcs;
      await client.batch.send([
        directories.createRequest({ parent: root.key, key, name: uniqueName() }),
        ...directories.moveRequests(
          arc.directory.resourceID(key),
          arc.resourceID(a.key),
        ),
      ]);
      expect((await client.arcs.retrieve({ key: a.key })).directory).toBe(key);
    });

    it("should create nothing when the move fails", async () => {
      const root = await client.arcs.directories.retrieveRoot();
      const other = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      const a = await createArc();
      const clash = await client.arcs.directories.create({
        parent: other.key,
        name: a.name,
      });
      const key = uuid.create();
      const { directories } = client.arcs;
      await expect(
        client.batch.send([
          directories.createRequest({ parent: root.key, key, name: uniqueName() }),
          ...directories.moveRequests(
            arc.directory.resourceID(key),
            arc.resourceID(a.key),
            arc.directory.resourceID(clash.key),
          ),
        ]),
      ).rejects.toSatisfy(ValidationError.matches);
      const created = await client.arcs.directories.retrieve({ parents: [root.key] });
      expect(created.map((d) => d.key)).not.toContain(key);
    });
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

  const createChain = async (): Promise<arc.directory.Directory[]> => {
    const root = await client.arcs.directories.retrieveRoot();
    const mid = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const leaf = await client.arcs.directories.create({
      parent: mid.key,
      name: uniqueName(),
    });
    return [root, mid, leaf];
  };

  describe("retrieveAncestors", () => {
    it("should retrieve the directories above a directory up to the root", async () => {
      const [root, mid, leaf] = await createChain();
      expect(await client.arcs.directories.retrieveAncestors(leaf.key)).toEqual([
        mid,
        root,
      ]);
    });

    it("should return each directory above the directories once", async () => {
      const [root, mid, leaf] = await createChain();
      expect(
        await client.arcs.directories.retrieveAncestors([leaf.key, mid.key]),
      ).toEqual([root]);
    });

    it("should throw when a directory does not exist", async () => {
      await expect(
        client.arcs.directories.retrieveAncestors(uuid.create()),
      ).rejects.toSatisfy(NotFoundError.matches);
    });
  });

  describe("getCachedAncestors", () => {
    it("should give the directories that a retrieve gives", async () => {
      const [, mid, leaf] = await createChain();
      const retrieved = await client.arcs.directories.retrieveAncestors([
        leaf.key,
        mid.key,
      ]);
      expect(client.arcs.directories.getCachedAncestors([leaf.key, mid.key])).toEqual(
        retrieved,
      );
    });

    it("should return undefined when a directory above is not cached", async () => {
      const [, , leaf] = await createChain();
      const other = createTestClient();
      await other.arcs.directories.retrieve([leaf.key]);
      expect(other.arcs.directories.getCachedAncestors(leaf.key)).toBeUndefined();
    });
  });

  describe("onAncestorsChange", () => {
    it("should call the handler when a directory above changes", async () => {
      const [root, mid, leaf] = await createChain();
      await client.arcs.directories.retrieveAncestors(leaf.key);
      const handler = vi.fn();
      const stop = client.arcs.directories.onAncestorsChange(leaf.key, handler);
      const name = uniqueName();
      await client.arcs.directories.rename(mid.key, name);
      await expect
        .poll(() => handler.mock.lastCall?.[0])
        .toEqual([
          expect.objectContaining({ key: mid.key, name }),
          expect.objectContaining({ key: root.key }),
        ]);
      const dest = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      await client.arcs.directories.move(
        arc.directory.resourceID(dest.key),
        arc.directory.resourceID(mid.key),
      );
      await expect
        .poll(() => handler.mock.lastCall?.[0])
        .toEqual([
          expect.objectContaining({ key: mid.key }),
          expect.objectContaining({ key: dest.key }),
          expect.objectContaining({ key: root.key }),
        ]);
      stop();
    });
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
