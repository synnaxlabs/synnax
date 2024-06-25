// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { nanoid } from "nanoid/non-secure";

import {
  type ChannelConfig,
  type Properties,
  type GroupConfig,
} from "@/hardware/ni/device/types";

const buildAnalogInputGroups = (
  info: Properties,
  identifier: string,
): GroupConfig[] => {
  const prefix = `${identifier.toLowerCase()}_ai_`;
  const dataChannels: ChannelConfig[] = Array.from(
    { length: info.analogInput.portCount },
    (_, i) => ({
      key: nanoid(),
      role: "analogInput",
      dataType: "float32",
      name: `${prefix}${i + 1}`,
      isIndex: false,
      line: -1,
      port: i + 1,
    }),
  );

  const idXChannel: ChannelConfig = {
    key: nanoid(),
    role: "analogIndex",
    dataType: "timestamp",
    name: `${prefix}idx`,
    isIndex: true,
    line: -1,
    port: 0,
  };

  return [
    {
      name: "Analog Inputs",
      key: nanoid(),
      role: "analogInput",
      channelPrefix: prefix,
      channelSuffix: "",
      channels: [idXChannel, ...dataChannels],
    },
  ];
};

const buildAnalogOutputGroups = (
  info: Properties,
  identifier: string,
): GroupConfig[] => {
  const identifierLower = identifier.toLowerCase();
  const commandGroups: GroupConfig[] = Array.from(
    { length: info.analogOutput.portCount },
    (_, i) => {
      const port = i + 1;
      const prefix = `${identifierLower}_ao_`;
      const cmdGroup: GroupConfig = {
        name: `Analog Output ${port}`,
        key: nanoid(),
        role: "analogOutputCommand",
        channelPrefix: prefix,
        channelSuffix: "",
        channels: [
          {
            key: nanoid(),
            role: "analogOutputCommand",
            dataType: "float32",
            name: `${prefix}cmd_${port}`,
            isIndex: false,
            line: -1,
            port,
          },
          {
            key: nanoid(),
            dataType: "timestamp",
            role: "index",
            name: `${prefix}cmd_${port}_time`,
            isIndex: true,
            line: -1,
            port: -1,
          },
        ],
      };
      return cmdGroup;
    },
  );

  const stateGroupTime: ChannelConfig = {
    key: nanoid(),
    dataType: "timestamp",
    role: "index",
    name: `${identifierLower}_ao_state_time`,
    isIndex: true,
    line: -1,
    port: -1,
  };

  const stateGroupData: ChannelConfig[] = Array.from(
    { length: info.analogOutput.portCount },
    (_, i) => ({
      key: nanoid(),
      role: "analogOutputState",
      dataType: "float32",
      name: `${identifierLower}_ao_state_${i + 1}`,
      isIndex: false,
      line: -1,
      port: i + 1,
    }),
  );

  const stateGroup: GroupConfig = {
    name: `Analog Output State`,
    key: nanoid(),
    channelPrefix: `${identifierLower}_ao_state_`,
    channelSuffix: "",
    role: "analogOutputState",
    channels: [stateGroupTime, ...stateGroupData],
  };

  return [...commandGroups, stateGroup];
};

const buildDigitalInputOutputGroups = (
  info: Properties,
  identifier: string,
): GroupConfig[] => {
  const commandGroups: GroupConfig[] = [];
  const stateGroup: GroupConfig = {
    name: "Digital Inputs",
    role: "digitalInput",
    key: nanoid(),
    channelPrefix: `${identifier.toLowerCase()}_di_`,
    channelSuffix: "",
    channels: [
      {
        key: nanoid(),
        dataType: "timestamp",
        name: `${identifier.toLowerCase()}_di_time`,
        isIndex: true,
        role: "index",
        line: -1,
        port: -1,
      },
    ],
  };
  info.digitalInputOutput.lineCounts.forEach((lineCount, i) => {
    const port = i + 1;
    for (let j = 0; j < lineCount; j++) {
      const line = j + 1;
      const portLine = `${port}_${line}`;
      const prefix = `${identifier.toLowerCase()}_`;
      commandGroups.push({
        key: nanoid(),
        name: `Digital Output ${port}/${line}`,
        role: "digitalOutputCommand",
        channelPrefix: prefix,
        channelSuffix: "",
        channels: [
          {
            key: nanoid(),
            dataType: "uint8",
            role: "digitalOutputCommand",
            name: `${prefix}do_${portLine}_cmd`,
            isIndex: false,
            line,
            port,
          },
          {
            key: nanoid(),
            dataType: "timestamp",
            role: "index",
            name: `${prefix}cmd_${portLine}_time`,
            isIndex: true,
            line: -1,
            port: -1,
          },
        ],
      });
      stateGroup.channels.push({
        key: nanoid(),
        dataType: "uint8",
        role: "digitalInput",
        name: `${prefix}di_${portLine}`,
        isIndex: false,
        line,
        port,
      });
    }
  });
  return [...commandGroups, stateGroup];
};

const buildDigitalInputGroups = (
  info: Properties,
  identifier: string,
): GroupConfig[] => {
  const prefix = `${identifier.toLowerCase()}_di_`;
  const timeChannel: ChannelConfig = {
    key: nanoid(),
    dataType: "timestamp",
    name: `${prefix}time`,
    role: "index",
    isIndex: true,
    line: 0,
    port: 0,
  };
  const dataChannels: ChannelConfig[] = [];
  info.digitalInput.lineCounts.forEach((lineCount, i) => {
    const port = i + 1;
    for (let j = 0; j < lineCount; j++) {
      const line = j + 1;
      dataChannels.push({
        key: nanoid(),
        dataType: "bool",
        role: "digitalInput",
        name: `${prefix}${port}_${line}`,
        isIndex: false,
        line,
        port,
      });
    }
  });
  return [
    {
      name: "Digital Inputs",
      key: nanoid(),
      channelPrefix: prefix,
      channelSuffix: "",
      role: "digitalInput",
      channels: [timeChannel, ...dataChannels],
    },
  ];
};

const buildDigitalOutputGroups = (
  info: Properties,
  identifier: string,
): GroupConfig[] => {
  const commandGroups: GroupConfig[] = [];
  const stateGroup: GroupConfig = {
    name: "Digital Output State",
    key: nanoid(),
    channelPrefix: `${identifier.toLowerCase()}_do_state`,
    role: "digitalOutputState",
    channelSuffix: "",
    channels: [
      {
        key: nanoid(),
        dataType: "timestamp",
        name: `${identifier.toLowerCase()}_do_state_time`,
        role: "index",
        isIndex: true,
        line: 0,
        port: 0,
      },
    ],
  };
  info.digitalOutput.lineCounts.forEach((lineCount, i) => {
    const port = i + 1;
    for (let j = 0; j < lineCount; j++) {
      const line = j + 1;
      const prefix = `${identifier.toLowerCase()}_do_${port}_${line}`;
      commandGroups.push({
        key: nanoid(),
        name: `Digital Output ${port}/${line}`,
        channelPrefix: prefix,
        channelSuffix: "",
        role: "digitalOutputCommand",
        channels: [
          {
            key: nanoid(),
            dataType: "uint8",
            name: `${prefix}cmd`,
            isIndex: false,
            role: "digitalOutputCommand",
            line,
            port,
          },
          {
            key: nanoid(),
            dataType: "timestamp",
            name: `${prefix}cmd_time`,
            role: "index",
            isIndex: true,
            line: 0,
            port: 0,
          },
        ],
      });
      stateGroup.channels.push({
        key: nanoid(),
        dataType: "bool",
        name: `${prefix}do_state`,
        role: "digitalOutputState",
        isIndex: false,
        line,
        port,
      });
    }
  });
  return [...commandGroups, stateGroup];
};

export const buildPhysicalDevicePlan = (
  input: Properties,
  identifier: string,
): GroupConfig[] => {
  const groups: GroupConfig[] = [];
  if (input.analogInput.portCount > 0)
    groups.push(...buildAnalogInputGroups(input, identifier));
  if (input.analogOutput.portCount > 0)
    groups.push(...buildAnalogOutputGroups(input, identifier));
  if (input.digitalInput.portCount > 0)
    groups.push(...buildDigitalInputGroups(input, identifier));
  if (input.digitalOutput.portCount > 0)
    groups.push(...buildDigitalOutputGroups(input, identifier));
  if (input.digitalInputOutput.portCount > 0)
    groups.push(...buildDigitalInputOutputGroups(input, identifier));
  return groups;
};
