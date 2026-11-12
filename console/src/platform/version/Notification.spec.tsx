// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status } from "@synnaxlabs/lyra/status";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithModals } from "@/platform/modals/testutil";
import { Version } from "@/platform/version";

const spec = (key: string): Status.NotificationSpec => ({
  key,
  name: "Version Update",
  variant: "info",
  message: "Update available",
  description: "",
  time: TimeStamp.now(),
  count: 1,
});

describe("version Notification", () => {
  it("should not match statuses that are not version updates", () => {
    expect(Version.Notification.match(spec("someOtherStatus"))).toBe(false);
  });

  it("should match version update statuses", () => {
    expect(Version.Notification.match(spec("versionUpdate-123"))).toBe(true);
  });

  it("should open the version info modal from its update action", async () => {
    renderWithModals(
      <Version.Notification status={spec("versionUpdate-123")} silence={vi.fn()} />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Update" }));
    await waitFor(() => expect(screen.getByText("Up to date")).toBeTruthy());
  });

  it("should be registered in NOTIFICATIONS", () => {
    expect(Version.NOTIFICATIONS).toContain(Version.Notification);
  });
});
