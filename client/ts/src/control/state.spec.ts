// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { newClient } from "@/setupspecs";

const client = newClient();

describe("state", () => {
  it("should receive the initial control state from the cluster", async () => {
    const s = await client.control.openStateTracker();
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(s.states.size).toBeGreaterThan(0);
    await s.close();
  });
});
