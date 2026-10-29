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
import { renderForward, settleWrites } from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

/** @returns the message of the status. */
const writeStatus = async (variant: status.Variant): Promise<string> => {
  const { message } = await client.statuses.set({
    // The Core writes statuses of its own on the key of each rack that exists.
    key: rack.statusKey(Math.floor(Math.random() * 2 ** 32)),
    name: uniqueName("rack"),
    variant,
    message: uniqueName("message"),
    time: TimeStamp.now(),
  });
  return message;
};

describe("Rack.NOTIFICATION_RULE", () => {
  it("should notify a rack warning at once", async () => {
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus("warning");
    await settleWrites(client);
    expect(screen.getByText(message)).toBeTruthy();
  });

  it("should not notify a rack status that is not a problem", async () => {
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus("info");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
  });
});
