// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { group } from "@/group";
import { createTestClient } from "@/testutil";

const remote = createTestClient();

describe("channel paths on a client with no change stream", () => {
  it("should return the path that the Core gives after another client renames a folder", async () => {
    const folder = await remote.groups.create({ name: id.create(), scope: "channel" });
    const ch = await remote.channels.create({
      name: `path_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
      group: folder.key,
    });
    const bare = createTestClient({ cache: false });
    const request = { where: { key: { in: [ch.key] } }, includePath: true };
    const [before] = await bare.channels.retrieve(request);
    expect(before.path).toEqual(`${folder.name}.${ch.name}`);
    const renamed = id.create();
    await remote.groups.rename(folder.key, renamed);
    const [after] = await bare.channels.retrieve(request);
    expect(after.path).toEqual(`${renamed}.${ch.name}`);
  });

  it("should return the path that the Core gives after another client moves a folder", async () => {
    const [from, to] = await Promise.all([
      remote.groups.create({ name: id.create(), scope: "channel" }),
      remote.groups.create({ name: id.create(), scope: "channel" }),
    ]);
    const inner = await remote.groups.create({
      name: id.create(),
      parent: group.resourceID(from.key),
    });
    const ch = await remote.channels.create({
      name: `path_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
      group: inner.key,
    });
    const bare = createTestClient({ cache: false });
    const request = { where: { key: { in: [ch.key] } }, includePath: true };
    await bare.channels.retrieve(request);
    await remote.groups.setParent(inner.key, group.resourceID(to.key));
    const [after] = await bare.channels.retrieve(request);
    expect(after.path).toEqual(`${to.name}.${inner.name}.${ch.name}`);
  });
});
