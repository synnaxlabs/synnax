// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { fireEvent, screen } from "@testing-library/react";
import { describe, it } from "vitest";

import { Device } from "@/feature/device";
import { createTestDevice } from "@/platform/device/testutil";
import { renderToolbar } from "@/platform/ontology/menuTestutil";
import { findTreeRow } from "@/platform/ontology/treeTestutil";

const client = createTestClient();

describe("device/Toolbar", () => {
  it("lists a created device under its rack in the devices tree", async () => {
    const dev = await createTestDevice(client);
    const rack = await client.racks.retrieve({ key: dev.rack });
    await renderToolbar(Device.TOOLBAR.content, {
      client,
      services: { device: Device.ONTOLOGY_SERVICE },
    });
    await screen.findByText("Devices");
    fireEvent.click(await findTreeRow(rack.name));
    await screen.findByText(dev.name);
  });
});
