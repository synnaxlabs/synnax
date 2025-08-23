// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";
import { Channel, Flex, Icon, Text } from "@synnaxlabs/pluto";
import { type Optional, primitive } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { useSelectActiveKey as useSelectActiveRangeKey } from "@/range/selectors";

const CHANNEL_NAME_REGEX = /^[a-z0-9_-]+$/;

const cleanChannelName = (name: string): string =>
  name
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "_")
    .replace(/[^a-z0-9_-]/g, "")
    .replace(/_{2,}/g, "_")
    .replace(/-{2,}/g, "-")
    .replace(/^[_-]+|[_-]+$/g, "");

const validateChannelName = (name: string): boolean => {
  const cleaned = cleanChannelName(name);
  return cleaned.length > 0 && CHANNEL_NAME_REGEX.test(cleaned);
};

export interface ChannelNameProps
  extends Optional<Omit<Text.MaybeEditableProps, "value">, "level"> {
  channel: channel.Key;
  defaultName?: string;
  previewDevice?: device.Device;
  previewChannelType?: string;
  previewPort?: number;
  previewLine?: number;
  customName?: string;
  onCustomNameChange?: (name: string) => void;
}

const generatePreviewName = (
  device: device.Device,
  channelType?: string,
  port?: number,
  line?: number,
): string => {
  if (!channelType) return "";
  const identifier = (device.properties as any)?.identifier || device.name;
  const portStr = port !== undefined ? `_${port}` : "";
  const lineStr = line !== undefined ? `_${line}` : "";
  
  // Check if channelType has a suffix (_cmd or _state) or is just cmd/state
  const cmdSuffix = "_cmd";
  const stateSuffix = "_state";
  let baseChannelType = channelType;
  let suffix = "";
  
  if (channelType.endsWith(cmdSuffix)) {
    baseChannelType = channelType.slice(0, -cmdSuffix.length);
    suffix = cmdSuffix;
  } else if (channelType.endsWith(stateSuffix)) {
    baseChannelType = channelType.slice(0, -stateSuffix.length);
    suffix = stateSuffix;
  } else if (channelType === "cmd") {
    baseChannelType = "";
    suffix = cmdSuffix;
  } else if (channelType === "state") {
    baseChannelType = "";
    suffix = stateSuffix;
  }
  
  // Handle LabJack-specific naming conventions
  const isLabJack = (device.make === "LabJack");
  let channelName = "";
  
  if (isLabJack) {
    // LabJack naming: dev2_ain0, dev2_dio4, etc.
    // Convert LabJack port names: "AIN0"→"ain0", "FIO4"→"dio4", "AO0"→"ao0"
    let labjackPortName = baseChannelType.toLowerCase();
    if (labjackPortName.startsWith('fio'))
      labjackPortName = labjackPortName.replace('fio', 'dio');
    
    channelName = suffix 
      ? `${identifier}_${labjackPortName}${suffix}`
      : `${identifier}_${labjackPortName}`;
  } else
    // NI naming: dev1_0_cmd, dev1_0_state (remove ao/ai/do/di from names)
    channelName = suffix 
      ? `${identifier}${portStr}${lineStr}${suffix}`
      : `${identifier}${portStr}${lineStr}`;
  
  return cleanChannelName(channelName);
};

export const ChannelName = ({
  channel,
  defaultName = "No Channel", 
  className,
  previewDevice,
  previewChannelType,
  previewPort,
  previewLine,
  customName,
  onCustomNameChange,
  ...rest
}: ChannelNameProps) => {
  const range = useSelectActiveRangeKey();
  const { data } = Channel.retrieve.useDirect({
    params: { key: channel, rangeKey: range ?? undefined },
  });
  const { update: rename } = Channel.rename.useDirect({ params: { key: channel } });
  
  const isUnconfigured = primitive.isZero(channel);
  let name = data?.name ?? defaultName;
  let isPreview = false;
  let canEdit = false;
  
  if (isUnconfigured) {
    canEdit = true;
    if (previewDevice) {
      name = customName || generatePreviewName(previewDevice, previewChannelType, previewPort, previewLine);
      isPreview = true;
    } else if (customName) name = customName;
  }

  const handleChange = (newName: string) => {
    const cleanedName = cleanChannelName(newName);
    if (!validateChannelName(cleanedName)) return;
    
    if (isUnconfigured && onCustomNameChange) onCustomNameChange(cleanedName);
    else rename(cleanedName);
  };

  const handleEditClick = () => {
    if (rest.id) Text.edit(rest.id);
  };

  // For unconfigured channels that can be edited, show edit icon
  if (canEdit) return (
    <Flex.Box direction="x" align="center" gap="small">
      <Icon.Edit 
        style={{
          fontSize: "var(--pluto-small-size)",
          color: "var(--pluto-gray-l6)",
          cursor: "pointer",
        }}
        onClick={handleEditClick}
      />
      <Text.MaybeEditable
        className={CSS(className, CSS.BE("task", "channel-name"))}
        status="warning"
        level="small"
        value={name}
        onChange={handleChange}
        allowDoubleClick={true}
        style={{ 
          color: isPreview ? "var(--pluto-warning-m1)" : undefined,
          ...rest.style 
        }}
        {...rest}
      />
    </Flex.Box>
  );

  // Regular configured channel
  const handleConfiguredChange = (newName: string) => {
    const cleanedName = cleanChannelName(newName);
    if (validateChannelName(cleanedName)) rename(cleanedName);
  };

  return (
    <Text.MaybeEditable
      className={CSS(className, CSS.BE("task", "channel-name"))}
      level="small"
      value={name}
      onChange={handleConfiguredChange}
      allowDoubleClick={false}
      {...rest}
    />
  );
};
