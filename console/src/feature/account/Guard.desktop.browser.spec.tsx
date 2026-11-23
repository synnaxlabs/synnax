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
  isConnectionError,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it, onTestFailed, vi } from "vitest";

import { Account } from "@/feature/account";
import { Modals } from "@/platform/modals";
import { findButton, getSwitch } from "@/platform/modals/testutil";
import { Portal } from "@/platform/portal";
import { TEST_URL } from "@/platform/portal/testutil";
import { Session } from "@/session";
import {
  CaptureStatuses,
  createConsoleWrapper,
  createStatusConsoleWrapper,
  type TestStore,
  UNLICENSED_STATUS,
} from "@/testutil";

interface Read {
  target?: string;
  start: number;
  end?: number;
  failure?: string;
}

/**
 * Creates a client that times each license read, each request it sends, and each
 * retry, so a failed test reports whether the Log in click reached its read and which
 * request it waited on: the login or the license read itself.
 */
const createReportingClient = (): Client => {
  const begun = performance.now();
  const retries: string[] = [];
  const client = createTestClient({
    onRetry: (error) =>
      retries.push(`at ${Math.round(performance.now() - begun)} ms: ${error.message}`),
  });
  const reads: Read[] = [];
  const requests: Read[] = [];
  // Each request, the login included, goes out through fetch.
  const fetch = globalThis.fetch.bind(globalThis);
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = input instanceof Request ? input.url : String(input);
    const request: Read = { target: new URL(url).pathname, start: performance.now() };
    requests.push(request);
    try {
      return await fetch(input, init);
    } finally {
      request.end = performance.now();
    }
  });
  const retrieve = client.license.retrieve.bind(client.license);
  vi.spyOn(client.license, "retrieve").mockImplementation(async () => {
    const read: Read = { start: performance.now() };
    reads.push(read);
    const result = retrieve();
    result.then(
      () => {
        read.end = performance.now();
      },
      (error: unknown) => {
        read.end = performance.now();
        read.failure = error instanceof Error ? error.message : typeof error;
      },
    );
    return await result;
  });
  onTestFailed(() => {
    const now = performance.now();
    const format = (list: Read[]): string =>
      list.length === 0
        ? "none"
        : list
            .map(({ target, start, end, failure }) => {
              const at = `${target ?? ""} at ${Math.round(start - begun)} ms`.trim();
              if (end == null)
                return `${at}, still waiting after ${Math.round(now - start)} ms`;
              const took = `${at}, took ${Math.round(end - start)} ms`;
              return failure == null ? took : `${took}, failed: ${failure}`;
            })
            .join("; ");
    // A browser spec has no process to write to stderr through.
    console.error(`license reads: ${format(reads)}`);
    console.error(`requests: ${format(requests)}`);
    console.error(`retries: ${retries.length === 0 ? "none" : retries.join("; ")}`);
  });
  return client;
};

interface RenderOptions {
  status?: connection.Status;
  account?: Session.Account.SliceState;
}

const renderGuard = async ({ status, account }: RenderOptions = {}): Promise<{
  client: Client;
  store: TestStore;
}> => {
  const client = createReportingClient();
  const { wrapper, store } = await createStatusConsoleWrapper({
    client,
    status,
    preloadedState:
      account == null ? undefined : { [Session.Account.SLICE_NAME]: account },
  });
  render(
    <>
      <Modals.Stack />
      <Account.Guard>
        <span>licensed content</span>
      </Account.Guard>
    </>,
    { wrapper },
  );
  return { client, store };
};

describe("Account.Guard", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should render children while the Core is not refused for a license", async () => {
    await renderGuard();
    expect(screen.getByText("licensed content")).toBeTruthy();
  });

  it("should ask for a login while the Core is unlicensed", async () => {
    await renderGuard({ status: UNLICENSED_STATUS });
    expect(screen.getByText("Log in to continue")).toBeTruthy();
    expect(screen.queryByText("licensed content")).toBeNull();
  });

  it("should render children once the Core is licensed", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({ client: null });
    const ui = (status: connection.Status): ReactElement => (
      <Synnax.TestProvider client={client} status={status}>
        <Account.Guard>
          <span>licensed content</span>
        </Account.Guard>
      </Synnax.TestProvider>
    );
    const { rerender } = render(ui(UNLICENSED_STATUS), { wrapper });
    expect(screen.getByText("Log in to continue")).toBeTruthy();
    rerender(ui(connection.DEFAULT_STATUS));
    expect(await screen.findByText("licensed content")).toBeTruthy();
    expect(screen.queryByText("Log in to continue")).toBeNull();
  });

  it("should say the login lapsed on a machine that was linked", async () => {
    await renderGuard({
      status: UNLICENSED_STATUS,
      account: {
        ...Session.Account.ZERO_SLICE_STATE,
        link: { secret: "shh", email: "someone@example.com", user: "user_a" },
      },
    });
    expect(screen.getByText("Your login has lapsed")).toBeTruthy();
    expect(screen.getByText(/someone@example\.com/)).toBeTruthy();
  });

  it("should open the usage data settings on a machine that was linked", async () => {
    await renderGuard({
      status: UNLICENSED_STATUS,
      account: {
        ...Session.Account.ZERO_SLICE_STATE,
        link: { secret: "shh", email: "someone@example.com", user: "user_a" },
        analyticsAllowed: true,
      },
    });
    fireEvent.click(findButton("Usage data"));
    await screen.findByText("Share usage data");
    expect(getSwitch("Share usage data").checked).toBe(true);
  });

  it("should not offer usage data on a machine that was never linked", async () => {
    await renderGuard({ status: UNLICENSED_STATUS });
    expect(screen.queryByText("Usage data")).toBeNull();
  });

  it("should open the hub with the state it minted", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const { client, store } = await renderGuard({ status: UNLICENSED_STATUS });
    fireEvent.click(findButton("Log in"));
    await waitFor(() => expect(open).toHaveBeenCalled());
    const url = new URL(String(open.mock.calls[0][0]));
    expect(url.origin + url.pathname).toBe(`${TEST_URL}/desktop/login`);
    const { fingerprint } = await client.license.retrieve();
    expect(url.searchParams.get("fp")).toBe(Portal.joinFingerprint(fingerprint));
    expect(url.searchParams.get("name")).toBe("Synnax Desktop");
    expect(url.searchParams.get("state")).toBe(
      Session.Account.selectPending(store.getState()),
    );
    expect(await screen.findByText("Finish in your browser")).toBeTruthy();
  });

  it("should send the same state when the page opens again", async () => {
    // A new state would make the first page's login link fail to finish.
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const { store } = await renderGuard({ status: UNLICENSED_STATUS });
    fireEvent.click(findButton("Log in"));
    expect(await screen.findByText("Finish in your browser")).toBeTruthy();
    fireEvent.click(findButton("Open the page again"));
    await waitFor(() => expect(open).toHaveBeenCalledTimes(2));
    const states = open.mock.calls.map(([url]) =>
      new URL(String(url)).searchParams.get("state"),
    );
    expect(states[1]).toBe(states[0]);
    expect(Session.Account.selectPending(store.getState())).toBe(states[0]);
  });

  it("should finish a login left pending from before", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    await renderGuard({
      status: UNLICENSED_STATUS,
      account: { ...Session.Account.ZERO_SLICE_STATE, pending: "minted" },
    });
    fireEvent.click(findButton("Log in"));
    await waitFor(() => expect(open).toHaveBeenCalled());
    const url = new URL(String(open.mock.calls[0][0]));
    expect(url.searchParams.get("state")).toBe("minted");
  });

  it("should let the user retry a log in whose fingerprint read failed", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    let statuses: Status.NotificationSpec[] = [];
    const { wrapper } = await createConsoleWrapper({ client: null });
    const ui = (client: Client): ReactElement => (
      <Synnax.TestProvider client={client} status={UNLICENSED_STATUS}>
        <CaptureStatuses onStatuses={(next) => (statuses = next)} />
        <Account.Guard>
          <span>licensed content</span>
        </Account.Guard>
      </Synnax.TestProvider>
    );
    // A browser reports the closed port as unreachable, which the test client retries
    // for 8 s, so this client gives up at once.
    const internal: Error[] = [];
    const unreachable = createTestClient({
      port: 1,
      retry: { maxRetries: 0 },
      onInternalError: (e) => internal.push(e),
    });
    const { rerender } = render(ui(unreachable), { wrapper });
    fireEvent.click(findButton("Log in"));
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to open login page"),
    );
    expect(open).not.toHaveBeenCalled();
    rerender(ui(createTestClient()));
    fireEvent.click(findButton("Log in"));
    await waitFor(() => expect(open).toHaveBeenCalledOnce());
    expect(internal).not.toHaveLength(0);
    expect(internal.every((e) => isConnectionError(e.cause))).toBe(true);
  });

  it("should ask to try again while the machine is offline", async () => {
    vi.spyOn(navigator, "onLine", "get").mockReturnValue(false);
    await renderGuard({ status: UNLICENSED_STATUS });
    expect(screen.getByText("You are offline")).toBeTruthy();
    expect(findButton("Try again")).toBeTruthy();
    expect(screen.queryByText("Log in")).toBeNull();
  });

  it("should offer the license file screen and a way back", async () => {
    await renderGuard({ status: UNLICENSED_STATUS });
    fireEvent.click(findButton("Use a license file"));
    expect(screen.getByPlaceholderText("Paste a license key")).toBeTruthy();
    expect(screen.queryByText("Log out")).toBeNull();
    fireEvent.click(findButton("Back"));
    expect(screen.getByText("Log in to continue")).toBeTruthy();
  });
});
