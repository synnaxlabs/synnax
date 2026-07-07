// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { describe, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { PagerDuty } from "@/feature/pagerduty";
import { stubGeometry, waitForPlacedLayout } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("PagerDuty.Task Commands", () => {
  it("should place the alert task layout from the create alert task command", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: PagerDuty.Task.COMMANDS,
      client,
    });
    await openCommandPalette("Create a PagerDuty");
    await selectCommand("Create a PagerDuty Alert Task");
    await waitForPlacedLayout(store, PagerDuty.Task.ALERT_TYPE);
  });
});
