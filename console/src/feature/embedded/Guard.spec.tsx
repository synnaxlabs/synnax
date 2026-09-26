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
  createDesktopWrapper,
  mockSupervisor,
} from "@/feature/embedded/testutil";
import { Session } from "@/session";

const { host, port, username, password } = TEST_CLIENT_PARAMS;

// The Core that the specs run against stands in for the embedded one.
const LIVE: Embedded.Status = {
  state: "running",
  connection: { host, port: Number(port), username, password, version: "0.0.0" },
};

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
    await act(async () => await emitStatus(LIVE));
    expect(await screen.findByText("workspace")).toBeTruthy();
  });

  it("should select the embedded Core and keep its password out of the store", async () => {
    mockSupervisor(() => LIVE);
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
    const { emitStatus } = mockSupervisor(() => LIVE);
    await renderGuard();
    expect(await screen.findByText("workspace")).toBeTruthy();
    await act(async () => await emitStatus({ state: "restarting" }));
    expect(screen.getByText("workspace")).toBeTruthy();
  });

  it("should replace the workspace when the Core stays down", async () => {
    const { commands, emitStatus } = mockSupervisor(() => LIVE);
    await renderGuard();
    expect(await screen.findByText("workspace")).toBeTruthy();
    await act(
      async () => await emitStatus({ state: "failed", message: "exited with 3" }),
    );
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
