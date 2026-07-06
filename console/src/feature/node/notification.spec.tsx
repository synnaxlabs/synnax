// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Node } from "@/feature/node";
import { Notifications } from "@/platform/notifications";
import { renderWithConsole } from "@/testutil";

interface HarnessProps {
  seed: status.Crude;
}

const Harness = ({ seed }: HarnessProps) => {
  const add = Status.useAdder();
  return (
    <>
      <button onClick={() => add(seed)}>add</button>
      <Notifications.Notifications adapters={Node.NOTIFICATION_ADAPTERS} />
    </>
  );
};
Harness.displayName = "Harness";

const addStatus = async (details?: Record<string, unknown>): Promise<void> => {
  await renderWithConsole(
    <Harness
      seed={{
        key: "version",
        variant: "warning",
        message: "Version mismatch",
        ...(details == null ? {} : { details }),
      }}
    />,
  );
  fireEvent.click(screen.getByText("add"));
};

describe("versionOutdatedAdapter", () => {
  beforeEach(() => {
    const root = document.createElement("div");
    root.id = "root";
    document.body.appendChild(root);
  });
  afterEach(() => document.getElementById("root")?.remove());

  it("should leave statuses without version details untouched", async () => {
    await addStatus();
    expect(screen.getByText("Version mismatch")).toBeTruthy();
    expect(screen.queryByText("Update Core")).toBeNull();
    expect(screen.queryByRole("button", { name: "Update" })).toBeNull();
  });

  it("should attach an update-core link when the server is outdated", async () => {
    await addStatus({
      type: Synnax.SERVER_VERSION_MISMATCH,
      oldServer: true,
      clientVersion: "1.0.0",
    });
    expect(screen.getByText("Update Core")).toBeTruthy();
  });

  it("should attach the console update action when the client is outdated", async () => {
    await addStatus({
      type: Synnax.SERVER_VERSION_MISMATCH,
      oldServer: false,
      clientVersion: "1.0.0",
    });
    expect(screen.getByRole("button", { name: "Update" })).toBeTruthy();
    expect(screen.queryByText("Update Core")).toBeNull();
  });
});
