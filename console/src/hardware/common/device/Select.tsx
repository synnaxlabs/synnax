// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Device, Form, type Icon, Status, Synnax } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type JSX, useCallback } from "react";

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
        const { configured, rack } = await client.devices.retrieve({ key });
        set("rackKey", rack);
        if (configured) return;
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
          initialQuery={{ makes: [make] }}
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
