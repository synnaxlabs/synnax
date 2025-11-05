// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Select, Synnax } from "@synnaxlabs/pluto";
import { deep, type record } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { type Device } from "@/hardware/ni/device";
import {
  CI_CHANNEL_SCHEMAS,
  CI_CHANNEL_TYPE_ICONS,
  CI_CHANNEL_TYPE_NAMES,
  type CIChannel,
  type CIChannelType,
  ZERO_CI_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends record.KeyedNamed<CIChannelType> {
  icon?: ReactElement;
}

export type SelectCIChannelTypeFieldProps = Form.SelectFieldProps<CIChannelType, Entry>;

export const SelectCIChannelTypeField = (
  props: SelectCIChannelTypeFieldProps,
): ReactElement => {
  const client = Synnax.use();
  const ctx = Form.useContext();
  const { path } = props;
  const parentPath = path.slice(0, path.lastIndexOf("."));
  const deviceKey = Form.useFieldValue<string>(`${parentPath}.device`, {
    optional: true,
  });
  const [supportedTypes, setSupportedTypes] = useState<CIChannelType[] | null>(null);

  // Fetch device capabilities when device changes
  useEffect(() => {
    if (deviceKey == null || client == null) {
      setSupportedTypes(null);
      return;
    }

    const fetchCapabilities = async (): Promise<void> => {
      try {
        const device = await client.hardware.devices.retrieve<Device.Properties>({
          key: deviceKey,
        });
        const supportedCITypes = device.properties.supportedCITypes;
        // If device has capability metadata, use it; otherwise show all types (backward compat)
        setSupportedTypes(supportedCITypes ?? null);
      } catch (error) {
        console.warn("Failed to fetch device capabilities:", error);
        setSupportedTypes(null); // Fall back to showing all types
      }
    };

    void fetchCapabilities();
  }, [deviceKey, client]);

  // Filter options based on supported types
  const data = Object.keys(CI_CHANNEL_TYPE_NAMES).map((key) => {
    const type = key as CIChannelType;
    const Icon = CI_CHANNEL_TYPE_ICONS[type];
    const isSupported = supportedTypes == null || supportedTypes.includes(type);
    return {
      key: type,
      name: CI_CHANNEL_TYPE_NAMES[type],
      icon: <Icon color={isSupported ? 8 : 6} />,
      disabled: !isSupported,
    };
  });

  return (
    <Form.Field<CIChannelType>
      label="Channel Type"
      {...props}
      onChange={(value) => {
        const prevType = ctx.get<CIChannelType>(path).value;
        if (prevType === value) return;
        const next = deep.copy(ZERO_CI_CHANNELS[value]);
        const prevParent = ctx.get<CIChannel>(parentPath).value;
        const schema = CI_CHANNEL_SCHEMAS[value];
        const nextValue = {
          ...deep.overrideValidItems(next, prevParent, schema),
          type: next.type,
          ...("minVal" in next && { minVal: next.minVal }),
          ...("maxVal" in next && { maxVal: next.maxVal }),
        };
        ctx.set(parentPath, nextValue);
      }}
    >
      {(fieldProps) => (
        <Select.Static<CIChannelType, Entry>
          {...fieldProps}
          resourceName="Channel Type"
          data={data}
        />
      )}
    </Form.Field>
  );
};
