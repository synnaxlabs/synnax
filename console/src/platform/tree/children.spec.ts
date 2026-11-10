// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, query } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { createTestDevice } from "@/platform/device/testutil";
import { Tree } from "@/platform/tree";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Tree.createSource", () => {
  it("should keep a resource when only a field outside it changes", async () => {
    const dev = await createTestDevice(client);
    const source = Tree.createSource(
      client.devices,
      { ...query.keysRequest([dev.key]), includeStatus: true },
      Tree.toResource("device"),
    );
    const {
      resources: [before],
    } = await source.retrieve();
    let release = () => {};
    const changed = new Promise<void>((resolve) => {
      release = source.onChange(() => resolve());
    });
    await client.statuses.set({
      owner: device.resourceID(dev.key),
      name: "",
      variant: "success",
      message: uniqueName("message"),
      time: TimeStamp.now(),
    });
    await changed;
    release();
    expect(source.getCached()?.resources[0]).toBe(before);
  });
});
