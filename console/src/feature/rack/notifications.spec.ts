// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Rack } from "@/feature/rack";
import {
  createRackKey,
  renderForward,
  settleWrites,
  writeStatus,
} from "@/platform/notifications/testutil";

const client = createTestClient();

describe("Rack.NOTIFICATION_RULE", () => {
  it("should notify a rack warning at once", async () => {
    const key = rack.statusKey(await createRackKey(client));
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus(client, key, "warning");
    await settleWrites(client);
    expect(screen.getByText(message)).toBeTruthy();
  });

  it("should not notify a rack status that is not a problem", async () => {
    const key = rack.statusKey(await createRackKey(client));
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus(client, key, "info");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
  });
});
