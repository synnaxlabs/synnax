import { z } from "zod";

import { ontology } from "@/ontology";

export const rackKeyZ = z.number();

export type RackKey = z.infer<typeof rackKeyZ>;

export const rackZ = z.object({
  key: rackKeyZ,
  name: z.string(),
});

export type RackPayload = z.infer<typeof rackZ>;

export const newRackZ = rackZ.partial({ key: true });

export type NewRack = z.input<typeof newRackZ>;

export const RackOntologyType = "rack" as ontology.ResourceType;

export const ontologyID = (key: RackKey): ontology.ID =>
  new ontology.ID({ type: RackOntologyType, key: key.toString() });
