import { channel, TimeSpan } from "@synnaxlabs/client";
import { z } from "zod";

export const stateZ = z.object({
  key: z.string(),
  version: z.literal("0.0.0"),
  retention: z.number(),
  channels: channel.keyZ.array(),
  remoteCreated: z.boolean(),
});

export type State = z.input<typeof stateZ>;

export const ZERO_STATE: State = {
  key: "",
  version: "0.0.0",
  retention: TimeSpan.hours(1).seconds,
  channels: [],
  remoteCreated: false,
};

export const sliceStateZ = z.object({
  version: z.literal("0.0.0"),
  logs: z.record(stateZ),
});

export type SliceState = z.input<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  logs: {},
};
