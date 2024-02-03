import { type hardware } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";
import { z } from "zod";

const linearScaleZ = z.object({
  type: z.literal("linear"),
  one: xy.xy,
  two: xy.xy,
});

const analogInputScaleZ = linearScaleZ;

const analogInputVoltageChannelZ = z.object({
  key: z.string(),
  type: z.literal("analogVoltageInput"),
  enabled: z.boolean(),
  port: z.number(),
  channel: z.number(),
  scale: z.optional(analogInputScaleZ),
});

export const analogReadChannelZ = analogInputVoltageChannelZ;

export type AnalogInputVoltageChannel = z.infer<typeof analogInputVoltageChannelZ>;

export const analodReadTaskConfigZ = z
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
  );

export type AnalogReadTaskConfig = z.infer<typeof analodReadTaskConfigZ>;

export type AnalogReadTask = hardware.Task<"ni.analogRead", AnalogReadTaskConfig>;

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

export type DigitalWriteTask = hardware.Task<"ni.digitalWrite", DigitalWriteTaskConfig>;

const digitalReadChannelZ = digitalInputChannelZ;

export const digitalReadTaskConfigZ = z.object({
  device: z.string().min(1),
  channels: z.array(digitalReadChannelZ),
});

export type DigitalReadTaskConfig = z.infer<typeof digitalReadTaskConfigZ>;

export type DigitalReadTask = hardware.Task<"ni.analogWrite", DigitalReadTaskConfig>;

export type NITask = AnalogReadTask | DigitalWriteTask | DigitalReadTask;

export type NIChannel =
  | DigitalInputChannel
  | AnalogInputVoltageChannel
  | DigitalOutputChannel;
