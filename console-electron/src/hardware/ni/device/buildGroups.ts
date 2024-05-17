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
  type GroupConfig,
  type EnrichedProperties,
} from "@/hardware/ni/device/types";

const buildAnalogInputGroups = (
  info: EnrichedProperties,
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
  info: EnrichedProperties,
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

  const ackGroupTime: ChannelConfig = {
    key: nanoid(),
    dataType: "timestamp",
    role: "index",
    name: `${identifierLower}_ao_ack_time`,
    isIndex: true,
    line: -1,
    port: -1,
  };

  const ackGroupData: ChannelConfig[] = Array.from(
    { length: info.analogOutput.portCount },
    (_, i) => ({
      key: nanoid(),
      role: "analogOutputAck",
      dataType: "float32",
      name: `${identifierLower}_ao_ack_${i + 1}`,
      isIndex: false,
      line: -1,
      port: i + 1,
    }),
  );

  const ackGroup: GroupConfig = {
    name: `Analog Output Ack`,
    key: nanoid(),
    channelPrefix: `${identifierLower}_ao_ack_`,
    channelSuffix: "",
    role: "analogOutputAck",
    channels: [ackGroupTime, ...ackGroupData],
  };

  return [...commandGroups, ackGroup];
};

const buildDigitalInputOutputGroups = (
  info: EnrichedProperties,
  identifier: string,
): GroupConfig[] => {
  const commandGroups: GroupConfig[] = [];
  const ackGroup: GroupConfig = {
    name: "Digital Inputs",
    role: "digitalInput",
    key: nanoid(),
    channelPrefix: `${identifier.toLowerCase()}__di_`,
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
      const prefix = `${identifier.toLowerCase()}_do_`;
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
            name: `${prefix}cmd_${port}_${line}`,
            isIndex: false,
            line,
            port,
          },
          {
            key: nanoid(),
            dataType: "timestamp",
            role: "index",
            name: `${prefix}cmd_time_${port}_${line}`,
            isIndex: true,
            line: -1,
            port: -1,
          },
        ],
      });
      ackGroup.channels.push({
        key: nanoid(),
        dataType: "uint8",
        role: "digitalInput",
        name: `${prefix}di_${port}_${line}`,
        isIndex: false,
        line,
        port,
      });
    }
  });
  return [...commandGroups, ackGroup];
};

const buildDigitalInputGroups = (
  info: EnrichedProperties,
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
  info: EnrichedProperties,
  identifier: string,
): GroupConfig[] => {
  const commandGroups: GroupConfig[] = [];
  const ackGroup: GroupConfig = {
    name: "Digital Output Acknowledgements",
    key: nanoid(),
    channelPrefix: `${identifier.toLowerCase()}_do_ack`,
    role: "digitalOutputAck",
    channelSuffix: "",
    channels: [
      {
        key: nanoid(),
        dataType: "timestamp",
        name: `${identifier.toLowerCase()}_do_ack_time`,
        role: "index",
        isIndex: true,
        line: 0,
        port: 0,
      },
    ],
  };
  info.digitalInputOutput.lineCounts.forEach((lineCount, i) => {
    const port = i + 1;
    for (let j = 0; j < lineCount; j++) {
      const line = j + 1;
      const prefix = `${identifier.toLowerCase()}_do_`;
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
            name: `${prefix}cmd_${port}_${line}`,
            isIndex: false,
            role: "digitalOutputCommand",
            line,
            port,
          },
          {
            key: nanoid(),
            dataType: "timestamp",
            name: `${prefix}cmd_time_${port}_${line}`,
            role: "index",
            isIndex: true,
            line: 0,
            port: 0,
          },
        ],
      });
      ackGroup.channels.push({
        key: nanoid(),
        dataType: "bool",
        name: `${prefix}_ack_${port}_${line}`,
        role: "digitalOutputAck",
        isIndex: false,
        line,
        port,
      });
    }
  });
  return [...commandGroups, ackGroup];
};

export const buildPhysicalDevicePlan = (
  input: EnrichedProperties,
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
