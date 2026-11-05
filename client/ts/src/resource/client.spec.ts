// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NotFoundError, ValidationError } from "@/errors";
import { group } from "@/group";
import { label } from "@/label";
import { type resource } from "@/resource";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const randomToken = (): string => `res${Math.random().toString(36).slice(2)}`;

const createGroup = async (name: string): Promise<resource.ID> =>
  group.resourceID((await client.groups.create({ scope: "channel", name })).key);

const createLabel = async (name: string): Promise<resource.ID> =>
  label.resourceID((await client.labels.create({ name, color: "#E87E04" })).key);

describe("Resource", () => {
  describe("search", () => {
    it("should return the matches of every type, closest match first", async () => {
      const token = randomToken();
      const exact = await createGroup(token);
      const partial = await createLabel(`${token} pump`);
      await expect
        .poll(async () => await client.resources.search({ term: token }))
        .toEqual([
          { key: `group:${exact.key}`, id: exact, name: token },
          { key: `label:${partial.key}`, id: partial, name: `${token} pump` },
        ]);
    });

    it("should search only the given types", async () => {
      const token = randomToken();
      await createGroup(token);
      const l = await createLabel(token);
      await expect
        .poll(async () =>
          (await client.resources.search({ term: token, types: ["label"] })).map(
            (r) => r.id,
          ),
        )
        .toEqual([l]);
    });

    it("should page through the matches with a limit and an offset", async () => {
      const token = randomToken();
      for (const suffix of ["a", "b", "c"]) await createGroup(`${token} ${suffix}`);
      const page = async (offset: number): Promise<number> =>
        (await client.resources.search({ term: token, limit: 2, offset })).length;
      await expect.poll(async () => await page(0)).toEqual(2);
      expect(await page(2)).toEqual(1);
    });

    it("should list the resources of the given types for an empty term", async () => {
      await createLabel(randomToken());
      await createLabel(randomToken());
      const page = await client.resources.search({
        term: "",
        types: ["label"],
        limit: 2,
      });
      expect(page).toHaveLength(2);
      expect(page.every((r) => r.id.type === "label")).toBe(true);
      expect(
        await client.resources.search({
          term: "",
          types: ["label"],
          limit: 1,
          offset: 1,
        }),
      ).toEqual([page[1]]);
    });

    it("should refuse a type that the Core cannot search", async () => {
      await expect(
        client.resources.search({ term: "pump", types: ["builtin"] }),
      ).rejects.toSatisfy(
        (e) =>
          ValidationError.matches(e) &&
          (e as Error).message === 'types: unknown resource type "builtin"',
      );
    });
  });

  describe("retrieve", () => {
    it("should retrieve the name of a single resource", async () => {
      const name = randomToken();
      const id = await createLabel(name);
      expect(await client.resources.retrieve(id)).toEqual({
        key: `label:${id.key}`,
        id,
        name,
      });
    });

    it("should retrieve resources of mixed types in the order of the IDs", async () => {
      const groupName = randomToken();
      const labelName = randomToken();
      const g = await createGroup(groupName);
      const l = await createLabel(labelName);
      expect(
        (await client.resources.retrieve([l, g])).map(({ id, name }) => ({ id, name })),
      ).toEqual([
        { id: l, name: labelName },
        { id: g, name: groupName },
      ]);
    });

    it("should leave out an ID that names no resource", async () => {
      const name = randomToken();
      const l = await createLabel(name);
      const missing = label.resourceID("00000000-0000-0000-0000-000000000000");
      expect(
        (await client.resources.retrieve([missing, l])).map((r) => r.name),
      ).toEqual([name]);
    });

    it("should throw a NotFoundError when a single ID names no resource", async () => {
      await expect(
        client.resources.retrieve(
          label.resourceID("00000000-0000-0000-0000-000000000000"),
        ),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
