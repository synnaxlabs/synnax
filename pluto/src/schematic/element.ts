import z from "zod";

import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";

export const elementConfigZ = z.discriminatedUnion("variant", [
  ...Node.configZ.options,
  ...Edge.configZ.options,
]);
export type ElementConfig = z.infer<typeof elementConfigZ>;

export const ELEMENT_REGISTRY = { ...Node.REGISTRY, ...Edge.REGISTRY };
