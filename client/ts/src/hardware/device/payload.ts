// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, record, status, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ as rackKeyZ } from "@/hardware/rack/payload";
import { decodeJSONString } from "@/util/decodeJSONString";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

export const statusZ = status.statusZ(z.object({ rack: rackKeyZ, device: keyZ }));

export interface Status extends z.infer<typeof statusZ> {}

export const deviceZ = z.object({
  key: keyZ,
  rack: rackKeyZ,
  name: z.string(),
  make: z.string(),
  model: z.string(),
  location: z.string(),
  configured: z.boolean().optional(),
  properties: record.unknownZ.or(z.string().transform(decodeJSONString)),
  status: zod.nullToUndefined(statusZ),
});

export interface Device<
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
> extends Omit<z.infer<typeof deviceZ>, "properties" | "status"> {
  properties: Properties;
  make: Make;
  model: Model;
  status?: Status;
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
