// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, type UnknownRecord } from "@synnaxlabs/x";
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
      return binary.JSON_CODEC.decodeString(c);
    }),
  ) as z.ZodType<UnknownRecord>,
});

export type Device<P extends UnknownRecord = UnknownRecord> = Omit<
  z.output<typeof deviceZ>,
  "properties"
> & { properties: P };

export type DeviceKey = z.infer<typeof deviceKeyZ>;

export const newDeviceZ = deviceZ.extend({
  properties: z.unknown().transform((c) => binary.JSON_CODEC.encodeString(c)),
});

export type NewDevice<P extends UnknownRecord = UnknownRecord> = Omit<
  z.input<typeof newDeviceZ>,
  "properties"
> & { properties: P };

export const ONTOLOGY_TYPE: ontology.ResourceType = "device";

export const ontologyID = (key: DeviceKey): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });
