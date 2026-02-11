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
import { type JSX, useCallback, useMemo } from "react";

import { Layout } from "@/layout";

export interface SelectProps extends Pick<Device.SelectSingleProps, "filter"> {
  configureLayout: Layout.BaseState;
  emptyContent?: string | JSX.Element;
  label?: string;
  make: string;
  model?: string;
  path?: string;
  icon?: Icon.ReactElement;
}

export const Select = ({
  configureLayout,
  emptyContent = "No devices connected.",
  filter: filterProp,
  label = "Device",
  make,
  model,
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
  const filter = useMemo(() => {
    const baseFilter = (d: device.Device) =>
      d.make === make && (model == null || d.model === model);
    if (filterProp == null) return baseFilter;
    return (d: device.Device) => baseFilter(d) && filterProp(d);
  }, [make, model, filterProp]);
  return (
    <Form.Field<string> grow label={label} onChange={handleDeviceChange} path={path}>
      {({ value, onChange, variant }) => (
        <Device.SelectSingle
          value={value}
          onChange={onChange}
          initialQuery={{ makes: [make] }}
          filter={filter}
          emptyContent={emptyContent}
          grow
          icon={icon}
          variant={variant}
        />
      )}
    </Form.Field>
  );
};
