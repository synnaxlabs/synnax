// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from "@testing-library/react";
import { Suspense } from "react";
import { describe, expect, it } from "vitest";

import { Settings } from "@/app/settings";
import { awaitGranted, createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

// Can read users and nothing else, so it can neither open tabs nor read the license.
const createLimitedClient = async () => {
  const limited = await createTestClientWithPolicy(client, {
    name: uniqueName("policy"),
    objects: [user.TYPE_RESOURCE_ID],
    actions: ["retrieve"],
  });
  await awaitGranted(limited, user.TYPE_RESOURCE_ID, "retrieve");
  return limited;
};

const renderSettings = async (as = client): Promise<void> => {
  const { Content } = Settings.TABS.settings;
  const { wrapper } = await createConsoleWrapper({ client: as });
  render(
    <Suspense fallback={null}>
      <Content />
    </Suspense>,
    { wrapper },
  );
};

const useOpenVisible = (): boolean | undefined => Settings.COMMANDS[0].useVisible?.();

describe("Settings", () => {
  it("should open the page of each section", async () => {
    await renderSettings();
    for (const name of ["Appearance", "Storage", "General", "License"]) {
      fireEvent.click(await screen.findByRole("menuitem", { name }));
      expect(await screen.findByRole("heading", { name })).toBeTruthy();
    }
  });

  it("should hide License from a user who cannot read it", async () => {
    await renderSettings(await createLimitedClient());
    expect(await screen.findByRole("menuitem", { name: "Appearance" })).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "License" })).toBeNull();
  });

  it("should offer the gear and the command to a user who can open tabs", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    const { result } = renderHook(useOpenVisible, { wrapper });
    render(<Settings.Button />, { wrapper });
    expect(await screen.findByRole("button", { name: "Settings" })).toBeTruthy();
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should hide the gear and the command from a user who cannot open tabs", async () => {
    const { wrapper } = await createConsoleWrapper({
      client: await createLimitedClient(),
    });
    const { result } = renderHook(useOpenVisible, { wrapper });
    render(<Settings.Button />, { wrapper });
    await act(async () => {});
    expect(result.current).toBe(false);
    expect(screen.queryByRole("button", { name: "Settings" })).toBeNull();
  });
});
