// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { sleep, TimeSpan } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { describe, expect, it } from "vitest";

import { startRackHeartbeat } from "@/platform/device/testutil";

describe("startRackHeartbeat", () => {
  it("should reject stop with the error of a report that failed", async () => {
    const client = createTestClient();
    const rck = await client.racks.create({
      name: uniqueName("rack"),
      integrations: [],
    });
    const stop = await startRackHeartbeat(client, rck);
    await client.close();
    await sleep.sleep(TimeSpan.seconds(1.5));
    await expect(stop()).rejects.toThrow("no credentials");
  });
});
