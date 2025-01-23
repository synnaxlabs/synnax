// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Align, Device, Form, Synnax, Text } from "@synnaxlabs/pluto";
import { type JSX } from "react";

import { Layout } from "@/layout";

export interface SelectProps {
  configureLayout: Omit<Layout.BaseState, "key">;
  emptyContent?: string | JSX.Element;
  label?: string;
  make: string;
  path?: string;
}

export const Select = ({
  configureLayout,
  emptyContent = "No devices connected.",
  label = "Device",
  make,
  path = "config.device",
}: SelectProps) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleDeviceChange = async (key: device.Key) => {
    if (client == null) return;
    const { configured } = await client.hardware.devices.retrieve(key);
    if (configured) return;
    placeLayout({ ...configureLayout, key });
  };
  return (
    <Form.Field<string>
      grow
      label={label}
      onChange={handleDeviceChange}
      path={path}
      style={{ width: "100%" }}
    >
      {(p) => (
        <Device.SelectSingle
          {...p}
          allowNone={false}
          autoSelectOnNone={false}
          emptyContent={
            typeof emptyContent === "string" ? (
              <Align.Center>
                <Text.Text shade={6} level="p">
                  {emptyContent}
                </Text.Text>
              </Align.Center>
            ) : (
              emptyContent
            )
          }
          grow
          searchOptions={{ makes: [make] }}
        />
      )}
    </Form.Field>
  );
};
