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
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { MQTT } from "@/feature/mqtt";
import { findCommand } from "@/platform/command/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

import { assertDefined, renderHookWithConsole } from "@/testutil";

describe("MQTT.Device Commands", () => {
  it("should open the connect modal from the connect broker command", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: MQTT.Device.COMMANDS,
      client,
    });
    await openCommandPalette("Connect an MQTT");
    await selectCommand("Connect an MQTT broker");
    expect(await screen.findByPlaceholderText("broker.example.com")).toBeTruthy();
  });
});

describe("MQTT.Device Commands permissions", () => {
  it("should offer Connect an MQTT broker to an engineer", async () => {
    const gate = findCommand(MQTT.Device.COMMANDS, "Connect an MQTT broker").useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold Connect an MQTT broker from a viewer", async () => {
    const gate = findCommand(MQTT.Device.COMMANDS, "Connect an MQTT broker").useVisible;
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
