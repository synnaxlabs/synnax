// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { panel } from "@/panel";
import { type project } from "@/project";
import {
  createEmptyRole,
  createPanelProject,
  createTestClient,
  expectLive,
} from "@/testutil";
import { type user } from "@/user";

const client = createTestClient();

describe("Panel Client", () => {
  // Every panel here shares this project.
  let proj: project.Key;

  beforeAll(async () => {
    proj = await createPanelProject(client);
  });

  const createPanel = async (): Promise<panel.Panel> =>
    await client.panels.create({ name: `panel-${id.create()}`, project: proj });

  describe("retrieve", () => {
    it("should combine a project filter with a where", async () => {
      const scope = await createPanelProject(client);
      const [kept] = await client.panels.create([
        { name: `panel-${id.create()}`, project: scope },
        { name: `panel-${id.create()}`, project: scope },
      ]);
      const outsider = await createPanel();
      const retrieved = await client.panels.retrieve({
        where: { projects: [scope], key: { in: [kept.key, outsider.key] } },
      });
      expect(retrieved.map((p) => p.key)).toEqual([kept.key]);
    });

    it("should combine a project filter with a flat filter", async () => {
      const scope = await createPanelProject(client);
      const [kept] = await client.panels.create([
        { name: `panel-${id.create()}`, project: scope },
        { name: `panel-${id.create()}`, project: scope },
      ]);
      const outsider = await createPanel();
      const retrieved = await client.panels.retrieve({
        projects: [scope],
        key: [kept.key, outsider.key],
      });
      expect(retrieved.map((p) => p.key)).toEqual([kept.key]);
    });
  });

  describe("drafts", () => {
    const createOwner = async (): Promise<user.Key> => {
      const role = await createEmptyRole(client);
      const u = await client.users.create({
        username: id.create(),
        password: "test",
        name: "owner",
        role: role.key,
      });
      return u.key;
    };

    it("should match the panels visible to one user", async () => {
      const scope = await createPanelProject(client);
      const [mine, theirs] = await Promise.all([createOwner(), createOwner()]);
      const where = panel.visibleTo(scope, mine);
      const list = client.panels.watch({ where });
      const off = list.onChange(vi.fn());
      expect((await list.retrieve()).records).toEqual([]);
      const created = await client.panels.create([
        { name: `panel-${id.create()}`, project: scope },
        { name: `panel-${id.create()}`, project: scope, owner: mine },
        { name: `panel-${id.create()}`, project: scope, owner: theirs },
      ]);
      const expected = [created[0].key, created[1].key].sort();
      const cached = expectLive(list.getCached()).records.map((p) => p.key);
      expect(cached.sort()).toEqual(expected);
      const retrieved = await client.panels.retrieve({ where });
      expect(retrieved.map((p) => p.key).sort()).toEqual(expected);
      off();
    });
  });

  describe("rename", () => {
    it("should cache the new name before the send resolves", async () => {
      const pan = await createPanel();
      const renamed = client.panels.rename(pan.key, "renamed");
      expect(expectLive(client.panels.watch(pan.key).getCached()).name).toEqual(
        "renamed",
      );
      await renamed;
    });

    it("should restore the previous name when the Core rejects it", async () => {
      const pan = await createPanel();
      await expect(client.panels.rename(pan.key, "")).rejects.toThrow("name: required");
      expect(expectLive(client.panels.watch(pan.key).getCached()).name).toEqual(
        pan.name,
      );
    });
  });
});
