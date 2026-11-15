// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Core } from "@/feature/core";
import {
  CONNECTION_PARAMS,
  createCore,
  createCoreState,
} from "@/session/core/testutil";
import { createConnectedConsoleWrapper, renderWithConsole } from "@/testutil";

describe("Core.Settings", () => {
  it("should show the selected Core's address, status, and version", async () => {
    const core = createCore("Local");
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.Settings />, { wrapper });
    expect(await screen.findByText(`${core.host}:${core.port}`)).toBeTruthy();
    expect(await screen.findByText("Connected")).toBeTruthy();
    expect(await screen.findByText(/^v\d+\.\d+/)).toBeTruthy();
  });

  it("should show nothing when no Core is selected", async () => {
    const { container } = await renderWithConsole(<Core.Settings />);
    expect(container.textContent).toBe("");
  });
});
