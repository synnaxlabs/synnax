// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Device, Synnax } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type JSX } from "react";

export interface SelectProps extends Pick<Device.SelectSingleProps, "filter"> {
  onConfigure: (deviceKey: device.Key) => void;
  emptyContent?: string | JSX.Element;
  label?: string;
  make: string;
  model?: string;
  path?: string;
  icon?: Icon.ReactElement;
}

export const Select = ({
  onConfigure,
  emptyContent = "No devices connected",
  filter: filterProp,
  label = "Device",
  make,
  model,
  path = "config.device",
  icon,
}: SelectProps) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleDeviceChange = (key: device.Key, { set }: Form.ContextValue) => {
    if (client == null || primitive.isZero(key)) return;
    handleError(async () => {
      const { configured, rack } = await client.devices.retrieve({ key });
      set("rack", rack);
      if (configured) return;
      onConfigure(key);
    }, "Failed to retrieve device");
  };
  const filter = (d: device.Device) =>
    d.make === make &&
    (model == null || d.model === model) &&
    (filterProp?.(d) ?? true);
  return (
    <Form.Field<string> grow label={label} onChange={handleDeviceChange} path={path}>
      {({ value, onChange, preview }) => (
        <Device.SelectSingle
          value={value}
          onChange={onChange}
          initialQuery={{ makes: [make] }}
          filter={filter}
          emptyContent={emptyContent}
          grow
          icon={icon}
          preview={preview}
        />
      )}
    </Form.Field>
  );
};
