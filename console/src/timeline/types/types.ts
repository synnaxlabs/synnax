import { arc } from "@synnaxlabs/client";
import z from "zod";

export const stateZ = z.object({
  key: z.string(),
  name: z.string(),
  nodes: z.array(arc.irNodeZ),
});

export const phaseZ = z.object({
  key: z.string(),
  name: z.string(),
  states: z.array(stateZ),
});

export const sequenceZ = z.object({
  name: z.string(),
  phases: z.array(phaseZ),
});
