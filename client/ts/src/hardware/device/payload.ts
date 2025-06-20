// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, record, status, zod } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { keyZ as rackKeyZ } from "@/hardware/rack/payload";
import { decodeJSONString } from "@/util/decodeJSONString";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

export const stateZ = z.object({
  key: keyZ,
  variant: status.variantZ.or(z.literal("").transform<status.Variant>(() => "info")),
  details: record.unknownZ.or(z.string().transform(decodeJSONString)),
});

export interface State<Details extends {} = record.Unknown>
  extends Omit<z.infer<typeof stateZ>, "details"> {
  details: Details;
}

export const deviceZ = z.object({
  key: keyZ,
  rack: rackKeyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  location: z.string(),
  configured: z.boolean().optional(),
  properties: record.unknownZ.or(z.string().transform(decodeJSONString)),
  state: zod.nullToUndefined(stateZ),
});

export interface Device<
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
  StateDetails extends {} = record.Unknown,
> extends Omit<z.infer<typeof deviceZ>, "properties" | "state"> {
  properties: Properties;
  make: Make;
  model: Model;
  state?: State<StateDetails>;
}

export const newZ = deviceZ.extend({
  properties: z.unknown().transform((c) => binary.JSON_CODEC.encodeString(c)),
});
export interface New<
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
> extends Omit<z.input<typeof newZ>, "properties"> {
  properties: Properties;
  make: Make;
  model: Model;
}

export const ONTOLOGY_TYPE = "device";
export type OntologyType = typeof ONTOLOGY_TYPE;
