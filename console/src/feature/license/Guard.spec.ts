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
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { GUARDED_CONTENT, renderGuard } from "@/feature/license/testutil";
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
    expect(screen.getByText("Get a new key from your Synnax account.")).toBeTruthy();
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
      store.dispatch(Session.Core.select(Session.Core.LOCAL_KEY));
    });
    expect(Session.Core.selectSelectedKey(store.getState())).toBe(
      Session.Core.LOCAL_KEY,
    );
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

  it("should link to the account page with the host fingerprint", async () => {
    const client = createTestClient();
    const { fingerprint } = await client.license.retrieve();
    expect(fingerprint.length).toBeGreaterThan(0);
    await renderGuard(client, UNLICENSED_STATUS);
    await waitFor(() => {
      const href = screen.getByText("Get a key").closest("a")?.getAttribute("href");
      expect(new URL(href ?? "").searchParams.get("fingerprint")).toBe(
        fingerprint.join(", "),
      );
    });
  });

  it("should copy the host fingerprint", async () => {
    const writeText = stubClipboardWriteText();
    const client = createTestClient();
    const { fingerprint } = await client.license.retrieve();
    expect(fingerprint.length).toBeGreaterThan(0);
    await renderGuard(client, UNLICENSED_STATUS);
    const copy = findButton("Copy fingerprint");
    await waitFor(() => expect(copy.getAttribute("aria-disabled")).toBeNull());
    fireEvent.click(copy);
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText).toHaveBeenCalledWith(fingerprint.join(", "));
  });

  it("should report a license key the Core rejects", async () => {
    await renderGuard(createTestClient(), UNLICENSED_STATUS);
    fireEvent.change(getKeyField(), {
      target: { value: uniqueName("not-a-license-key") },
    });
    fireEvent.click(findButton("Activate"));
    expect(await screen.findByText("Failed to activate the license")).toBeTruthy();
    expect(
      screen.getByText("a license key has three parts: invalid license: license error"),
    ).toBeTruthy();
    expect(screen.getByText(MISSING_TITLE)).toBeTruthy();
  });
});
