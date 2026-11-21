// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, status } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/device";

interface CreateStatusArgs {
  variant: status.Variant;
  message: string;
  connected: boolean;
}

const createStatus = ({ connected, ...rest }: CreateStatusArgs): device.Status =>
  status.create<typeof device.statusDetailsZ>({
    details: { rack: 1, connected },
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
});
