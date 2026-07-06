// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

describe("Modbus.Device.useConnectModal", () => {
  it("should populate the form from an existing device", async () => {
    const dev = await createModbusDevice(client, {
      properties: {
        connection: {
          host: "modbus-existing.local",
          port: 1502,
          swapBytes: true,
          swapWords: false,
        },
      },
    });
    await renderModalOpener(Modbus.Device.useConnectModal, [{ deviceKey: dev.key }], {
      client,
    });
    await screen.findByDisplayValue(dev.name);
    expect(screen.getByDisplayValue("modbus-existing.local")).toBeTruthy();
    expect(screen.getByDisplayValue("1502")).toBeTruthy();
  });
});
