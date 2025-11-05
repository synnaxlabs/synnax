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
  AO_CHANNEL_SCHEMAS,
  AO_CHANNEL_TYPE_ICONS,
  AO_CHANNEL_TYPE_NAMES,
  AO_CHANNEL_TYPES,
  type AOChannel,
  type AOChannelType,
  ZERO_AO_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends record.KeyedNamed<AOChannelType> {
  icon?: ReactElement;
}

export type SelectAOChannelTypeFieldProps = Form.SelectFieldProps<AOChannelType, Entry>;

export const SelectAOChannelTypeField = (
  props: SelectAOChannelTypeFieldProps,
): ReactElement => {
  const client = Synnax.use();
  const ctx = Form.useContext();
  const { path } = props;
  const parentPath = path.slice(0, path.lastIndexOf("."));
  const deviceKey = Form.useFieldValue<string>(`${parentPath}.device`, {
    optional: true,
  });
  const [supportedTypes, setSupportedTypes] = useState<AOChannelType[] | null>(null);

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
        const supportedAOTypes = device.properties.supportedAOTypes;
        // If device has capability metadata, use it; otherwise show all types (backward compat)
        setSupportedTypes(supportedAOTypes ?? null);
      } catch (error) {
        console.warn("Failed to fetch device capabilities:", error);
        setSupportedTypes(null); // Fall back to showing all types
      }
    };

    void fetchCapabilities();
  }, [deviceKey, client]);

  // Filter options based on supported types
  const data = AO_CHANNEL_TYPES.map((type) => {
    const Icon = AO_CHANNEL_TYPE_ICONS[type];
    const isSupported = supportedTypes == null || supportedTypes.includes(type);
    return {
      key: type,
      name: AO_CHANNEL_TYPE_NAMES[type],
      icon: <Icon color={isSupported ? 8 : 6} />,
      disabled: !isSupported,
    };
  });

  return (
    <Form.Field<AOChannelType>
      label="Channel Type"
      {...props}
      onChange={(value) => {
        const prevType = ctx.get<AOChannelType>(path).value;
        if (prevType === value) return;
        const next = deep.copy(ZERO_AO_CHANNELS[value]);
        const prevParent = ctx.get<AOChannel>(parentPath).value;
        const schema = AO_CHANNEL_SCHEMAS[value];
        ctx.set(parentPath, {
          ...deep.overrideValidItems(next, prevParent, schema),
          type: next.type,
        });
      }}
    >
      {(fieldProps) => (
        <Select.Static<AOChannelType, Entry>
          {...fieldProps}
          allowNone={false}
          resourceName="Channel Type"
          data={data}
        />
      )}
    </Form.Field>
  );
};
