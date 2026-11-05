// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Version } from "@/feature/version";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  CONNECTION_PARAMS,
  createCore,
  createCoreState,
} from "@/session/core/testutil";
import {
  createConsoleWrapper,
  INCOMPATIBLE_UNLICENSED_STATUS,
  type TestStore,
} from "@/testutil";

const WORKSPACE = "workspace";
const TITLE = "Console 0.61.0 cannot work with Core 0.60.1";

const INCOMPATIBLE_STATUS: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "success",
  message: "Connected",
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    clientVersion: "0.61.0",
    nodeVersion: "0.60.1",
    clientServerCompatible: false,
  },
};

const COMPATIBLE_STATUS: connection.Status = {
  ...INCOMPATIBLE_STATUS,
  details: {
    ...INCOMPATIBLE_STATUS.details,
    nodeVersion: "0.61.0",
    clientServerCompatible: true,
  },
};

interface GuardHarness {
  store: TestStore;
  setStatus: (status: connection.Status) => void;
}

const renderGuard = async (
  status: connection.Status,
  secure: boolean = false,
): Promise<GuardHarness> => {
  const core = createCore("Local", { secure });
  const { wrapper, store } = await createConsoleWrapper({
    client: null,
    preloadedState: createCoreState([core], core.key),
  });
  const ui = (current: connection.Status): ReactElement => (
    <Synnax.TestProvider client={null} status={current}>
      <Version.Guard>
        <span>{WORKSPACE}</span>
      </Version.Guard>
    </Synnax.TestProvider>
  );
  const { rerender } = render(ui(status), { wrapper });
  return { store, setStatus: (next) => rerender(ui(next)) };
};

const getOpenInBrowserLink = (): string | null =>
  screen.getByRole("link", { name: /Open in browser/ }).getAttribute("href");

const { host, port } = CONNECTION_PARAMS;

describe("Version.Guard", () => {
  it("should show the mismatch screen in place of the workspace", async () => {
    await renderGuard(INCOMPATIBLE_STATUS);
    expect(screen.getByText(TITLE)).toBeTruthy();
    expect(
      screen.getByText("The Console and the Core must have the same minor version."),
    ).toBeTruthy();
    expect(screen.queryByText(WORKSPACE)).toBeNull();
  });

  it("should show the mismatch screen while the Core is also unlicensed", async () => {
    await renderGuard(INCOMPATIBLE_UNLICENSED_STATUS);
    expect(screen.getByText(TITLE)).toBeTruthy();
    expect(screen.queryByText(WORKSPACE)).toBeNull();
  });

  it("should lift the screen once the Core reports a matching version", async () => {
    const { setStatus } = await renderGuard(INCOMPATIBLE_STATUS);
    expect(screen.queryByText(WORKSPACE)).toBeNull();
    setStatus(COMPATIBLE_STATUS);
    expect(screen.getByText(WORKSPACE)).toBeTruthy();
    expect(screen.queryByText(TITLE)).toBeNull();
  });

  it("should open the Console that the Core serves", async () => {
    await renderGuard(INCOMPATIBLE_STATUS);
    expect(getOpenInBrowserLink()).toBe(`http://${host}:${port}`);
  });

  it("should open the served Console over TLS for a secure Core", async () => {
    await renderGuard(INCOMPATIBLE_STATUS, true);
    expect(getOpenInBrowserLink()).toBe(`https://${host}:${port}`);
  });

  it("should clear the selected Core to choose another one", async () => {
    const { store } = await renderGuard(INCOMPATIBLE_STATUS);
    expect(Session.Core.selectSelectedKey(store.getState())).toBeDefined();
    fireEvent.click(findButton("Choose another Core"));
    expect(Session.Core.selectSelectedKey(store.getState())).toBeUndefined();
  });

  it("should render the workspace before a check reports a version", async () => {
    await renderGuard(connection.DEFAULT_STATUS);
    expect(screen.getByText(WORKSPACE)).toBeTruthy();
  });
});
