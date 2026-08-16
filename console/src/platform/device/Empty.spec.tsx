// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";

const DEVICE = { key: "dev-key", name: "My Device" };

describe("device empty states", () => {
  it("should render the none-selected message", () => {
    render(<Device.NoneSelected />);
    expect(screen.getByText("No device selected")).toBeTruthy();
  });

  it("should prompt to configure and hand the device key to onConfigure", () => {
    const onConfigure = vi.fn();
    render(
      <Device.Unconfigured device={DEVICE} canConfigure onConfigure={onConfigure} />,
    );
    expect(screen.getByText(`${DEVICE.name} is not configured.`)).toBeTruthy();
    fireEvent.click(screen.getByText(`Configure ${DEVICE.name}`));
    expect(onConfigure).toHaveBeenCalledWith(DEVICE.key);
  });

  it("should hide the configure action when configuration is not allowed", () => {
    render(
      <Device.Unconfigured
        device={DEVICE}
        canConfigure={false}
        onConfigure={vi.fn()}
      />,
    );
    expect(screen.getByText(`${DEVICE.name} is not configured.`)).toBeTruthy();
    expect(screen.queryByText(`Configure ${DEVICE.name}`)).toBeNull();
  });
});
