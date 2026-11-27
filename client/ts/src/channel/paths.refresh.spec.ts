// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

import { group } from "@/group";
import { createTestClient, requestsTo, spyOnSend } from "@/testutil";

const remote = createTestClient();

describe("channel path folder reads on a client with a change stream", () => {
  beforeAll(async () => await remote.connect());

  it("should not read again a cached folder whose parent alone the client lacks", async () => {
    const outer = await remote.groups.create({ name: id.create(), scope: "channel" });
    const inner = await remote.groups.create({
      name: id.create(),
      parent: group.resourceID(outer.key),
    });
    const ch = await remote.channels.create({
      name: `path_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
      group: inner.key,
    });
    const client = createTestClient();
    await client.connect();
    // The change stream keeps the cached folder current.
    await client.groups.retrieve({ key: inner.key });
    expect(client.groups.store.has(outer.key)).toBe(false);
    const send = spyOnSend(client);
    onTestFinished(() => send.mockRestore());
    const [res] = await client.channels.retrieve({
      where: { key: { in: [ch.key] } },
      includePath: true,
    });
    expect(res.path).toEqual(`${outer.name}.${inner.name}.${ch.name}`);
    const naming = (target: string) =>
      requestsTo(send, target).filter((req) => JSON.stringify(req).includes(inner.key));
    expect(naming("/group/retrieve-ancestors")).toHaveLength(1);
    expect(naming("/group/retrieve")).toHaveLength(0);
  });
});
