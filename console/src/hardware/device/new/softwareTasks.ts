import { type hardware } from "@synnaxlabs/client";

import {
  DigitalWriteTaskConfig,
  type AnalogReadTaskConfig,
} from "@/hardware/configure/ni/types";

import {
  type EnrichedProperties,
  type PhysicalGroupPlan,
  type PhysicalPlan,
} from "./types";

const buildAnalogReadTask = (
  properties: EnrichedProperties,
  group: PhysicalGroupPlan,
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
