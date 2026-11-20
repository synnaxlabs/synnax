// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, rack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { assertDefined } from "@synnaxlabs/x/testutil";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { findCommand } from "@/platform/command/testutil";
import {
  createTestClientWithGrants,
  type Grants,
  renderHookWithConsole,
} from "@/testutil";

const client = createTestClient();

const renderCreateRackGate = async (grants: Grants) => {
  const gate = findCommand(Device.COMMANDS, "Create rack").useVisible;
  assertDefined(gate);
  return await renderHookWithConsole(
    () => ({
      visible: gate(),
      loaded: Access.useRetrieveGranted(rack.TYPE_RESOURCE_ID),
    }),
    { client: await createTestClientWithGrants(client, grants) },
  );
};

describe("Device Commands", () => {
  it("should offer Create rack to a subject who may create racks but not read devices", async () => {
    const { result } = await renderCreateRackGate({
      retrieve: [rack.TYPE_RESOURCE_ID],
      create: [rack.TYPE_RESOURCE_ID],
    });
    await waitFor(() => expect(result.current.visible).toBe(true));
  });

  it("should withhold Create rack from a subject who cannot create racks", async () => {
    const { result } = await renderCreateRackGate({
      retrieve: [rack.TYPE_RESOURCE_ID, device.TYPE_RESOURCE_ID],
    });
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });
});
