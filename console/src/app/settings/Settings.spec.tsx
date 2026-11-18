// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, label, user } from "@synnaxlabs/client";
import {
  CERTIFICATE,
  createTestClient,
  createTestClientWithPolicy,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { Suspense } from "react";
import { assert, describe, expect, it } from "vitest";

import { Settings } from "@/app/settings";
import { renderSettings } from "@/app/settings/testutil";
import { renderPalette } from "@/feature/command/testutil";
import { findCommand } from "@/platform/command/testutil";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Session } from "@/session";
import {
  awaitGranted,
  createConsoleWrapper,
  createTestClientWithGrants,
  renderHookWithConsole,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

// Can read users and nothing else, so it can neither open tabs, read the license, nor
// manage users.
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

describe("Settings", () => {
  it("should open the page of each section", async () => {
    await renderSettings();
    const sections = ["Profile", "API keys", "Appearance", "Storage", "About"];
    for (const name of [
      ...sections,
      "Users",
      "Labels",
      "Colors",
      "Secrets",
      "Certificates",
      "License",
    ]) {
      fireEvent.click(await screen.findByRole("menuitem", { name }));
      expect(await screen.findByRole("heading", { name })).toBeTruthy();
    }
  });

  it("should hide the sections that a user cannot use", async () => {
    await renderSettings(await createLimitedClient());
    expect(await screen.findByRole("menuitem", { name: "Appearance" })).toBeTruthy();
    for (const name of [
      "License",
      "Users",
      "Labels",
      "Colors",
      "Secrets",
      "Certificates",
    ])
      expect(screen.queryByRole("menuitem", { name })).toBeNull();
  });

  it("should show Certificates and not Secrets to a user who can read only certificates", async () => {
    const reader = await createTestClientWithGrants(client, {
      retrieve: [certificate.TYPE_RESOURCE_ID],
    });
    await awaitGranted(reader, certificate.TYPE_RESOURCE_ID, "retrieve");
    await renderSettings(reader);
    expect(await screen.findByRole("menuitem", { name: "Certificates" })).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "Secrets" })).toBeNull();
  });

  it("should show the Certificates page for a certificate that uses a secret", async () => {
    const c = await client.certificates.set({
      name: uniqueName("certificate"),
      source: "stored",
      ...CERTIFICATE,
    });
    assert(c.source === "stored" && c.privateKey != null);
    const privateKey = await client.secrets.retrieve(c.privateKey);
    try {
      const { wrapper } = await renderSettings();
      const pick = renderHook(Session.Nav.useSelectSettingsItem, { wrapper });
      fireEvent.click(await screen.findByRole("menuitem", { name: "Secrets" }));
      const item = await screen.findByRole("option", { name: privateKey.name });
      const count = await within(item).findByText("Used by 1");
      await act(async () => {
        fireEvent.click(count);
      });
      const dialog = await screen.findByRole("dialog");
      fireEvent.click(await within(dialog).findByText(c.name));
      expect(await screen.findByRole("heading", { name: "Certificates" })).toBeTruthy();
      const picked = await screen.findByRole("option", { name: c.name });
      await waitFor(() => expect(picked.getAttribute("aria-selected")).toBe("true"));
      // The page consumes the pick, so a later visit selects nothing.
      await waitFor(() => expect(pick.result.current).toBeUndefined());
    } finally {
      await client.certificates.delete(c.key);
      await client.secrets.delete(c.privateKey);
    }
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
    expect(await screen.findByRole("heading", { name: "Profile" })).toBeTruthy();
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Settings.COMMANDS,
      client,
      store,
    });
    await openCommandPalette();
    await selectCommand("Edit labels");
    expect(await screen.findByRole("heading", { name: "Labels" })).toBeTruthy();
  });

  it("should show the License page when Show license info runs", async () => {
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
    expect(await screen.findByRole("heading", { name: "Profile" })).toBeTruthy();
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Settings.COMMANDS,
      client,
      store,
    });
    await openCommandPalette();
    await selectCommand("Show license info");
    expect(await screen.findByRole("heading", { name: "License" })).toBeTruthy();
  });

  it("should withhold Show license info without a license grant", async () => {
    const gate = findCommand(Settings.COMMANDS, "Show license info").useVisible;
    assertDefined(gate);
    // The user grant resolves through the same query, so the gate has answered too.
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(user.TYPE_RESOURCE_ID),
      }),
      { client: await createLimitedClient() },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });

  it("should offer the gear and Open settings to a Viewer", async () => {
    const viewer = await roles.get("Viewer");
    const { wrapper } = await createConsoleWrapper({ client: viewer });
    render(<Settings.Button />, { wrapper });
    expect(await screen.findByRole("button", { name: "Settings" })).toBeTruthy();
    const { openCommandPalette } = await renderPalette({
      commands: Settings.COMMANDS,
      client: viewer,
    });
    await openCommandPalette();
    expect(await screen.findByText("Open settings")).toBeTruthy();
  });

  it("should offer Edit labels to an Engineer", async () => {
    const visible = findCommand(Settings.COMMANDS, "Edit labels").useVisible;
    assertDefined(visible);
    const { result } = await renderHookWithConsole(visible, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Edit labels from a Viewer", async () => {
    const visible = findCommand(Settings.COMMANDS, "Edit labels").useVisible;
    assertDefined(visible);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: visible(),
        loaded: Access.useRetrieveGranted(label.TYPE_RESOURCE_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
