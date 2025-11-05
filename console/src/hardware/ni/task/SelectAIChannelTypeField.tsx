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
  AI_CHANNEL_SCHEMAS,
  AI_CHANNEL_TYPE_ICONS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ZERO_AI_CHANNELS,
} from "@/hardware/ni/task/types";

export interface Entry extends record.KeyedNamed<AIChannelType> {
  icon?: ReactElement;
}

export type SelectAIChannelTypeFieldProps = Form.SelectFieldProps<AIChannelType, Entry>;

export const SelectAIChannelTypeField = (
  props: SelectAIChannelTypeFieldProps,
): ReactElement => {
  const client = Synnax.use();
  const ctx = Form.useContext();
  const { path } = props;
  const parentPath = path.slice(0, path.lastIndexOf("."));
  const deviceKey = Form.useFieldValue<string>(`${parentPath}.device`, {
    optional: true,
  });
  const [supportedTypes, setSupportedTypes] = useState<AIChannelType[] | null>(null);

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
        const supportedAITypes = device.properties.supportedAITypes;
        // If device has capability metadata, use it; otherwise show all types (backward compat)
        setSupportedTypes(supportedAITypes ?? null);
      } catch (error) {
        console.warn("Failed to fetch device capabilities:", error);
        setSupportedTypes(null); // Fall back to showing all types
      }
    };

    void fetchCapabilities();
  }, [deviceKey, client]);

  // Filter options based on supported types
  const data = Object.keys(AI_CHANNEL_TYPE_NAMES).map((key) => {
    const type = key as AIChannelType;
    const Icon = AI_CHANNEL_TYPE_ICONS[type];
    const isSupported = supportedTypes == null || supportedTypes.includes(type);
    return {
      key: type,
      name: AI_CHANNEL_TYPE_NAMES[type],
      icon: <Icon color={isSupported ? 8 : 6} />,
      disabled: !isSupported,
    };
  });

  return (
    <Form.Field<AIChannelType>
      label="Channel Type"
      {...props}
      onChange={(value) => {
        const prevType = ctx.get<AIChannelType>(path).value;
        if (prevType === value) return;
        const next = deep.copy(ZERO_AI_CHANNELS[value]);
        const prevParent = ctx.get<AIChannel>(parentPath).value;
        const schema = AI_CHANNEL_SCHEMAS[value];
        const nextValue = {
          ...deep.overrideValidItems(next, prevParent, schema),
          type: next.type,
        };
        ctx.set(parentPath, nextValue);
      }}
    >
      {(fieldProps) => (
        <Select.Static<AIChannelType, Entry>
          {...fieldProps}
          resourceName="Channel Type"
          data={data}
        />
      )}
    </Form.Field>
  );
};
