// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { InvalidCredentialsError } from "@synnaxlabs/client";
import { TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { Command } from "@/feature/command";
import { Embedded } from "@/feature/embedded";
import {
  clearSupervisor,
  createDesktopUser,
  createDesktopWrapper,
  createDiscoveryFile,
  createScriptClient,
  deleteKeysCreatedByTest,
  type DiscoveryFile,
  mockSupervisor,
  retrieveKeyIDs,
} from "@/feature/embedded/testutil";
import { findDismissButton } from "@/platform/modals/testutil";
import { CaptureStatuses, getIconButton } from "@/testutil";

const { host, port } = TEST_CLIENT_PARAMS;

const { client, live } = await createDesktopUser();

/** Opens the dialog from the palette and waits until it shows the saved key. */
const openConnect = async (
  ...args: Parameters<typeof createDiscoveryFile>
): Promise<{ file: DiscoveryFile; statuses: () => Status.NotificationSpec[] }> => {
  const { file, handlers } = createDiscoveryFile(...args);
  mockSupervisor(() => live, handlers);
  const { wrapper } = await createDesktopWrapper();
  let statuses: Status.NotificationSpec[] = [];
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

/** Clicks "Reset key", then confirms the reset. */
const resetKey = async (): Promise<void> => {
  fireEvent.click(screen.getByRole("button", { name: "Reset key" }));
  await screen.findByText(/want to reset the API key/);
  const buttons = screen.getAllByRole("button", { name: "Reset key" });
  fireEvent.click(buttons[buttons.length - 1]);
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
    mockSupervisor(() => live, handlers);
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

  it("should copy the TypeScript code with the API key", async () => {
    await deleteKeysCreatedByTest(client);
    const writeText = stubClipboardWriteText();
    const { file } = await openConnect();
    fireEvent.click(screen.getByRole("tab", { name: "TypeScript" }));
    fireEvent.click(
      await screen.findByRole("button", { name: "Copy TypeScript code" }),
    );
    await waitFor(() => expect(writeText).toHaveBeenCalled());
    expect(writeText.mock.calls[0][0]).toContain(`apiKey: "${savedKey(file)}"`);
  });

  it("should count the sessions of the key", async () => {
    await deleteKeysCreatedByTest(client);
    const { token } = await client.auth.createKey({ name: "counted" });
    await createScriptClient(token).auth.retrieveSessions();
    await openConnect(token);
    expect(await screen.findByText("1 session")).toBeTruthy();
  });

  it("should count sessions opened since the dialog last closed", async () => {
    await deleteKeysCreatedByTest(client);
    const { token } = await client.auth.createKey({ name: "reopened" });
    await openConnect(token);
    await screen.findByText("0 sessions");
    fireEvent.click(findDismissButton());
    await waitFor(() => expect(screen.queryByLabelText("Python code")).toBeNull());
    await createScriptClient(token).auth.retrieveSessions();
    await act(async () => {
      fireEvent.click(screen.getByText("Connect from code"), { detail: 1 });
    });
    expect(await screen.findByText("1 session")).toBeTruthy();
  });

  it("should show the key of a reset that another window saved first", async () => {
    await deleteKeysCreatedByTest(client);
    const { token, key } = await client.auth.createKey({ name: "replaced" });
    const { token: other } = await client.auth.createKey({ name: "other window" });
    const before = (await retrieveKeyIDs(client)).filter((k) => k !== key.key);
    const writeText = stubClipboardWriteText();
    const { file, statuses } = await openConnect(token, (f, apiKey) => {
      if (apiKey !== token && f.apiKey === token) f.apiKey = other;
    });
    await resetKey();
    await waitFor(() =>
      expect(statuses().map((s) => s.message)).toContain("Reset API key"),
    );
    fireEvent.click(screen.getByRole("tab", { name: "TypeScript" }));
    fireEvent.click(
      await screen.findByRole("button", { name: "Copy TypeScript code" }),
    );
    await waitFor(() => expect(writeText).toHaveBeenCalled());
    expect(writeText.mock.calls[0][0]).toContain(`apiKey: "${other}"`);
    expect(file.apiKey).toBe(other);
    await expect.poll(async () => await retrieveKeyIDs(client)).toEqual(before);
  });

  it("should end the old key before it saves the new one", async () => {
    await deleteKeysCreatedByTest(client);
    const { token, key } = await client.auth.createKey({ name: "replaced" });
    let oldKeyAtSave: boolean | null = null;
    const { file } = await openConnect(token, async (_, apiKey) => {
      if (apiKey === token) return;
      oldKeyAtSave = (await retrieveKeyIDs(client)).includes(key.key);
    });
    await resetKey();
    await waitFor(() => expect(file.apiKey).not.toBe(token));
    expect(oldKeyAtSave).toBe(false);
  });

  it("should replace the key after asking and end the sessions of the old one", async () => {
    await deleteKeysCreatedByTest(client);
    const { token } = await client.auth.createKey({ name: "replaced" });
    const script = createScriptClient(token);
    await script.auth.retrieveSessions();
    const { file, statuses } = await openConnect(token);
    await screen.findByText("1 session");
    await resetKey();
    await waitFor(() => expect(file.apiKey).not.toBe(token));
    await waitFor(() =>
      expect(statuses().map((s) => s.message)).toContain("Reset API key"),
    );
    expect(await screen.findByText("0 sessions")).toBeTruthy();
    await expect(script.auth.retrieveSessions()).rejects.toThrow(
      InvalidCredentialsError,
    );
    await createScriptClient(savedKey(file)).auth.retrieveSessions();
  });
});
