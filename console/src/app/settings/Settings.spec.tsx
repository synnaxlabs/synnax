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
import { uuid } from "@synnaxlabs/x";
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
import { renderPalette } from "@/feature/command/testutil";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { awaitGranted, createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

// Can read users and nothing else, so it can neither open tabs nor read the license.
// Every grant check reads the role's policies, which awaitGranted caches, so the checks
// in each test answer on the first render.
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
    for (const name of ["Appearance", "Storage", "General", "Labels", "License"]) {
      fireEvent.click(await screen.findByRole("menuitem", { name }));
      expect(await screen.findByRole("heading", { name })).toBeTruthy();
    }
  });

  it("should hide License and Labels from a user who cannot use them", async () => {
    await renderSettings(await createLimitedClient());
    expect(await screen.findByRole("menuitem", { name: "Appearance" })).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "License" })).toBeNull();
    expect(screen.queryByRole("menuitem", { name: "Labels" })).toBeNull();
  });

  it("should show the Labels page when Edit labels runs", async () => {
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "view", key: uuid.create(), type: "other" }],
    });
    const { wrapper, store } = await createPanelWrapper({
      client,
      panelKey: existing.key,
    });
    await primePanel(wrapper, existing.key);
    const { Content } = Settings.TABS.settings;
    render(
      <Suspense fallback={null}>
        <Content />
      </Suspense>,
      { wrapper },
    );
    expect(await screen.findByRole("heading", { name: "Appearance" })).toBeTruthy();
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Settings.COMMANDS,
      client,
      store,
    });
    await openCommandPalette();
    await selectCommand("Edit labels");
    expect(await screen.findByRole("heading", { name: "Labels" })).toBeTruthy();
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
