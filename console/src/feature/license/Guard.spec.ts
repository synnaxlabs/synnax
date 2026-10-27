// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  AuthError,
  connection,
  ExpiredLicenseError,
  license,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, onTestFinished, vi } from "vitest";

import {
  findPortalLink,
  GUARDED_CONTENT,
  HANDOFF_CODE,
  renderGuard,
} from "@/feature/license/testutil";
import { License } from "@/platform/license";
import { stubHub } from "@/platform/license/testutil";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  fakePickedFile,
  interceptFilePicker,
  uniqueName,
  UNLICENSED_STATUS,
} from "@/testutil";

const AUTH_FAILED_STATUS: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "error",
  message: "Invalid credentials",
  details: {
    ...connection.DEFAULT_STATUS.details,
    reason: "auth",
    error: new AuthError("Invalid credentials"),
  },
};

const EXPIRED_STATUS: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "error",
  message: license.STATE_MESSAGES.expired,
  details: {
    ...connection.DEFAULT_STATUS.details,
    authenticated: true,
    reason: "unlicensed",
    error: new ExpiredLicenseError(license.STATE_MESSAGES.expired),
  },
};

const MISSING_TITLE = "This Core needs a license";

const getKeyField = (): HTMLElement =>
  screen.getByRole("textbox", { name: "License key" });

/** Waits for the button that opens the portal, and returns the link it opens. */
describe("License.Guard", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should render children while the connection is not refused for a license", async () => {
    await renderGuard(null);
    expect(screen.getByText(GUARDED_CONTENT)).toBeTruthy();
  });

  it("should render children while the connection fails for another reason", async () => {
    await renderGuard(null, AUTH_FAILED_STATUS);
    expect(screen.getByText(GUARDED_CONTENT)).toBeTruthy();
  });

  it("should render the activation screen while the Core is unlicensed", async () => {
    await renderGuard(null, UNLICENSED_STATUS);
    expect(screen.getByText(MISSING_TITLE)).toBeTruthy();
    expect(screen.queryByText(GUARDED_CONTENT)).toBeNull();
  });

  it("should say when the license on the Core has expired", async () => {
    await renderGuard(null, EXPIRED_STATUS);
    expect(screen.getByText("This Core's license has expired")).toBeTruthy();
    expect(
      screen.getByText(
        "Get a new license key on the Synnax portal. It comes back here on its own.",
      ),
    ).toBeTruthy();
  });

  it("should offer no way to check the connection again", async () => {
    await renderGuard(null, UNLICENSED_STATUS);
    expect(screen.queryByText("Check again")).toBeNull();
  });

  it("should render children once a license applies", async () => {
    const { setStatus } = await renderGuard(null, UNLICENSED_STATUS);
    expect(screen.queryByText(GUARDED_CONTENT)).toBeNull();
    setStatus(connection.DEFAULT_STATUS);
    expect(screen.getByText(GUARDED_CONTENT)).toBeTruthy();
  });

  it("should clear the selected Core on log out", async () => {
    const { store } = await renderGuard(null, UNLICENSED_STATUS);
    act(() => {
      store.dispatch(Session.Core.select("LOCAL"));
    });
    expect(Session.Core.selectSelectedKey(store.getState())).toBe("LOCAL");
    fireEvent.click(findButton("Log out"));
    expect(Session.Core.selectSelectedKey(store.getState())).toBeUndefined();
  });

  it("should enable activation only once a license key is entered", async () => {
    await renderGuard(null, UNLICENSED_STATUS);
    const activate = findButton("Activate");
    expect(activate.getAttribute("aria-disabled")).toBe("true");
    fireEvent.change(getKeyField(), { target: { value: "   " } });
    expect(activate.getAttribute("aria-disabled")).toBe("true");
    fireEvent.change(getKeyField(), { target: { value: "  key  " } });
    expect(activate.getAttribute("aria-disabled")).toBeNull();
  });

  it("should read the license key from a picked file", async () => {
    const picker = interceptFilePicker();
    await renderGuard(null, UNLICENSED_STATUS);
    fireEvent.click(screen.getByRole("button", { name: "Load a license file" }));
    picker.selectFiles([fakePickedFile("synnax.lic", "abc.def.ghi\n")]);
    await waitFor(() => {
      const input = getKeyField();
      if (!(input instanceof HTMLInputElement)) throw new Error("not an input");
      expect(input.value).toBe("abc.def.ghi");
    });
  });

  it("should open the portal with the fingerprint and a handoff code", async () => {
    const client = createTestClient();
    const { fingerprint } = await client.license.retrieve();
    expect(fingerprint.length).toBeGreaterThan(0);
    await renderGuard(client, UNLICENSED_STATUS);
    const url = new URL(await findPortalLink("Activate on the portal"));
    expect(url.searchParams.get("fingerprint")).toBe(fingerprint.join(","));
    expect(url.searchParams.get("handoff")).toMatch(HANDOFF_CODE);
  });

  it("should copy the link that the portal opens with", async () => {
    const writeText = stubClipboardWriteText();
    await renderGuard(createTestClient(), UNLICENSED_STATUS);
    const href = await findPortalLink("Activate on the portal");
    fireEvent.click(findButton("Copy link"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(href));
    expect(
      await screen.findByText("Waiting for the portal. This updates on its own."),
    ).toBeTruthy();
  });

  it("should activate the license key that the portal hands back", async () => {
    const hub = stubHub(async () => Response.json({ key: "not-a-license-key" }));
    await renderGuard(createTestClient(), UNLICENSED_STATUS);
    const code = new URL(
      await findPortalLink("Activate on the portal"),
    ).searchParams.get("handoff");
    fireEvent.click(screen.getByText("Activate on the portal"));
    expect(await screen.findByText("Failed to activate license")).toBeTruthy();
    expect(hub).toHaveBeenCalledWith(
      License.HANDOFF_URL,
      expect.objectContaining({ headers: { authorization: `Bearer ${code}` } }),
    );
  });

  it("should wait for a portal it cannot reach", async () => {
    const hub = stubHub(async () => {
      throw new TypeError("Failed to fetch");
    });
    await renderGuard(createTestClient(), UNLICENSED_STATUS);
    await findPortalLink("Activate on the portal");
    fireEvent.click(screen.getByText("Activate on the portal"));
    await waitFor(() => expect(hub).toHaveBeenCalledTimes(2));
    expect(
      screen.getByText("Waiting for the portal. This updates on its own."),
    ).toBeTruthy();
  });

  it("should ask the portal less often after ten minutes", async () => {
    const hub = stubHub(async () => new Response(null, { status: 204 }));
    await renderGuard(createTestClient(), UNLICENSED_STATUS);
    await findPortalLink("Activate on the portal");
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    fireEvent.click(screen.getByText("Activate on the portal"));
    const wait = async (span: TimeSpan): Promise<void> =>
      await act(async () => {
        await vi.advanceTimersByTimeAsync(span.milliseconds);
      });
    await wait(TimeSpan.minutes(10));
    expect(hub).toHaveBeenCalledTimes(300);
    hub.mockClear();
    await wait(TimeSpan.minutes(10));
    expect(hub).toHaveBeenCalledTimes(10);
  });

  it("should report a license key the Core rejects", async () => {
    await renderGuard(createTestClient(), UNLICENSED_STATUS);
    fireEvent.change(getKeyField(), {
      target: { value: uniqueName("not-a-license-key") },
    });
    fireEvent.click(findButton("Activate"));
    expect(await screen.findByText("Failed to activate license")).toBeTruthy();
    expect(
      screen.getByText("a license key has three parts: invalid license: license error"),
    ).toBeTruthy();
    expect(screen.getByText(MISSING_TITLE)).toBeTruthy();
  });
});
