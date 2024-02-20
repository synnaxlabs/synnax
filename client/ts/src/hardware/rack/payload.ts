import { z } from "zod";


export const rackKeyZ = z.number();

export type RackKey = z.infer<typeof rackKeyZ>;

export const rackZ = z.object({
  key: rackKeyZ,
  name: z.string(),
});

export type RackPayload = z.infer<typeof rackZ>;

export const newRackZ = rackZ.partial({ key: true });

export type NewRack = z.input<typeof newRackZ>;