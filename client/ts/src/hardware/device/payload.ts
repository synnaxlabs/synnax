import { z } from "zod";

import { rackKeyZ } from "@/hardware/rack/payload";

export const deviceKeyZ = z.string();

export const deviceZ = z.object({
  key: deviceKeyZ,
  rack: rackKeyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  location: z.string(),
  properties: z.string(),
});

export type Device = z.infer<typeof deviceZ>;
export type DeviceKey = z.infer<typeof deviceKeyZ>;
