// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, type status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Rack } from "@/feature/rack";
import {
  createRackKey,
  renderForward,
  settleWrites,
} from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

/** @returns the message of the status. */
const writeStatus = async (
  key: status.Key,
  variant: status.Variant,
): Promise<string> => {
  const { message } = await client.statuses.set({
    key,
    name: uniqueName("rack"),
    variant,
    message: uniqueName("message"),
    time: TimeStamp.now(),
  });
  return message;
};

describe("Rack.NOTIFICATION_RULE", () => {
  it("should notify a rack warning at once", async () => {
    const key = rack.statusKey(await createRackKey(client));
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus(key, "warning");
    await settleWrites(client);
    expect(screen.getByText(message)).toBeTruthy();
  });

  it("should not notify a rack status that is not a problem", async () => {
    const key = rack.statusKey(await createRackKey(client));
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus(key, "info");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
  });
});
