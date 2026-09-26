// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { Embedded } from "@/feature/embedded";
import {
  clearSupervisor,
  createDesktopWrapper,
  mockSupervisor,
} from "@/feature/embedded/testutil";

const { host, port, username, password } = TEST_CLIENT_PARAMS;

describe("Embedded.Indicator", () => {
  afterEach(clearSupervisor);

  it("should show while the Core cannot be reached", async () => {
    // Nothing listens on the port, as in the gap while the Core restarts.
    mockSupervisor(() => ({
      state: "running",
      connection: { host: "127.0.0.1", port: 1, username, password, version: "0.0.0" },
    }));
    const { wrapper } = await createDesktopWrapper();
    render(<Embedded.Indicator />, { wrapper });
    expect(await screen.findByText("Reconnecting")).toBeTruthy();
  });

  it("should stay hidden while the Core is healthy", async () => {
    mockSupervisor(() => ({
      state: "running",
      connection: { host, port: Number(port), username, password, version: "0.0.0" },
    }));
    const { wrapper } = await createDesktopWrapper();
    render(
      <Embedded.Guard>
        <Embedded.Indicator />
        <span>workspace</span>
      </Embedded.Guard>,
      { wrapper },
    );
    // The guard shows the workspace only after the client connects.
    expect(await screen.findByText("workspace")).toBeTruthy();
    expect(screen.queryByText("Reconnecting")).toBeNull();
  });
});
