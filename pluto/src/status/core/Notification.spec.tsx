// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Button } from "@/button";
import { Notification, type NotificationProps } from "@/status/core/Notification";

const mockSilence = vi.fn();

describe("Notification Component", () => {
  const notificationProps: NotificationProps = {
    status: {
      key: "test-key",
      name: "test-name",
      time: TimeStamp.now(),
      count: 1,
      message: "Test notification message",
      description: "Test notification description",
      variant: "info",
    },
    silence: mockSilence,
    actions: [
      <Button.Button key="action1">Action 1</Button.Button>,
      { key: "action2", children: "Action 2" },
    ],
  };

  it("renders notification message and description", () => {
    const c = render(<Notification {...notificationProps} />);

    expect(c.getByText("Test notification message")).toBeTruthy();
    expect(c.getByText("Test notification description")).toBeTruthy();
  });

  it("calls silence function when close button is clicked", () => {
    const c = render(<Notification {...notificationProps} />);

    const closeButton = c.getByRole("button", { name: /close/i });
    fireEvent.click(closeButton);

    expect(mockSilence).toHaveBeenCalledWith("test-key");
  });

  it("renders action buttons correctly", () => {
    const c = render(<Notification {...notificationProps} />);

    expect(c.getByText("Action 1")).toBeTruthy();
    expect(c.getByText("Action 2")).toBeTruthy();
  });
});
