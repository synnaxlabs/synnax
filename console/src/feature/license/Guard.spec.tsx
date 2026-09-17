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
  MissingLicenseError,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { License } from "@/feature/license";
import { findButton } from "@/platform/modals/testutil";
import {
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  uniqueName,
} from "@/testutil";

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

// The client is handed to the provider unconnected: a Core that refuses requests for
// want of a license never settles a connection, so the screen must not wait on one.
const renderGuard = async (
  client: Client | null,
  status?: connection.Status,
): Promise<void> => {
  const { wrapper: Console } = await createConsoleWrapper({ client: null });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Synnax.TestProvider client={client} status={status}>
        {children}
      </Synnax.TestProvider>
    </Console>
  );
  Wrapper.displayName = "GuardWrapper";
  render(
    <License.Guard>
      <span>licensed content</span>
    </License.Guard>,
    { wrapper: Wrapper },
  );
};

const HASH = /^[0-9a-f]{64}$/;

describe("License.Guard", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should render children while the connection is not refused for a license", async () => {
    await renderGuard(null);
    expect(screen.getByText("licensed content")).toBeTruthy();
  });

  it("should render the activation screen while the Core is unlicensed", async () => {
    await renderGuard(null, UNLICENSED);
    expect(screen.getByText(MESSAGE)).toBeTruthy();
    expect(screen.queryByText("licensed content")).toBeNull();
  });

  it("should enable activation only once a token is entered", async () => {
    await renderGuard(null, UNLICENSED);
    const activate = findButton("Activate");
    expect(activate.getAttribute("aria-disabled")).toBe("true");
    fireEvent.change(screen.getByPlaceholderText("Paste the token"), {
      target: { value: "  token  " },
    });
    expect(activate.getAttribute("aria-disabled")).toBeNull();
  });

  it("should read the token from a picked file", async () => {
    const picker = interceptFilePicker();
    await renderGuard(null, UNLICENSED);
    fireEvent.click(findButton("Select file"));
    picker.selectFiles([fakePickedFile("synnax.license", "abc.def.ghi\n")]);
    await waitFor(() => {
      const input = screen.getByPlaceholderText("Paste the token");
      if (!(input instanceof HTMLTextAreaElement)) throw new Error("not a textarea");
      expect(input.value).toBe("abc.def.ghi");
    });
  });

  it("should show the host fingerprint the Core reports", async () => {
    await renderGuard(createTestClient(), UNLICENSED);
    const hashes = await screen.findAllByText(HASH);
    expect(hashes.length).toBeGreaterThan(0);
  });

  it("should report a token the Core rejects", async () => {
    await renderGuard(createTestClient(), UNLICENSED);
    fireEvent.change(screen.getByPlaceholderText("Paste the token"), {
      target: { value: uniqueName("not-a-token") },
    });
    fireEvent.click(findButton("Activate"));
    expect(await screen.findByText("Failed to activate the license")).toBeTruthy();
    expect(screen.getByText(MESSAGE)).toBeTruthy();
  });
});
