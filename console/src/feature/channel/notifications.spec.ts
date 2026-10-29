// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, type status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { renderForward, settleWrites } from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

/** @returns the message of the status. */
const writeStatus = async (variant: status.Variant): Promise<string> => {
  const source = await client.channels.create({
    name: uniqueName("source"),
    dataType: DataType.FLOAT32,
    virtual: true,
  });
  const calculated = await client.channels.create({
    name: uniqueName("calculated"),
    dataType: DataType.FLOAT32,
    virtual: true,
    expression: `return ${source.name} * 2`,
  });
  const { message } = await client.statuses.set({
    key: channel.statusKey(calculated.key),
    name: calculated.name,
    variant,
    message: uniqueName("message"),
    time: TimeStamp.now(),
  });
  return message;
};

describe("Channel.NOTIFICATION_RULE", () => {
  it("should notify a calculated channel warning at once", async () => {
    await renderForward(client, { channel: Channel.NOTIFICATION_RULE });
    const message = await writeStatus("warning");
    await settleWrites(client);
    expect(screen.getByText(message)).toBeTruthy();
  });

  it("should not notify a channel status that is not a problem", async () => {
    await renderForward(client, { channel: Channel.NOTIFICATION_RULE });
    const message = await writeStatus("info");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
  });
});
