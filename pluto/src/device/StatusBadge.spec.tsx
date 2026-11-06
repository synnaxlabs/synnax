// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, status } from "@synnaxlabs/client";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { TimeSpan } from "@synnaxlabs/x";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Device } from "@/device";

const BADGE_RECT = {
  left: 100,
  top: 100,
  right: 200,
  bottom: 120,
  width: 100,
  height: 20,
  x: 100,
  y: 100,
  toJSON: () => ({}),
} as DOMRect;

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config delay={TimeSpan.milliseconds(1)}>{children}</Tooltip.Config>
  </Triggers.Provider>
);

interface CreateStatusArgs {
  variant: status.Variant;
  message: string;
  connected: boolean;
  description?: string;
}

const createStatus = ({ connected, ...rest }: CreateStatusArgs): device.Status =>
  status.create<typeof device.statusDetailsZ>({
    key: device.statusKey("dev"),
    details: { rack: 1, device: "dev", connected },
    ...rest,
  });

describe("StatusBadge", () => {
  it("should say connected when the Driver reaches the device", () => {
    const s = createStatus({
      variant: "success",
      message: "Device connected",
      connected: true,
    });
    const c = render(<Device.StatusBadge status={s} />);
    expect(c.getByText("Connected").classList).toContain("lyra--status-success");
  });

  it("should say connected in the color of an error from the device", () => {
    const s = createStatus({
      variant: "error",
      message: "Health check failed",
      connected: true,
    });
    const c = render(<Device.StatusBadge status={s} />);
    expect(c.getByText("Connected").classList).toContain("lyra--status-error");
  });

  it("should say disconnected when the Driver cannot reach the device", () => {
    const s = createStatus({
      variant: "warning",
      message: "Failed to reach device",
      connected: false,
    });
    const c = render(<Device.StatusBadge status={s} />);
    expect(c.getByText("Disconnected").classList).toContain("lyra--status-warning");
  });

  it("should say disconnected for a device without a status", () => {
    const c = render(<Device.StatusBadge />);
    expect(c.getByText("Disconnected").classList).toContain("lyra--status-disabled");
  });

  it("should show the message and the description in its tooltip", async () => {
    const s = createStatus({
      variant: "error",
      message: "Health check failed",
      connected: true,
      description: "The server returned 503.",
    });
    const c = render(<Device.StatusBadge status={s} />, { wrapper: Wrapper });
    const badge = c.getByText("Connected");
    badge.getBoundingClientRect = () => BADGE_RECT;
    fireEvent.pointerOver(badge, { pointerType: "mouse" });
    await waitFor(() =>
      expect(document.querySelector(".lyra-tooltip")?.textContent).toBe(
        "Health check failedThe server returned 503.",
      ),
    );
  });
});
