// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { Embedded } from "@/feature/embedded";
import {
  clearSupervisor,
  createDesktopUser,
  createDesktopWrapper,
  createDiscoveryFile,
  createScriptClient,
  deleteKeysCreatedByTest,
  mockSupervisor,
  retrieveKeyIDs,
} from "@/feature/embedded/testutil";
import { Session } from "@/session";

const { port } = TEST_CLIENT_PARAMS;

const { client, live } = await createDesktopUser();

const renderGuard = async (): ReturnType<typeof createDesktopWrapper> => {
  const result = await createDesktopWrapper();
  render(
    <Embedded.Guard>
      <span>workspace</span>
    </Embedded.Guard>,
    { wrapper: result.wrapper },
  );
  return result;
};

describe("Embedded.Guard", () => {
  afterEach(clearSupervisor);

  it("should hold the workspace back until the Core runs", async () => {
    const { emitStatus } = mockSupervisor(() => ({ state: "starting" }));
    await renderGuard();
    expect(await screen.findByText("Starting Synnax...")).toBeTruthy();
    expect(screen.queryByText("workspace")).toBeNull();
    await emitStatus(live);
    expect(await screen.findByText("workspace")).toBeTruthy();
  });

  it("should select the embedded Core and keep its password out of the store", async () => {
    mockSupervisor(() => live);
    const { store } = await renderGuard();
    await waitFor(() =>
      expect(Session.Core.selectSelectedKey(store.getState())).toBe(
        Session.Core.EMBEDDED_KEY,
      ),
    );
    const core = Session.Core.selectSelected(store.getState());
    expect(core?.port).toBe(Number(port));
    expect(core?.password).toBe("");
    expect(await screen.findByText("workspace")).toBeTruthy();
  });

  it("should leave the workspace mounted while the Core restarts", async () => {
    const { emitStatus } = mockSupervisor(() => live);
    await renderGuard();
    expect(await screen.findByText("workspace")).toBeTruthy();
    await emitStatus({ state: "restarting" });
    expect(screen.getByText("workspace")).toBeTruthy();
  });

  it("should replace the workspace when the Core stays down", async () => {
    const { commands, emitStatus } = mockSupervisor(() => live);
    await renderGuard();
    expect(await screen.findByText("workspace")).toBeTruthy();
    await emitStatus({ state: "failed", reason: "crashed", message: "exited with 3" });
    expect(await screen.findByText("Synnax stopped unexpectedly")).toBeTruthy();
    expect(screen.queryByText("workspace")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_restart"));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Diagnostics" }));
    });
    expect(await screen.findByText("Stopped unexpectedly")).toBeTruthy();
  });
});

describe("Embedded.Guard stops and waits", () => {
  afterEach(clearSupervisor);

  it("should replace the workspace while the Core stops", async () => {
    const { emitStatus } = mockSupervisor(() => live);
    await renderGuard();
    expect(await screen.findByText("workspace")).toBeTruthy();
    await emitStatus({ state: "stopping" });
    expect(await screen.findByText("Stopping Synnax...")).toBeTruthy();
    expect(screen.queryByText("workspace")).toBeNull();
  });

  it("should wait for the Core of the last session to free the data", async () => {
    const { emitStatus } = mockSupervisor(() => ({ state: "starting" }));
    await renderGuard();
    await emitStatus({ state: "waiting" });
    expect(await screen.findByText("Waiting for Synnax to stop...")).toBeTruthy();
    await emitStatus(live);
    expect(await screen.findByText("workspace")).toBeTruthy();
  });

  it("should name the data folder when it stays in use", async () => {
    const { commands, emitStatus } = mockSupervisor(() => ({ state: "waiting" }));
    await renderGuard();
    await emitStatus({
      state: "failed",
      reason: "locked",
      message: "the data directory stayed locked for 60s",
    });
    expect(await screen.findByText("Data folder in use")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_restart"));
  });
});

describe("Embedded.Guard API key for scripts", () => {
  afterEach(clearSupervisor);

  it("should save an API key that logs in as the user of the workspace", async () => {
    await deleteKeysCreatedByTest(client);
    const { file, handlers } = createDiscoveryFile();
    mockSupervisor(() => live, handlers);
    await renderGuard();
    const apiKey = await waitFor(() => {
      if (file.apiKey == null) throw new Error("no API key saved");
      return file.apiKey;
    });
    const script = createScriptClient(apiKey);
    await script.auth.retrieveSessions();
    expect(script.auth.subject).toEqual(client.auth.subject);
  });

  it("should keep a saved key that the Core accepts", async () => {
    await deleteKeysCreatedByTest(client);
    const { token } = await client.auth.createKey({ name: "saved" });
    const keys = (await client.auth.retrieveKeys()).length;
    const { file, handlers } = createDiscoveryFile(token);
    const { commands } = mockSupervisor(() => live, handlers);
    await renderGuard();
    await waitFor(() =>
      expect(commands).toHaveBeenCalledWith("supervisor_save_api_key"),
    );
    expect(file.apiKey).toBe(token);
    expect(await client.auth.retrieveKeys()).toHaveLength(keys);
  });

  it("should delete the new key when the save fails", async () => {
    await deleteKeysCreatedByTest(client);
    const before = await retrieveKeyIDs(client);
    const { handlers } = createDiscoveryFile();
    const { commands } = mockSupervisor(() => live, {
      ...handlers,
      supervisor_save_api_key: () => {
        throw new Error("disk full");
      },
    });
    await renderGuard();
    await waitFor(() =>
      expect(commands).toHaveBeenCalledWith("supervisor_save_api_key"),
    );
    await expect.poll(async () => await retrieveKeyIDs(client)).toEqual(before);
  });

  it("should use the key that another window saved first", async () => {
    await deleteKeysCreatedByTest(client);
    const { token: other } = await client.auth.createKey({ name: "other window" });
    const before = await retrieveKeyIDs(client);
    const { file, handlers } = createDiscoveryFile(null, (f) => {
      f.apiKey ??= other;
    });
    mockSupervisor(() => live, handlers);
    await renderGuard();
    await waitFor(() => expect(file.apiKey).toBe(other));
    await expect.poll(async () => await retrieveKeyIDs(client)).toEqual(before);
  });

  it("should replace a saved key that the Core no longer accepts", async () => {
    await deleteKeysCreatedByTest(client);
    const { token, key } = await client.auth.createKey({ name: "deleted" });
    await client.auth.deleteKeys(key.key);
    const { file, handlers } = createDiscoveryFile(token);
    mockSupervisor(() => live, handlers);
    await renderGuard();
    const apiKey = await waitFor(() => {
      if (file.apiKey == null || file.apiKey === token)
        throw new Error("saved key not replaced");
      return file.apiKey;
    });
    await createScriptClient(apiKey).auth.retrieveSessions();
  });

  it("should replace a saved key that is not an API key", async () => {
    await deleteKeysCreatedByTest(client);
    const { file, handlers } = createDiscoveryFile("edited by hand");
    mockSupervisor(() => live, handlers);
    await renderGuard();
    const apiKey = await waitFor(() => {
      if (file.apiKey == null || file.apiKey === "edited by hand")
        throw new Error("saved key not replaced");
      return file.apiKey;
    });
    await createScriptClient(apiKey).auth.retrieveSessions();
  });
});
