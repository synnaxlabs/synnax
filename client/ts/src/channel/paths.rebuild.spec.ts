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

import { createTestClient } from "@/testutil";

const remote = createTestClient();

/** Longer than the debounce of a refetch and the trip to the Core. */
const SETTLE = TimeSpan.milliseconds(600);

const settle = async () =>
  await new Promise((resolve) => setTimeout(resolve, SETTLE.milliseconds));

describe("channel path rebuilds", () => {
  beforeAll(async () => await remote.connect());

  it.each(["device", "schematic"] as const)(
    "should not notify a watched answer with paths when a %s folder is created",
    async (scope) => {
      const folder = await remote.groups.create({
        name: id.create(),
        scope: "channel",
      });
      const ch = await remote.channels.create({
        name: `path_${id.create()}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        group: folder.key,
      });
      const client = createTestClient();
      onTestFinished(async () => await client.close());
      await client.connect();
      const live = client.channels.watch({
        where: { key: { in: [ch.key] } },
        includePath: true,
      });
      let changes = 0;
      onTestFinished(live.onChange(() => changes++));
      await live.retrieve();
      await settle();
      const before = changes;
      await remote.groups.create({ name: id.create(), scope });
      await settle();
      expect(changes - before).toEqual(0);
    },
  );
});
