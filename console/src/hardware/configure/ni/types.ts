// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";
import { clientXY } from "node_modules/@synnaxlabs/x/dist/src/spatial/base";
import { z } from "zod";

const linearScaleTypeZ = z.enum(["linear", "none"]);

export type LinearScaleType = z.infer<typeof linearScaleTypeZ>;

const linearScaleZ = z.object({
  type: linearScaleTypeZ,
  one: xy.xy,
  two: xy.xy,
});

export type LinearScale = z.infer<typeof linearScaleZ>;

const DEFAULT_LINEAR_SCALE: LinearScale = {
  type: "none",
  one: { x: 0, y: 0 },
  two: { x: 0, y: 0 },
};

export const DEFAULT_SCALES: Record<LinearScaleType, LinearScale> = {
  linear: DEFAULT_LINEAR_SCALE,
  none: DEFAULT_LINEAR_SCALE,
};

const analogInputScaleZ = linearScaleZ;

const analogInputVoltageChannelZ = z.object({
  key: z.string(),
  type: z.literal("analogVoltageInput"),
  enabled: z.boolean(),
  port: z.number(),
  channel: z.number(),
  scale: analogInputScaleZ,
});

export const analogReadChannelZ = analogInputVoltageChannelZ;

export type AnalogInputVoltageChannel = z.infer<typeof analogInputVoltageChannelZ>;

export const analogReadTaskConfigZ = z
  .object({
    device: z.string().min(1),
    sampleRate: z.number().min(0).max(50000),
    streamRate: z.number().min(0).max(50000),
    channels: z.array(analogReadChannelZ),
  })
  .refine(
    (c) =>
      // Ensure that the stream Rate is lower than the sample rate
      c.sampleRate > c.streamRate,
    {
      path: ["streamRate"],
      message: "Stream rate must be lower than sample rate",
    },
  )
  .superRefine((cfg, ctx) => {
    const ports = new Map<number, number>();
    cfg.channels.forEach(({ port }) => ports.set(port, (ports.get(port) ?? 0) + 1));
    cfg.channels.forEach((channel, i) => {
      if ((ports.get(channel.port) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "port"],
        message: `Port ${channel.port} has already been used`,
      });
    });
  })
  .superRefine((cfg, ctx) => {
    const channels = new Map<number, number>();
    cfg.channels.forEach(({ channel }) =>
      channels.set(channel, (channels.get(channel) ?? 0) + 1),
    );
    cfg.channels.forEach((channel, i) => {
      if ((channels.get(channel.channel) ?? 0) < 2) return;
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["channels", i, "channel"],
        message: `Channel has already been used on port ${channel.port}`,
      });
    });
  });

export type AnalogReadTaskConfig = z.infer<typeof analogReadTaskConfigZ>;

export const ZERO_ANALOG_READ_TASK_CONFIG: AnalogReadTaskConfig = {
  device: "Dev1",
  sampleRate: 10,
  streamRate: 5,
  channels: [],
};

export type AnalogReadTask = task.Task<"ni.analogRead", AnalogReadTaskConfig>;

const digitalOutputChannelZ = z.object({
  key: z.string(),
  type: z.literal("digitalOutput"),
  enabled: z.boolean(),
  port: z.number(),
  line: z.number(),
  channel: z.number(),
});

export type DigitalOutputChannel = z.infer<typeof digitalOutputChannelZ>;

const digitalInputChannelZ = z.object({
  key: z.string(),
  type: z.literal("digitalInput"),
  enabled: z.boolean(),
  port: z.number(),
  line: z.number(),
  channel: z.number(),
});

export type DigitalInputChannel = z.infer<typeof digitalInputChannelZ>;

const digitalWriteChannelZ = z.union([digitalOutputChannelZ, digitalInputChannelZ]);

export const digitalWriteTaskConfigZ = z.object({
  device: z.string().min(1),
  channels: z.array(digitalWriteChannelZ),
});

export type DigitalWriteTaskConfig = z.infer<typeof digitalWriteTaskConfigZ>;

export type DigitalWriteTask = task.Task<"ni.digitalWrite", DigitalWriteTaskConfig>;

const digitalReadChannelZ = digitalInputChannelZ;

export const digitalReadTaskConfigZ = z.object({
  device: z.string().min(1),
  channels: z.array(digitalReadChannelZ),
});

export type DigitalReadTaskConfig = z.infer<typeof digitalReadTaskConfigZ>;

export type DigitalReadTask = task.Task<"ni.analogWrite", DigitalReadTaskConfig>;

export type NITask = AnalogReadTask | DigitalWriteTask | DigitalReadTask;

export type NIChannel =
  | DigitalInputChannel
  | AnalogInputVoltageChannel
  | DigitalOutputChannel;

export const CHANNEL_TYPE_DISPLAY: Record<NIChannel["type"], string> = {
  analogVoltageInput: "Analog Voltage Input",
  digitalInput: "Digital Input",
  digitalOutput: "Digital Output",
};
