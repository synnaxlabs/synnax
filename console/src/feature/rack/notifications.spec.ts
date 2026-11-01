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
import { TimeSpan } from "@synnaxlabs/x";
import { act, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Rack } from "@/feature/rack";
import {
  renderForward,
  settleWrites,
  writeStatus,
} from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

/** @returns the status key of a new rack, without the status that the Core gives it. */
const createRackKey = async (): Promise<status.Key> => {
  const r = await client.racks.create({ name: uniqueName("rack") });
  const key = rack.statusKey(r.key);
  // The Core gives a new rack an open "Rack status unknown" problem.
  await client.statuses.delete(key);
  return key;
};

describe("Rack.NOTIFICATION_RULE", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  const fakeTimers = () =>
    vi.useFakeTimers({
      toFake: ["setTimeout", "clearTimeout"],
      shouldAdvanceTime: true,
    });

  it("should notify a rack warning after 5 s", async () => {
    fakeTimers();
    const key = await createRackKey();
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus(client, key, "warning");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
    act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
    expect(screen.getByText(message)).toBeTruthy();
  });

  it("should not notify the unknown status of a rack whose Driver reports", async () => {
    fakeTimers();
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const name = uniqueName("rack");
    const r = await client.racks.create({ name });
    await settleWrites(client);
    await writeStatus(client, rack.statusKey(r.key), "success");
    await settleWrites(client);
    act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
    expect(screen.queryByText(name)).toBeNull();
  });

  it("should not notify a rack status that is not a problem", async () => {
    const key = await createRackKey();
    await renderForward(client, { rack: Rack.NOTIFICATION_RULE });
    const message = await writeStatus(client, key, "info");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
  });
});
