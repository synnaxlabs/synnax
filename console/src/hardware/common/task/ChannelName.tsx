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
  channelType: string,
  port?: number,
  line?: number,
): string => {
  const deviceName = device.name;
  const portStr = port !== undefined ? `_${port}` : "";
  const lineStr = line !== undefined ? `_${line}` : "";
  return `${deviceName}_${channelType}${portStr}${lineStr}`;
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
  
  // If no channel exists yet
  if (isUnconfigured) {
    if (previewDevice) {
      // Has preview information - use custom name or generate preview name
      if (customName) {
        name = customName;
      } else {
        name = generatePreviewName(previewDevice, previewChannelType!, previewPort, previewLine);
      }
      isPreview = true;
    } else if (customName) {
      // No device but has custom name - use it
      name = customName;
    }
    canEdit = true; // Allow editing when no channel exists (yellow preview or "No Channel" state)
  }

  const handleChange = (newName: string) => {
    if (isUnconfigured && onCustomNameChange) {
      // For preview channels, update the custom name
      onCustomNameChange(newName);
    } else {
      // For existing channels, rename them
      rename(newName);
    }
  };

  const handleEditClick = () => {
    const elementId = rest.id;
    if (elementId) {
      Text.edit(elementId);
    }
  };

  // For unconfigured channels that can be edited, show edit icon
  if (canEdit) {
    return (
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
  }

  // Regular configured channel
  return (
    <Text.MaybeEditable
      className={CSS(className, CSS.BE("task", "channel-name"))}
      level="small"
      value={name}
      onChange={rename}
      allowDoubleClick={false}
      {...rest}
    />
  );
};
