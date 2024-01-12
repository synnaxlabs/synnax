import { xy } from "@synnaxlabs/x";
import { z } from "zod";

export namespace AnalogInput {
  export const type = z.literal("ni-analog-input");
  const linearScale = z.object({
    type: z.literal("linear"),
    one: xy.xy,
    two: xy.xy,
  });

  const scale = linearScale;

  const voltageChannel = z.object({
    key: z.string(),
    type: z.literal("voltage"),
    active: z.boolean(),
    port: z.number(),
    channel: z.number(),
    scale: z.optional(scale),
  });

  export type Channel = z.infer<typeof voltageChannel>;

  export const channel = voltageChannel;

  export const config = z
    .object({
      type,
      device: z.string().min(1),
      sampleRate: z.number().min(0).max(50000),
      streamRate: z.number().min(0).max(50000),
      channels: z.array(channel),
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

  export const error = z.object({
    field: z.string().array(),
    message: z.string(),
  });

  export const errorPayload = z.object({
    errors: z.array(error),
  });
}
