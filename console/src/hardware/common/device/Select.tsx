// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device, Form, type Icon, Status, Synnax } from "@synnaxlabs/pluto";
import { deep, primitive } from "@synnaxlabs/x";
import { type JSX, useCallback } from "react";

import { ZERO_PROPERTIES as LABJACK_ZERO_PROPERTIES } from "@/hardware/labjack/device/types";
import { ZERO_PROPERTIES as NI_ZERO_PROPERTIES } from "@/hardware/ni/device/types";
import { Layout } from "@/layout";

export interface SelectProps {
  configureLayout: Layout.BaseState;
  emptyContent?: string | JSX.Element;
  label?: string;
  make: string;
  path?: string;
  icon?: Icon.ReactElement;
}

export const Select = ({
  configureLayout,
  emptyContent = "No devices connected.",
  label = "Device",
  make,
  path = "config.device",
  icon,
}: SelectProps) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  const handleDeviceChange = useCallback(
    (key: device.Key, { set }: Form.ContextValue) => {
      if (client == null || primitive.isZero(key)) return;
      handleError(async () => {
        const device = await client.hardware.devices.retrieve({ key });
        const { configured, rack } = device;
        set("rackKey", rack);
        if (configured) return;
        
        // Auto-configure LabJack and NI devices without showing popup
        //
        // May be able to completely remove popup related code if/when 
        // we migrate away from the naming conventions used to match 
        // using the  location as the immutable identifier. 
        //
        // Changing the name/location should be a command request
        // to the respective device service registry. 

        // TODO: Reduce complexity after device handling refactor

        if (device.make === "LabJack" || device.make === "NI") {
          const identifier = device.location.toLowerCase();
          const initialProperties = device.make === "LabJack" ? LABJACK_ZERO_PROPERTIES : NI_ZERO_PROPERTIES;
          
          await client.hardware.devices.create({
            ...device,
            configured: true,
            properties: {
              ...deep.copy(initialProperties),
              ...device.properties,
              identifier,
            },
          });
          return;
        }

        placeLayout({ ...configureLayout, key });
      }, "Failed to retrieve device");
    },
    [client, placeLayout, configureLayout, handleError],
  );
  return (
    <Form.Field<string>
      grow
      label={label}
      onChange={handleDeviceChange}
      path={path}
      style={{ flexBasis: 150 }}
    >
      {({ value, onChange, variant }) => (
        <Device.SelectSingle
          value={value}
          onChange={onChange}
          initialParams={{ makes: [make] }}
          filter={(p) => p.make === make}
          emptyContent={emptyContent}
          grow
          icon={icon}
          variant={variant}
        />
      )}
    </Form.Field>
  );
};
