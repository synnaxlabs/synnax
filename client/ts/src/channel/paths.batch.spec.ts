// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeSpan } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

import type { channel } from "@/channel";
import { group } from "@/group";
import { createTestClient, spyOnSend } from "@/testutil";

const remote = createTestClient();

/** Longer than the debounce of a refetch and the trip to the Core. */
const SETTLE = TimeSpan.milliseconds(600);

const settle = async () =>
  await new Promise((resolve) => setTimeout(resolve, SETTLE.milliseconds));

interface Request {
  where?: { key?: { in?: channel.Key[] } };
  includePath?: boolean;
}

describe("channel paths after one batch makes a folder and moves a channel into it", () => {
  beforeAll(async () => await remote.connect());

  it.each([
    ["at the top", false],
    ["inside another folder", true],
  ])(
    "should not fetch a watched answer again when another client makes a folder %s and moves the channel into it",
    async (_, nested) => {
      const outer = await remote.groups.create({ name: id.create(), scope: "channel" });
      const ch = await remote.channels.create({
        name: `path_${id.create()}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        group: nested ? outer.key : undefined,
      });
      const client = createTestClient();
      await client.connect();
      const live = client.channels.watch({
        where: { key: { in: [ch.key] } },
        includePath: true,
      });
      let latest: channel.Channel[] = [];
      onTestFinished(live.onChange((w) => (latest = w?.records ?? [])));
      await live.retrieve();
      await settle();
      const send = spyOnSend(client);
      onTestFinished(() => send.mockRestore());
      const key = crypto.randomUUID();
      const name = id.create();
      await remote.batch.send([
        nested
          ? remote.groups.createRequest({
              key,
              name,
              parent: group.resourceID(outer.key),
            })
          : remote.groups.createRequest({ key, name, scope: "channel" }),
        ...remote.channels.setGroupRequest(ch.key, key),
      ]);
      const path = nested ? `${outer.name}.${name}.${ch.name}` : `${name}.${ch.name}`;
      await expect.poll(() => latest[0]?.path).toEqual(path);
      await settle();
      // The change stream carries the new folder and the move, so the client builds the
      // path with no fetch.
      const refetches = send.mock.calls.filter(([target, req]) => {
        if (target !== "/channel/retrieve") return false;
        const { where, includePath } = req as Request;
        return includePath === true && where?.key?.in?.includes(ch.key) === true;
      });
      expect(refetches).toHaveLength(0);
    },
  );
});
