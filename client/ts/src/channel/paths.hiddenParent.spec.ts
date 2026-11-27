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
import type { Synnax } from "@/index";
import {
  createEmptyRole,
  createTestClient,
  createTestClientAs,
  createTestUser,
} from "@/testutil";

const root = createTestClient();

type Connect = (username: string, password: string) => Synnax;

const CLIENTS: Array<[string, Connect]> = [
  [
    "with no change stream",
    (username, password) => createTestClient({ username, password, cache: false }),
  ],
  // The role cannot read the channels of the change stream, so the Core refuses it.
  ["whose change stream the Core refuses", createTestClientAs],
];

describe("channel paths under a folder whose parent the caller cannot read", () => {
  it.each(CLIENTS)(
    "should leave the path absent on a client %s after another client moves the folder under a folder the caller cannot read",
    async (_, connect) => {
      const folder = await root.groups.create({ name: id.create(), scope: "channel" });
      const ch = await root.channels.create({
        name: `path_${id.create()}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        group: folder.key,
      });
      const role = await createEmptyRole(root);
      await root.access.policies.create([
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
      const restricted = connect(username, password);
      const request = { where: { key: { in: [ch.key] } }, includePath: true };
      const [before] = await restricted.channels.retrieve(request);
      expect(before.path).toEqual(`${folder.name}.${ch.name}`);
      const hidden = await root.groups.create({ name: id.create(), scope: "channel" });
      await root.groups.setParent(folder.key, group.resourceID(hidden.key));
      const fresh = createTestClient({ username, password, cache: false });
      const [core] = await fresh.channels.retrieve(request);
      expect(core.path).toBeUndefined();
      const [after] = await restricted.channels.retrieve(request);
      expect(after.path).toBeUndefined();
    },
  );
});
