// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { deep, id } from "@synnaxlabs/x";

import { Device } from "@/feature/modbus/device";
import { uniqueName } from "@/testutil";

export interface CreateModbusDeviceOptions {
  configured?: boolean;
  properties?: Partial<Device.Properties>;
}

/** Creates a rack and a configured Modbus server device on the live Core. */
export const createModbusDevice = async (
  client: Synnax,
  { configured = true, properties }: CreateModbusDeviceOptions = {},
): Promise<Device.Device> => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await client.devices.create(
    {
      key: id.create(),
      name: uniqueName("modbus_server"),
      rack: rack.key,
      location: "localhost:502",
      make: "Modbus",
      model: "Modbus",
      configured,
      properties: { ...deep.copy(Device.ZERO_PROPERTIES), ...properties },
    },
    Device.SCHEMAS,
  );
};
