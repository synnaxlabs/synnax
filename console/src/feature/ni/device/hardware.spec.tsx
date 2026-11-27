// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ni } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { NI } from "@/feature/ni";
import { Device } from "@/platform/device";
import { renderMenuItem } from "@/platform/device/testutil";
import { createResource, createSelection } from "@/platform/tree/testutil";

const client = createTestClient();

const ConfigureMenuItem = Device.createConfigureMenuItem(NI.Device.TYPED_ADAPTER);
const ChangeIdentifierMenuItem = Device.createChangeIdentifierMenuItem({
  adapter: NI.Device.TYPED_ADAPTER,
  icon: "Logo.NI",
});

const renderItems = async (identifier: string): Promise<void> => {
  const rack = await client.racks.create({ name: uniqueName("ni_rack") });
  const dev = await client.ni.devices.create({
    rack: rack.key,
    name: uniqueName("ni_dev"),
    resourceName: id.create(),
    model: "PXIe-4300",
    identifier,
  });
  const selection = createSelection({
    ids: [createResource(ni.device.resourceID(dev.key), dev.name).id],
  });
  await renderMenuItem(
    <>
      <ConfigureMenuItem onConfigure={vi.fn()} selection={selection} />
      <ChangeIdentifierMenuItem selection={selection} handleError={() => {}} />
    </>,
    { client },
  );
};

describe("TYPED_ADAPTER", () => {
  it("should offer to configure a device without an identifier", async () => {
    await renderItems("");
    await screen.findByText("Configure");
    expect(screen.queryByText("Change identifier")).toBeNull();
  });

  it("should offer to change the identifier of a configured device", async () => {
    await renderItems("dev1");
    await screen.findByText("Change identifier");
    expect(screen.queryByText("Configure")).toBeNull();
  });
});
