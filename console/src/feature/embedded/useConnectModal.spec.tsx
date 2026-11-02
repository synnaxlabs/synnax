// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AuthError } from "@synnaxlabs/client";
import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { type Status as LyraStatus } from "@synnaxlabs/lyra/status";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { Command } from "@/feature/command";
import { Embedded } from "@/feature/embedded";
import {
  clearSupervisor,
  createDesktopWrapper,
  createDiscoveryFile,
  createScriptClient,
  deleteKeysCreatedByTest,
  type DiscoveryFile,
  LIVE,
  mockSupervisor,
} from "@/feature/embedded/testutil";
import { CaptureStatuses, getIconButton } from "@/testutil";

const { host, port } = TEST_CLIENT_PARAMS;

const client = createTestClient();

/** Opens the dialog from the palette and waits until it shows the saved key. */
const openConnect = async (
  apiKey: string | null = null,
): Promise<{ file: DiscoveryFile; statuses: () => LyraStatus.NotificationSpec[] }> => {
  const { file, handlers } = createDiscoveryFile(apiKey);
  mockSupervisor(() => LIVE, handlers);
  const { wrapper } = await createDesktopWrapper();
  let statuses: LyraStatus.NotificationSpec[] = [];
  render(
    <>
      <Command.List
        commands={Embedded.COMMANDS}
        value=">"
        onChange={() => {}}
        inputPlaceholder={<>Commands</>}
      />
      <CaptureStatuses onStatuses={(next) => (statuses = next)} />
    </>,
    { wrapper },
  );
  const command = await screen.findByText("Connect from code");
  await act(async () => {
    fireEvent.click(command, { detail: 1 });
  });
  await screen.findByLabelText("Python code");
  return { file, statuses: () => statuses };
};

const savedKey = (file: DiscoveryFile): string => {
  if (file.apiKey == null) throw new Error("no API key saved");
  return file.apiKey;
};

describe("Embedded connect from code dialog", () => {
  afterEach(clearSupervisor);

  it("should save a key on open and keep it out of the Python code", async () => {
    await deleteKeysCreatedByTest(client);
    const { file } = await openConnect();
    const apiKey = savedKey(file);
    await createScriptClient(apiKey).auth.retrieveSessions();
    expect(screen.getByLabelText("Python code").textContent).not.toContain(apiKey);
  });

  it("should open from the top bar button", async () => {
    await deleteKeysCreatedByTest(client);
    const { handlers } = createDiscoveryFile();
    mockSupervisor(() => LIVE, handlers);
    const { wrapper } = await createDesktopWrapper();
    const { container } = render(<Embedded.ConnectButton />, { wrapper });
    fireEvent.click(getIconButton(container, "code"));
    expect(await screen.findByLabelText("Python code")).toBeTruthy();
  });

  it("should fill the TypeScript code with the address and mask the key", async () => {
    await deleteKeysCreatedByTest(client);
    const { file } = await openConnect();
    fireEvent.click(screen.getByRole("tab", { name: "TypeScript" }));
    const code = (await screen.findByLabelText("TypeScript code")).textContent;
    expect(code).toContain(`host: "${host}"`);
    expect(code).toContain(`port: ${port}`);
    expect(code).toContain('apiKey: "syk_');
    expect(code).not.toContain(savedKey(file));
  });

  it("should show the API key only on request", async () => {
    await deleteKeysCreatedByTest(client);
    const { file } = await openConnect();
    fireEvent.click(screen.getByRole("tab", { name: "Other" }));
    const value = await screen.findByLabelText("API key");
    expect(value.textContent).not.toBe(savedKey(file));
    fireEvent.click(screen.getByRole("button", { name: "Show API key" }));
    expect(value.textContent).toBe(savedKey(file));
  });

  it("should count the sessions of the key", async () => {
    await deleteKeysCreatedByTest(client);
    const { token } = await client.auth.createKey({ name: "counted" });
    await createScriptClient(token).auth.retrieveSessions();
    await openConnect(token);
    expect(await screen.findByText("1 session")).toBeTruthy();
  });

  it("should replace the key after asking and end the sessions of the old one", async () => {
    await deleteKeysCreatedByTest(client);
    const { token } = await client.auth.createKey({ name: "replaced" });
    const script = createScriptClient(token);
    await script.auth.retrieveSessions();
    const { file, statuses } = await openConnect(token);
    await screen.findByText("1 session");
    fireEvent.click(screen.getByRole("button", { name: "Reset key" }));
    await screen.findByText(/want to reset the API key/);
    const buttons = screen.getAllByRole("button", { name: "Reset key" });
    fireEvent.click(buttons[buttons.length - 1]);
    await waitFor(() => expect(file.apiKey).not.toBe(token));
    await waitFor(() =>
      expect(statuses().map((s) => s.message)).toContain("Reset API key"),
    );
    expect(await screen.findByText("0 sessions")).toBeTruthy();
    await expect(script.auth.retrieveSessions()).rejects.toThrow(AuthError);
    await createScriptClient(savedKey(file)).auth.retrieveSessions();
  });
});
