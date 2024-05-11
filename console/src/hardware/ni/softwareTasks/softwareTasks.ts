// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type hardware } from "@synnaxlabs/client";

import { DigitalWriteTaskConfig, type AnalogReadTaskConfig } from "@/hardware/ni/types";

import {
  type EnrichedProperties,
  type GroupConfig,
  type PhysicalPlan,
} from "../device/types";

const buildAnalogReadTask = (
  properties: EnrichedProperties,
  group: GroupConfig,
): hardware.Task => {
  const config: AnalogReadTaskConfig = {
    sampleRate: 100,
    streamRate: 25,
    device: properties.location,
    channels: group.channels.map((channel) => ({
      key: channel.key,
      type: "analogVoltageInput",
      enabled: true,
      port: channel.port,
      channel: 0,
      name: channel.name,
    })),
  };
  return {
    key: BigInt(0),
    name: "Analog Read",
    type: "ni-analog-input",
    config,
  };
};

export const buildSoftwareTasks = (
  properties: EnrichedProperties,
  plan: PhysicalPlan,
): hardware.Task[] => {
  return plan.groups.map((group) => buildAnalogReadTask(properties, group));
};
