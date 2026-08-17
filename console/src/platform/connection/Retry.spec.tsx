// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { RetrySchedule } from "@/platform/connection/Retry";
import { renderWithConsole } from "@/testutil";

const createDetails = (
  overrides: Partial<connection.StatusDetails> = {},
): connection.StatusDetails => ({
  authenticated: true,
  streamLive: true,
  streamDenied: false,
  epoch: 1,
  clusterKey: "cluster",
  clientVersion: "0.0.0",
  clientServerCompatible: true,
  clockSkew: TimeSpan.ZERO,
  clockSkewExceeded: false,
  retry: null,
  checking: false,
  ...overrides,
});

const line = (): HTMLElement | null =>
  document.querySelector(".console-connection-retry");

describe("RetrySchedule", () => {
  it("counts down to the next attempt", async () => {
    const retry = { attempt: 3, nextAt: TimeStamp.now().add(TimeSpan.seconds(5)) };
    await renderWithConsole(<RetrySchedule details={createDetails({ retry })} />);
    expect(screen.getByText("Retrying in 5s (attempt 3)")).toBeTruthy();
  });

  it("reports the check instead of the countdown while one runs", async () => {
    const retry = { attempt: 3, nextAt: TimeStamp.now().add(TimeSpan.seconds(5)) };
    await renderWithConsole(
      <RetrySchedule details={createDetails({ retry, checking: true })} />,
    );
    expect(screen.getByText("Checking connection")).toBeTruthy();
    expect(screen.queryByText(/Retrying in/)).toBeNull();
  });

  it("keeps its line when there is nothing scheduled", async () => {
    await renderWithConsole(<RetrySchedule details={createDetails()} />);
    expect(line()).toBeTruthy();
    expect(line()?.textContent).toBe("");
  });
});
