// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { findCommand } from "@/platform/command/testutil";
import { assertDefined, renderHookWithConsole } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const NAMES = ["Create channel", "Create calculated channel"];

describe("Channel Commands", () => {
  it.each(NAMES)("should offer %s to an engineer", async (name) => {
    const gate = findCommand(Channel.COMMANDS, name).useVisible;
    assertDefined(gate);
    const { result } = await renderHookWithConsole(gate, {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it.each(NAMES)("should withhold %s from a viewer", async (name) => {
    const gate = findCommand(Channel.COMMANDS, name).useVisible;
    assertDefined(gate);
    const viewer = await roles.get("Viewer");
    const { result } = await renderHookWithConsole(
      () => ({
        visible: gate(),
        loaded: Access.useRetrieveGranted(channel.TYPE_ONTOLOGY_ID),
      }),
      { client: viewer },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
