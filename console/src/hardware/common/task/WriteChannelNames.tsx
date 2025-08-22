// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";
import { useCallback } from "react";

import { CSS } from "@/css";
import { ChannelName, type ChannelNameProps } from "@/hardware/common/task/ChannelName";
import { getChannelNameID } from "@/hardware/common/task/getChannelNameID";

const CMD_SUFFIX = "_cmd";
const STATE_SUFFIX = "_state";

const extractBaseName = (name: string): string => {
  if (name.endsWith(CMD_SUFFIX)) return name.slice(0, -CMD_SUFFIX.length);
  if (name.endsWith(STATE_SUFFIX)) return name.slice(0, -STATE_SUFFIX.length);
  return name;
};

export interface WriteChannelNamesProps
  extends Omit<ChannelNameProps, "channel" | "defaultName" | "id"> {
  cmdChannel: channel.Key;
  stateChannel: channel.Key;
  itemKey: string;
  previewDevice?: device.Device;
  previewChannelType?: string;
  previewPort?: number;
  previewLine?: number;
  customName?: string;
  onCustomNameChange?: (name: string) => void;
}

export const WriteChannelNames = ({
  cmdChannel,
  stateChannel,
  itemKey,
  previewDevice,
  previewChannelType,
  previewPort,
  previewLine,
  customName,
  onCustomNameChange,
  ...rest
}: WriteChannelNamesProps) => {
  const handleNameChange = useCallback(
    (newName: string) => {
      if (onCustomNameChange == null) return;
      const baseName = extractBaseName(newName);
      onCustomNameChange(baseName);
    },
    [onCustomNameChange],
  );

  return (
    <>
      <ChannelName
        channel={cmdChannel}
        defaultName="No Command Channel"
        id={getChannelNameID(itemKey, "cmd")}
        previewDevice={previewDevice}
        previewChannelType={previewChannelType ? `${previewChannelType}${CMD_SUFFIX}` : "cmd"}
        previewPort={previewPort}
        previewLine={previewLine}
        customName={customName ? `${customName}${CMD_SUFFIX}` : undefined}
        onCustomNameChange={handleNameChange}
        {...rest}
      />
      <ChannelName
        channel={stateChannel}
        className={CSS.B("state-channel")}
        defaultName="No State Channel"
        id={getChannelNameID(itemKey, "state")}
        previewDevice={previewDevice}
        previewChannelType={previewChannelType ? `${previewChannelType}${STATE_SUFFIX}` : "state"}
        previewPort={previewPort}
        previewLine={previewLine}
        customName={customName ? `${customName}${STATE_SUFFIX}` : undefined}
        onCustomNameChange={handleNameChange}
        {...rest}
      />
    </>
  );
};
