// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { findCommand } from "@/platform/command/testutil";
import { assertDefined, renderHookWithConsole } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("OPCUA.Device Commands", () => {
  it("should expose a connect command visible to device creators", async () => {
    const [connect] = OPCUA.Device.COMMANDS;
    assertDefined(connect.useVisible);
    const { result } = await renderHookWithConsole(connect.useVisible, { client });
    await waitFor(() => expect(result.current).toBe(true));
  });
});

describe("OPCUA.Device Commands permissions", () => {
  it("should offer Connect an OPC UA server to an engineer", async () => {
    const gate = findCommand(
      OPCUA.Device.COMMANDS,
      "Connect OPC UA server",
    ).useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Connect an OPC UA server from a viewer", async () => {
    const gate = findCommand(
      OPCUA.Device.COMMANDS,
      "Connect OPC UA server",
    ).useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(device.TYPE_ONTOLOGY_ID),
      }),
      { client: await roles.get("Viewer") },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
