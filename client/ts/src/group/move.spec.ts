// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { group } from "@/group";
import { type resource } from "@/resource";
import { createTestClient, newVirtualChannel } from "@/testutil";

const client = createTestClient();

describe("moveRequests", () => {
  const createGroup = async (): Promise<group.Group> =>
    await client.groups.create({ name: id.create(), scope: "channel" });

  it("should move records and groups under a group", async () => {
    const [parent, child, ch] = await Promise.all([
      createGroup(),
      createGroup(),
      newVirtualChannel(client),
    ]);
    await client.batch.send(
      group.moveRequests(
        client,
        [channel.resourceID(ch.key), group.resourceID(child.key)],
        group.resourceID(parent.key),
      ),
    );
    expect((await client.channels.retrieve(ch.key)).group).toEqual(parent.key);
    expect((await client.groups.retrieve({ key: child.key })).parent).toEqual(
      group.resourceID(parent.key),
    );
  });

  it("should move records out of their group when the parent is not a group", async () => {
    const [g, ch] = await Promise.all([createGroup(), newVirtualChannel(client)]);
    await client.batch.send([client.channels.setGroupRequest(ch.key, g.key)]);
    await client.batch.send(group.moveRequests(client, [channel.resourceID(ch.key)]));
    expect((await client.channels.retrieve(ch.key)).group).toBeUndefined();
  });

  it("should throw for a type that cannot sit in a group", () => {
    expect(() =>
      group.moveRequests(client, [{ type: "label", key: id.create() }]),
    ).toThrow("a label cannot sit in a group");
  });
});

describe("canMove", () => {
  const root: resource.ID = { type: "builtin", key: "devices" };
  const rack = (key: string): resource.ID => ({ type: "rack", key });
  const destination: resource.ID = { type: "group", key: "g" };
  const device: resource.ID = { type: "device", key: "d" };
  const holder: resource.ID = { type: "device", key: "p" };
  const ch: resource.ID = { type: "channel", key: "1" };
  const above: resource.ID = { type: "group", key: "p" };

  it("should move a rack or project only into a group at the top", () => {
    for (const type of ["rack", "project"] as const) {
      const moved = [{ id: { type, key: "m" }, ancestors: [root] }];
      expect(group.canMove({ root, destination, ancestors: [root], moved })).toBe(true);
      expect(
        group.canMove({ root, destination, ancestors: [rack("a"), root], moved }),
      ).toBe(false);
    }
  });

  it("should keep a device under its rack", () => {
    const moved = [{ id: device, ancestors: [rack("a"), root] }];
    expect(
      group.canMove({ root, destination, ancestors: [rack("a"), root], moved }),
    ).toBe(true);
    expect(
      group.canMove({ root, destination, ancestors: [rack("b"), root], moved }),
    ).toBe(false);
  });

  it("should keep a device held by another device in place", () => {
    const moved = [{ id: device, ancestors: [holder, rack("a"), root] }];
    expect(
      group.canMove({ root, destination, ancestors: [rack("a"), root], moved }),
    ).toBe(false);
  });

  it("should move a record that is not pinned into any group", () => {
    const moved = [{ id: ch, ancestors: [root] }];
    expect(group.canMove({ root, destination, ancestors: [above, root], moved })).toBe(
      true,
    );
  });
});
