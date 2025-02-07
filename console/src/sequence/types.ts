import { channel, type task } from "@synnaxlabs/client";
import { z } from "zod";

export const SEQUENCE_TYPE = "sequence";
export type SequenceType = typeof SEQUENCE_TYPE;

export const configZ = z.object({
  rate: z.number().min(1),
  read: z.array(channel.keyZ),
  write: z.array(channel.keyZ),
  script: z.string(),
  globals: z.record(z.string(), z.any()),
});

export type Config = z.infer<typeof configZ>;

export const ZERO_CONFIG: Config = {
  rate: 10,
  read: [],
  write: [],
  script: "",
  globals: {},
};

export const stateDetailsZ = z.object({
  running: z.boolean(),
});

export type StateDetails = z.infer<typeof stateDetailsZ>;

export type Task = task.Task<Config, StateDetails, "sequence">;
export type Payload = task.Payload<Config, StateDetails, "sequence">;

export const ZERO_PAYLOAD: Payload = {
  key: "",
  name: "Sequence Task",
  config: ZERO_CONFIG,
  type: SEQUENCE_TYPE,
};
