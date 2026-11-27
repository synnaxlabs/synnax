// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Device } from "@/platform/device";
import { createRunningRack } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

describe("NI.Device.TYPED_HARDWARE find view", () => {
  it("should list typed devices and configure the one the user picks", async () => {
    const rack = await createRunningRack(client, NI.Device.INTEGRATION_KEY);
    const dev = await client.ni.devices.create({
      rack: rack.key,
      name: uniqueName("ni_dev"),
      resourceName: id.create(),
      model: "PXIe-4300",
    });
    const configured = await client.ni.devices.create({
      rack: rack.key,
      name: uniqueName("ni_dev"),
      resourceName: id.create(),
      model: "PXIe-4300",
      identifier: "dev9",
    });
    const { result } = await renderModalOpener(
      Device.useFindModal,
      [{ hardware: NI.Device.TYPED_HARDWARE }],
      { client },
    );
    await screen.findByText(dev.name);
    expect(await screen.findByText(configured.name)).toBeTruthy();
    fireEvent.click(findButton("Configure"));
    fireEvent.change(await screen.findByPlaceholderText("dev1"), {
      target: { value: "dev2" },
    });
    fireEvent.click(findButton("Save"));
    expect(await result()).toBe(dev.key);
    const stored = await client.ni.devices.retrieve({ key: dev.key });
    expect(stored.identifier).toBe("dev2");
  });
});
