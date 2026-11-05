// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, status } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/device";

describe("StatusChip", () => {
  it("should name the status with a word from its variant", () => {
    const s = status.create<typeof device.statusDetailsZ>({
      key: device.statusKey("dev"),
      variant: "success",
      message: "Device connected",
      details: { rack: 1, device: "dev" },
    });
    const c = render(<Device.StatusChip status={s} />);
    expect(c.getByText("Connected").classList).toContain("lyra--status-success");
  });

  it("should name a missing status unknown", () => {
    const c = render(<Device.StatusChip />);
    expect(c.getByText("Unknown").classList).toContain("lyra--status-disabled");
  });
});
