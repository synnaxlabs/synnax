// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type UnaryClient } from "@synnaxlabs/freighter";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { type framer } from "@/framer";
import { group } from "@/group";
import { query } from "@/query";
import { type resource } from "@/resource";

const RACK: resource.ID = { type: "rack", key: "1" };
const OTHER_RACK: resource.ID = { type: "rack", key: "2" };

const createGroup = (
  key: string,
  overrides: Partial<group.Group> = {},
): group.Group => ({
  key,
  name: `group-${key}`,
  scope: "device",
  ...overrides,
});

const TOP = createGroup("00000000-0000-4000-8000-000000000001");
const UNDER_RACK = createGroup("00000000-0000-4000-8000-000000000002", {
  parent: RACK,
});
const UNDER_GROUP = createGroup("00000000-0000-4000-8000-000000000003", {
  parent: group.resourceID(UNDER_RACK.key),
});

interface Sent {
  target: string;
  req: unknown;
}

// A live stream that never delivers, so subscribed answers stay maintained.
const silentStreamer = (): framer.Streamer => {
  let closed = false;
  const streamer: framer.Streamer = {
    keys: [],
    update: async () => {},
    close: () => {
      closed = true;
    },
    next: async () => {
      while (!closed) await new Promise((resolve) => setTimeout(resolve, 5));
      return { done: true, value: undefined };
    },
    read: async () => {
      await streamer.next();
      throw new EOF();
    },
    [Symbol.asyncIterator]: () => streamer,
  };
  return streamer;
};

const createClient = (
  respond: (target: string, req: unknown) => unknown = () => ({}),
) => {
  const sent: Sent[] = [];
  const unary: UnaryClient = {
    use: () => {},
    send: async (target, req, reqZ, resZ) => {
      const parsed: unknown = reqZ.parse(req);
      sent.push({ target, req: parsed });
      return resZ.parse(respond(target, parsed));
    },
  };
  const cache = new query.Cache({ openStreamer: async () => silentStreamer() });
  onTestFinished(async () => await cache.close());
  return { client: new group.Client({ unary, cache }), sent };
};

describe("group request builders", () => {
  it("should build a create request", () => {
    expect(group.createRequest({ name: "Pumps", parent: RACK })).toEqual({
      endpoint: "/group/create",
      payload: { name: "Pumps", parent: RACK },
    });
  });

  it("should build a rename request", () => {
    expect(group.renameRequest(TOP.key, "Valves")).toEqual({
      endpoint: "/group/rename",
      payload: { key: TOP.key, name: "Valves" },
    });
  });

  it("should build a delete request from a single key", () => {
    expect(group.deleteRequest(TOP.key)).toEqual({
      endpoint: "/group/delete",
      payload: { keys: [TOP.key] },
    });
  });

  it("should build a set-parent request that moves groups to the top level", () => {
    expect(group.setParentRequest([TOP.key], null)).toEqual({
      endpoint: "/group/set-parent",
      payload: { keys: [TOP.key], parent: undefined },
    });
  });
});

describe("group client", () => {
  it("should create a group through the group route", async () => {
    const { client, sent } = createClient(() => ({ group: UNDER_RACK }));
    const created = await client.create({ name: UNDER_RACK.name, parent: RACK });
    expect(created).toEqual(UNDER_RACK);
    expect(sent).toEqual([
      { target: "/group/create", req: { name: UNDER_RACK.name, parent: RACK } },
    ]);
    expect(client.store.get(UNDER_RACK.key)).toEqual(UNDER_RACK);
  });

  it("should retrieve groups by key through the group route", async () => {
    const { client, sent } = createClient(() => ({ groups: [TOP] }));
    expect(await client.retrieve({ key: TOP.key })).toEqual(TOP);
    expect(sent).toEqual([{ target: "/group/retrieve", req: { keys: [TOP.key] } }]);
  });

  it("should keep a parent query live as matching groups land", async () => {
    const { client } = createClient(() => ({ groups: [UNDER_RACK] }));
    const params = { parents: [RACK] };
    const handler = vi.fn();
    const stop = client.onChange(params, handler);
    await client.retrieve(params);
    client.store.set([TOP, createGroup(UNDER_GROUP.key, { parent: OTHER_RACK })]);
    expect(client.getCached(params)).toEqual([UNDER_RACK]);
    const moved = createGroup(TOP.key, { parent: RACK });
    client.store.set(moved);
    expect(handler).toHaveBeenLastCalledWith([UNDER_RACK, moved]);
    stop();
  });

  it("should match nested and scope filters on the client", async () => {
    const { client } = createClient(() => ({ groups: [TOP] }));
    const params = { scopes: ["device" as const], nested: false };
    const stop = client.onChange(params, vi.fn());
    await client.retrieve(params);
    client.store.set([UNDER_RACK, createGroup(UNDER_GROUP.key, { scope: "channel" })]);
    expect(client.getCached(params)).toEqual([TOP]);
    stop();
  });

  it("should move groups optimistically and send the new parent", async () => {
    const { client, sent } = createClient();
    client.store.set([TOP]);
    await client.setParent(TOP.key, RACK);
    expect(client.store.get(TOP.key)?.parent).toEqual(RACK);
    expect(sent).toEqual([
      { target: "/group/set-parent", req: { keys: [TOP.key], parent: RACK } },
    ]);
  });

  it("should restore the parent when the move fails", async () => {
    const { client } = createClient(() => {
      throw new Error("refused");
    });
    client.store.set([UNDER_RACK]);
    await expect(client.setParent(UNDER_RACK.key, null)).rejects.toThrow("refused");
    expect(client.store.get(UNDER_RACK.key)?.parent).toEqual(RACK);
  });

  it("should retrieve ancestors and cache every group in the answer", async () => {
    const { client, sent } = createClient(() => ({
      groups: [UNDER_GROUP],
      ancestors: [UNDER_RACK],
    }));
    expect(await client.retrieveAncestors(UNDER_GROUP.key)).toEqual([UNDER_RACK]);
    expect(sent).toEqual([
      {
        target: "/group/retrieve",
        req: { keys: [UNDER_GROUP.key], ancestors: true },
      },
    ]);
    expect(client.store.get([UNDER_GROUP.key, UNDER_RACK.key])).toEqual([
      UNDER_GROUP,
      UNDER_RACK,
    ]);
  });
});
