// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("state", () => {
  it("should receive the initial control state from the cluster", async () => {
    const s = await client.control.openStateTracker();
    await expect.poll(() => s.states.size > 0).toBe(true);
    await s.close();
  });
});
