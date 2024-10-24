// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device as PlutoDevice, Form, Synnax } from "@synnaxlabs/pluto";

import { Device } from "@/hardware/labjack/device";
import { type Properties } from "@/hardware/labjack/device/types";
import { Layout } from "@/layout";

export const SelectDevice = () => {
  const client = Synnax.use();
  const placer = Layout.usePlacer();
  const handleDeviceChange = async (v: string) => {
    if (client == null) return;
    const { configured } = await client.hardware.devices.retrieve<Properties>(v);
    if (configured) return;
    placer(Device.createConfigureLayout(v, {}));
  };
  return (
    <Form.Field<string>
      path="config.deviceKey"
      label="Device"
      grow
      onChange={handleDeviceChange}
      style={{ width: "100%" }}
    >
      {(p) => (
        <PlutoDevice.SelectSingle
          allowNone={false}
          grow
          {...p}
          autoSelectOnNone={false}
          searchOptions={{ makes: ["LabJack"] }}
        />
      )}
    </Form.Field>
  );
};
