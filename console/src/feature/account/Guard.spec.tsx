// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  connection,
  type license,
  MissingLicenseError,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Account } from "@/feature/account";
import { License } from "@/platform/license";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const MESSAGE = "No license is active on this Core";

const UNLICENSED: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "error",
  message: MESSAGE,
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    reason: "unlicensed",
    error: new MissingLicenseError(MESSAGE),
  },
};

const MISSING: license.Info = {
  state: "missing",
  warning: undefined,
  fingerprint: ["aa", "bb"],
  license: undefined,
};

interface RenderOptions {
  status?: connection.Status;
  account?: Session.Account.SliceState;
}

const renderGuard = async ({ status, account }: RenderOptions = {}): Promise<{
  client: Client;
  store: TestStore;
}> => {
  const client = createTestClient();
  vi.spyOn(client.license, "retrieve").mockResolvedValue(MISSING);
  const { wrapper: Console, store } = await createConsoleWrapper({
    client: null,
    preloadedState:
      account == null ? undefined : { [Session.Account.SLICE_NAME]: account },
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Synnax.TestProvider client={client} status={status}>
        {children}
      </Synnax.TestProvider>
    </Console>
  );
  Wrapper.displayName = "GuardWrapper";
  render(
    <Account.Guard>
      <span>licensed content</span>
    </Account.Guard>,
    { wrapper: Wrapper },
  );
  return { client, store };
};

describe("Account.Guard", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should render children while the Core is not refused for a license", async () => {
    await renderGuard();
    expect(screen.getByText("licensed content")).toBeTruthy();
  });

  it("should ask for a sign-in while the Core is unlicensed", async () => {
    await renderGuard({ status: UNLICENSED });
    expect(screen.getByText("Sign in to continue")).toBeTruthy();
    expect(screen.queryByText("licensed content")).toBeNull();
  });

  it("should say the sign-in lapsed on a machine that was linked", async () => {
    await renderGuard({
      status: UNLICENSED,
      account: { version: 0, email: "someone@example.com" },
    });
    expect(screen.getByText("Your sign-in has lapsed")).toBeTruthy();
    expect(screen.getByText(/someone@example\.com/)).toBeTruthy();
  });

  it("should open the portal with the state it minted", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const { store } = await renderGuard({ status: UNLICENSED });
    const signIn = findButton("Sign in");
    await waitFor(() => expect(signIn.getAttribute("aria-disabled")).toBeNull());
    fireEvent.click(signIn);
    await waitFor(() => expect(open).toHaveBeenCalled());
    const url = new URL(String(open.mock.calls[0][0]));
    expect(url.origin + url.pathname).toBe(License.PORTAL_SIGN_IN_URL);
    expect(url.searchParams.get("fp")).toBe("aa, bb");
    expect(url.searchParams.get("name")).toBe(Account.DEFAULT_MACHINE_NAME);
    expect(url.searchParams.get("state")).toBe(
      Session.Account.select(store.getState()).pending,
    );
    expect(await screen.findByText("Waiting for your browser...")).toBeTruthy();
  });

  it("should ask to try again while the machine is offline", async () => {
    vi.spyOn(navigator, "onLine", "get").mockReturnValue(false);
    await renderGuard({ status: UNLICENSED });
    expect(screen.getByText("You are offline")).toBeTruthy();
    expect(findButton("Try again")).toBeTruthy();
    expect(screen.queryByText("Sign in")).toBeNull();
  });

  it("should offer the license file screen and a way back", async () => {
    await renderGuard({ status: UNLICENSED });
    fireEvent.click(findButton("Use a license file"));
    expect(screen.getByPlaceholderText("Paste the token")).toBeTruthy();
    expect(screen.queryByText("Log out")).toBeNull();
    fireEvent.click(findButton("Back"));
    expect(screen.getByText("Sign in to continue")).toBeTruthy();
  });
});
