import { binary, UnknownRecord } from "@synnaxlabs/x";
import { z } from "zod";

import { rackKeyZ } from "@/hardware/rack/payload";
import { ontology } from "@/ontology";

export const deviceKeyZ = z.string();

export const deviceZ = z.object({
  key: deviceKeyZ,
  rack: rackKeyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  location: z.string(),
  configured: z.boolean().optional(),
  properties: z.record(z.unknown()).or(
    z.string().transform((c) => {
      if (c === "") return {};
      return binary.JSON_ECD.decodeString(c);
    }),
  ) as z.ZodType<UnknownRecord>,
});

export type Device<P extends UnknownRecord = UnknownRecord> = Omit<
  z.output<typeof deviceZ>,
  "properties"
> & { properties: P };

export type DeviceKey = z.infer<typeof deviceKeyZ>;

export const newDeviceZ = deviceZ.extend({
  properties: z.unknown().transform((c) => binary.JSON_ECD.encodeString(c)),
});

export type NewDevice<P extends UnknownRecord = UnknownRecord> = Omit<
  z.input<typeof newDeviceZ>,
  "properties"
> & { properties: P };

export const DeviceOntologyType = "device" as ontology.ResourceType;

export const ontologyID = (key: DeviceKey): ontology.ID =>
  new ontology.ID({ type: DeviceOntologyType, key: key.toString() });
