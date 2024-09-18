import { access } from "@synnaxlabs/client";
import { z } from "zod";

export const stateZ = z.object({
  version: z.literal("0.0.0"),
  policies: access.policyZ.array(),
});

export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  version: "0.0.0",
  policies: [],
};
