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

import { channel } from "@/channel";
import { group } from "@/group";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  createTestUser,
} from "@/testutil";

const root = createTestClient();

describe("channel paths after the caller loses a folder", () => {
  it("should keep the cached path after the caller loses a folder on it", async () => {
    const folder = await root.groups.create({ name: id.create(), scope: "channel" });
    const ch = await root.channels.create({
      name: `path_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
      group: folder.key,
    });
    const role = await createEmptyRole(root);
    const [, folderPolicy] = await root.access.policies.create([
      {
        name: "channel",
        role: role.key,
        objects: [channel.resourceID(ch.key)],
        actions: ["retrieve"],
      },
      {
        name: "folder",
        role: role.key,
        objects: [group.resourceID(folder.key)],
        actions: ["retrieve"],
      },
    ]);
    const { username, password } = await createTestUser(root, { role: role.key });
    const restricted = createTestClientAs(username, password);
    const request = { where: { key: { in: [ch.key] } }, includePath: true };
    const [before] = await restricted.channels.retrieve(request);
    expect(before.path).toEqual(`${folder.name}.${ch.name}`);
    await root.access.policies.delete(folderPolicy.key);
    const fresh = createTestClientAs(username, password);
    const [core] = await fresh.channels.retrieve(request);
    expect(core.path).toBeUndefined();
    // The client builds the path from the folder it cached while the caller could read
    // it, until the cache drops the folder.
    const [after] = await restricted.channels.retrieve(request);
    expect(after.path).toEqual(before.path);
  });
});
