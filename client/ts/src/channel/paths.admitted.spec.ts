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

import type { channel } from "@/channel";
import { createTestClient } from "@/testutil";

const remote = createTestClient();

describe("channel paths of a channel that joins an answer", () => {
  beforeAll(async () => await remote.connect());

  it("should show the path of a channel made in a folder that a watched answer lists", async () => {
    const folder = await remote.groups.create({ name: id.create(), scope: "channel" });
    const later = createTestClient();
    onTestFinished(async () => await later.close());
    await later.connect();
    const live = later.channels.watch({
      where: { groups: [folder.key] },
      includePath: true,
    });
    let latest: channel.Channel[] = [];
    onTestFinished(live.onChange((w) => (latest = w?.records ?? [])));
    await live.retrieve();
    const ch = await remote.channels.create({
      name: `path_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
      group: folder.key,
    });
    await expect.poll(() => latest.some((c) => c.key === ch.key)).toBe(true);
    await expect
      .poll(() => latest.find((c) => c.key === ch.key)?.path)
      .toEqual(`${folder.name}.${ch.name}`);
  });
});
