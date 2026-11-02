// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { renderWithDeviceForm } from "@/platform/device/testutil";

const client = createTestClient();

const CONNECTION = { host: "10.0.0.7", port: 502, swapBytes: false, swapWords: false };

const DeviceHost = () => {
  const connection = Modbus.Device.useFromConfig()?.properties.connection;
  return (
    <span>{`host:${connection?.transport === "tcp" ? connection.host : ""}`}</span>
  );
};

describe("useFromConfig", () => {
  it("should return the device the form's config names", async () => {
    const dev = await createModbusDevice(client, {
      properties: { connection: CONNECTION },
    });
    await renderWithDeviceForm(<DeviceHost />, { deviceKey: dev.key, client });
    await screen.findByText(`host:${CONNECTION.host}`);
  });
});
