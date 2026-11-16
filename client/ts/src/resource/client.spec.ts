// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { channel } from "@/channel";
import { NotFoundError, ValidationError } from "@/errors";
import { group } from "@/group";
import { label } from "@/label";
import { lineplot } from "@/lineplot";
import { query } from "@/query";
import { resource } from "@/resource";
import { status } from "@/status";
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
    });

    it("should refuse a negative limit", async () => {
      await expect(
        client.resources.search({ searchTerm: "pump", limit: -1 }),
      ).rejects.toSatisfy(ValidationError.matches);
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

  describe("watch a search", () => {
    // Subscribes before the first read, so the window is maintained from its first
    // fetch and only change channels can move it afterward.
    const subscribe = async (
      params: resource.SearchParams,
    ): Promise<() => resource.Resource[] | undefined> => {
      const list = client.resources.watch(params);
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      return () => list.getCached()?.records;
    };

    // The Core holds groups from other specs, so the list takes the most it can.
    it("should add a created resource to a subscribed list", async () => {
      const read = await subscribe({ searchTerm: "", types: ["group"], limit: 1000 });
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

    it("should leave a retrieved internal channel out of a subscribed list", async () => {
      // The term names the channel, so it would rank first if the list took it.
      const read = await subscribe({
        searchTerm: channel.SET_CHANNEL_NAME,
        types: ["channel"],
        limit: 50,
      });
      const internal = await client.channels.retrieve(channel.SET_CHANNEL_NAME);
      const id = channel.resourceID(internal.key);
      await client.resources.retrieve(id);
      expect(read()?.map((r) => r.id)).not.toContainEqual(id);
    });

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

    it("should add a held resource to a subscribed search that its rename makes it match", async () => {
      const token = randomToken();
      const other = randomToken();
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({ project: proj.key, name: token });
      await awaitIndexed(token, 1);
      const id = lineplot.resourceID(plot.key);
      await client.resources.retrieve(id);
      const read = await subscribe({ searchTerm: other, types: ["lineplot"] });
      expect(read()).toEqual([]);
      await client.lineplots.rename(plot.key, `${other} plot`);
      await expect.poll(() => read()?.map((r) => r.id)).toContainEqual(id);
    });

    // Two seconds stays inside the five second grace window after an unsubscribe.
    const GRACE_POLL = { timeout: 2_000 };

    const subscribeThenLeave = async (
      params: resource.SearchParams,
    ): Promise<query.List<resource.Resource, string>> => {
      const list = client.resources.watch(params);
      const off = list.onChange(() => {});
      await list.retrieve();
      off();
      return list;
    };

    it("should show a rename made after the last unsubscribe", async () => {
      const token = randomToken();
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({ project: proj.key, name: token });
      await awaitIndexed(token, 1);
      const params: resource.SearchParams = { searchTerm: token, types: ["lineplot"] };
      const list = await subscribeThenLeave(params);
      await client.lineplots.rename(plot.key, `${token} renamed`);
      await expect
        .poll(async () => (await list.retrieve()).records[0]?.name, GRACE_POLL)
        .toEqual(`${token} renamed`);
    });

    it("should show a resource created after the last unsubscribe", async () => {
      const token = randomToken();
      const params: resource.SearchParams = { searchTerm: token, types: ["label"] };
      const list = await subscribeThenLeave(params);
      const l = await createLabel(token);
      await expect
        .poll(async () => (await list.retrieve()).records.map((r) => r.id), GRACE_POLL)
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
    it("should return the window of a search read before", async () => {
      const token = randomToken();
      const l = await createLabel(token);
      const list = client.resources.watch({ searchTerm: token, types: ["label"] });
      await expect
        .poll(async () => (await list.retrieve()).records.map((r) => r.id))
        .toEqual([l]);
      expect(list.getCached()?.records.map((r) => r.id)).toEqual([l]);
    });

    it("should return undefined for a search never read", () => {
      expect(
        client.resources.watch({ searchTerm: randomToken() }).getCached(),
      ).toBeUndefined();
    });
  });

  describe("watch an ID", () => {
    it("should call the handler with the new name of a renamed resource", async () => {
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({
        project: proj.key,
        name: randomToken(),
      });
      const id = lineplot.resourceID(plot.key);
      let latest: resource.Resource | undefined;
      const off = client.resources.watch(id).onChange((answer) => {
        if (query.isLive(answer)) latest = answer;
      });
      onTestFinished(off);
      await client.resources.retrieve(id);
      await client.lineplots.rename(plot.key, "renamed");
      await expect.poll(() => latest?.name).toEqual("renamed");
    });
  });

  describe("a list of IDs", () => {
    it("should call the handler with the new name of a renamed resource", async () => {
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({
        project: proj.key,
        name: randomToken(),
      });
      const ids = [lineplot.resourceID(plot.key), group.resourceID(uuid.create())];
      let latest: resource.Resource[] | undefined;
      const off = client.resources.watch(ids).onChange((answer) => {
        if (query.isLive(answer)) latest = answer;
      });
      onTestFinished(off);
      await client.resources.retrieve(ids);
      await client.lineplots.rename(plot.key, "renamed");
      await expect.poll(() => latest?.map((r) => r.name)).toEqual(["renamed"]);
    });

    it("should drop a deleted resource from the answer", async () => {
      const proj = await client.projects.create({ name: randomToken() });
      const plot = await client.lineplots.create({
        project: proj.key,
        name: randomToken(),
      });
      const ids = [lineplot.resourceID(plot.key)];
      let latest: resource.Resource[] | undefined;
      const off = client.resources.watch(ids).onChange((answer) => {
        if (query.isLive(answer)) latest = answer;
      });
      onTestFinished(off);
      await client.resources.retrieve(ids);
      await client.lineplots.delete(plot.key);
      await expect.poll(() => latest).toEqual([]);
    });

    it("should add a resource created again with the key of a deleted one", async () => {
      const key = uuid.create();
      const name = randomToken();
      const set = async () =>
        await client.statuses.set({ key, name, variant: "info", message: "a" });
      await set();
      const ids = [status.resourceID(key)];
      let latest: resource.Resource[] | undefined;
      const off = client.resources.watch(ids).onChange((answer) => {
        if (query.isLive(answer)) latest = answer;
      });
      onTestFinished(off);
      await client.resources.retrieve(ids);
      await client.statuses.delete(key);
      await expect.poll(() => latest).toEqual([]);
      await set();
      await expect.poll(() => latest?.map((r) => r.name)).toEqual([name]);
    });

    it("should keep the order of the IDs when a resource is created again", async () => {
      const first = uuid.create();
      const second = uuid.create();
      const set = async (key: string) =>
        await client.statuses.set({ key, name: key, variant: "info", message: "a" });
      await set(first);
      await set(second);
      const ids = [status.resourceID(first), status.resourceID(second)];
      let latest: resource.Resource[] | undefined;
      const off = client.resources.watch(ids).onChange((answer) => {
        if (query.isLive(answer)) latest = answer;
      });
      onTestFinished(off);
      await client.resources.retrieve(ids);
      await client.statuses.delete(first);
      await expect.poll(() => latest?.map((r) => r.name)).toEqual([second]);
      await set(first);
      await expect.poll(() => latest?.map((r) => r.name)).toEqual([first, second]);
    });

    it("should name a resource once when the IDs repeat it", async () => {
      const l = await createLabel(randomToken());
      const ids = [l, l];
      expect((await client.resources.retrieve(ids)).map((r) => r.id)).toEqual([l]);
      const cached = client.resources.watch(ids).getCached();
      expect(query.isLive(cached) && cached.map((r) => r.id)).toEqual([l]);
    });

    it("should return the cached answer of a list read before", async () => {
      const l = await createLabel(randomToken());
      await client.resources.retrieve([l]);
      const cached = client.resources.watch([l]).getCached();
      expect(query.isLive(cached) && cached.map((r) => r.id)).toEqual([l]);
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

    it("should retrieve a resource created again with the key of a deleted one", async () => {
      const key = uuid.create();
      const name = randomToken();
      const set = async () =>
        await client.statuses.set({ key, name, variant: "info", message: "a" });
      await set();
      const id = status.resourceID(key);
      await client.resources.retrieve(id);
      let deleted = false;
      const off = client.resources.watch(id).onChange((answer) => {
        deleted = !query.isLive(answer);
      });
      onTestFinished(off);
      await client.statuses.delete(key);
      await expect.poll(() => deleted).toBe(true);
      await set();
      await expect
        .poll(async () => (await client.resources.retrieve(id)).name)
        .toEqual(name);
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
      await expect(read).rejects.toThrow(
        'Found no resource where id = "label:00000000-0000-0000-0000-000000000000"',
      );
    });
  });
});
