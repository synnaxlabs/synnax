// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, onTestFinished } from "vitest";

import { NotFoundError, ValidationError } from "@/errors";
import { group } from "@/group";
import { label } from "@/label";
import { lineplot } from "@/lineplot";
import { query } from "@/query";
import { resource } from "@/resource";
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
        .poll(async () => await client.resources.search({ searchTerm: token }))
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
          (await client.resources.search({ searchTerm: token, types: ["label"] })).map(
            (r) => r.id,
          ),
        )
        .toEqual([l]);
    });

    it("should page through the matches with a limit and an offset", async () => {
      const token = randomToken();
      for (const suffix of ["a", "b", "c"]) await createGroup(`${token} ${suffix}`);
      const page = async (offset: number): Promise<number> =>
        (await client.resources.search({ searchTerm: token, limit: 2, offset })).length;
      await expect.poll(async () => await page(0)).toEqual(2);
      expect(await page(2)).toEqual(1);
    });

    it("should list the resources of the given types for an empty term", async () => {
      await createLabel(randomToken());
      await createLabel(randomToken());
      const page = await client.resources.search({
        searchTerm: "",
        types: ["label"],
        limit: 2,
      });
      expect(page).toHaveLength(2);
      expect(page.every((r) => r.id.type === "label")).toBe(true);
      expect(
        await client.resources.search({
          searchTerm: "",
          types: ["label"],
          limit: 1,
          offset: 1,
        }),
      ).toEqual([page[1]]);
    });

    it("should refuse a type that the Core cannot search", async () => {
      await expect(
        client.resources.search({ searchTerm: "pump", types: ["builtin"] }),
      ).rejects.toSatisfy(
        (e) =>
          ValidationError.matches(e) &&
          (e as Error).message === 'types: unknown resource type "builtin"',
      );
    });
  });

  describe("onChange", () => {
    // Subscribes before the first read, so the answer is maintained from its first
    // fetch and only change channels can move it afterward.
    const subscribe = async (
      params: resource.SearchParams,
    ): Promise<() => resource.Resource[] | undefined> => {
      let latest: resource.Resource[] | undefined;
      const off = client.resources.onChange(params, (answer) => {
        if (query.isLive(answer)) latest = answer;
      });
      onTestFinished(off);
      latest = await client.resources.search(params);
      return () => latest;
    };

    it("should add a created resource to a subscribed list", async () => {
      const read = await subscribe({ searchTerm: "", types: ["group"] });
      const g = await createGroup(randomToken());
      await expect.poll(() => read()?.map((r) => r.id)).toContainEqual(g);
    });

    // Waits for the search index to hold the term. A different limit keeps this read
    // off the answer that a spec subscribes to.
    const awaitIndexed = async (term: string, count: number): Promise<void> => {
      await expect
        .poll(
          async () =>
            (await client.resources.search({ searchTerm: term, limit: 10 })).length,
        )
        .toEqual(count);
    };

    it("should rename a resource in a subscribed search", async () => {
      const token = randomToken();
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({ project: proj.key, name: token });
      await awaitIndexed(token, 1);
      const read = await subscribe({ searchTerm: token, types: ["lineplot"] });
      await client.lineplots.rename(plot.key, `${token} renamed`);
      const key = resource.idToString(lineplot.resourceID(plot.key));
      await expect
        .poll(() => read()?.find((r) => r.key === key)?.name)
        .toEqual(`${token} renamed`);
    });

    // Two seconds stays inside the five second grace window after an unsubscribe.
    const GRACE_POLL = { timeout: 2_000 };

    const subscribeThenLeave = async (params: resource.SearchParams): Promise<void> => {
      const off = client.resources.onChange(params, () => {});
      await client.resources.search(params);
      off();
    };

    it("should show a rename made after the last unsubscribe", async () => {
      const token = randomToken();
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({ project: proj.key, name: token });
      await awaitIndexed(token, 1);
      const params: resource.SearchParams = { searchTerm: token, types: ["lineplot"] };
      await subscribeThenLeave(params);
      await client.lineplots.rename(plot.key, `${token} renamed`);
      await expect
        .poll(async () => (await client.resources.search(params))[0]?.name, GRACE_POLL)
        .toEqual(`${token} renamed`);
    });

    it("should show a resource created after the last unsubscribe", async () => {
      const token = randomToken();
      const params: resource.SearchParams = { searchTerm: token, types: ["label"] };
      await subscribeThenLeave(params);
      const l = await createLabel(token);
      await expect
        .poll(
          async () => (await client.resources.search(params)).map((r) => r.id),
          GRACE_POLL,
        )
        .toEqual([l]);
    });

    it("should drop a deleted resource from a subscribed search", async () => {
      const token = randomToken();
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({ project: proj.key, name: token });
      await awaitIndexed(token, 1);
      const read = await subscribe({ searchTerm: token, types: ["lineplot"] });
      await client.lineplots.delete(plot.key);
      await expect.poll(() => read()).toEqual([]);
    });
  });

  describe("getCached", () => {
    it("should return the answer of a search read before", async () => {
      const token = randomToken();
      const l = await createLabel(token);
      const params: resource.SearchParams = { searchTerm: token, types: ["label"] };
      await expect
        .poll(async () => (await client.resources.search(params)).map((r) => r.id))
        .toEqual([l]);
      const cached = client.resources.getCached(params);
      expect(query.isLive(cached) && cached.map((r) => r.id)).toEqual([l]);
    });

    it("should return undefined for a search never read", () => {
      expect(client.resources.getCached({ searchTerm: randomToken() })).toBeUndefined();
    });
  });

  describe("onChange by ID", () => {
    it("should call the handler with the new name of a renamed resource", async () => {
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({
        project: proj.key,
        name: randomToken(),
      });
      const id = lineplot.resourceID(plot.key);
      let latest: resource.Resource | undefined;
      const off = client.resources.onChange(id, (answer) => {
        if (query.isLive(answer)) latest = answer;
      });
      onTestFinished(off);
      await client.resources.retrieve(id);
      await client.lineplots.rename(plot.key, "renamed");
      await expect.poll(() => latest?.name).toEqual("renamed");
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
      const read = client.resources.retrieve(
        label.resourceID("00000000-0000-0000-0000-000000000000"),
      );
      await expect(read).rejects.toThrow(NotFoundError);
      await expect(read).rejects.toThrow(/Resource not found matching/);
    });
  });
});
