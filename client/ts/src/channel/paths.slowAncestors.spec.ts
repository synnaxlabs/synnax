// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware } from "@synnaxlabs/freighter";
import { DataType, id, TimeSpan } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

import type { channel } from "@/channel";
import { group } from "@/group";
import { createTestClient, requestsTo, spyOnSend } from "@/testutil";

const remote = createTestClient();

/** Longer than the debounce of a refetch and the trip to the Core. */
const SETTLE = TimeSpan.milliseconds(800);

const sleep = async (span: TimeSpan) =>
  await new Promise((resolve) => setTimeout(resolve, span.milliseconds));

describe("channel paths when the folders above a folder come back late", () => {
  beforeAll(async () => await remote.connect());

  it("should read the folders above a folder once when the folder comes back before them", async () => {
    const outer = await remote.groups.create({ name: id.create(), scope: "channel" });
    const inner = await remote.groups.create({
      name: id.create(),
      parent: group.resourceID(outer.key),
    });
    const ch = await remote.channels.create({
      name: `path_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const slow: Middleware = async (ctx, next) => {
      if (ctx.target.endsWith("/group/retrieve-ancestors"))
        await sleep(TimeSpan.milliseconds(300));
      return await next(ctx);
    };
    const client = createTestClient({ middleware: [slow] });
    await client.connect();
    const live = client.channels.watch({
      where: { key: { in: [ch.key] } },
      includePath: true,
    });
    let latest: channel.Channel[] = [];
    onTestFinished(live.onChange((w) => (latest = w?.records ?? [])));
    await live.retrieve();
    const send = spyOnSend(client);
    onTestFinished(() => send.mockRestore());
    await remote.batch.send(remote.channels.setGroupRequest(ch.key, inner.key));
    await expect
      .poll(() => latest[0]?.path)
      .toEqual(`${outer.name}.${inner.name}.${ch.name}`);
    await sleep(SETTLE);
    const reads = requestsTo(send, "/group/retrieve-ancestors").filter((req) =>
      JSON.stringify(req).includes(inner.key),
    );
    expect(reads).toHaveLength(1);
  });
});
