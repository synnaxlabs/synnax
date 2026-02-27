// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record, status, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { keyZ as rackKeyZ } from "@/rack/payload";

export const keyZ = z.string();
export type Key = z.infer<typeof keyZ>;

export const statusDetailsZ = z.object({ rack: rackKeyZ, device: keyZ });
export const statusZ = status.statusZ(statusDetailsZ);

export interface Status extends z.infer<typeof statusZ> {}

export interface DeviceSchemas<
  Properties extends z.ZodType<record.Unknown> = typeof record.unknownZ,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
> {
  properties?: Properties;
  make?: Make;
  model?: Model;
}

export const deviceZ = <
  Properties extends z.ZodType<record.Unknown> = typeof record.unknownZ,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>({ properties, make, model }: DeviceSchemas<Properties, Make, Model> = {}) =>
  z.object({
    key: keyZ,
    rack: rackKeyZ.min(1, "Must select a location to connect from"),
    name: z.string().min(1, "Name is required"),
    make: make ?? z.string().min(1, "Make is required"),
    model: model ?? z.string().min(1, "Model is required"),
    location: z.string().min(1, "Location is required"),
    configured: z.boolean().optional(),
    parentDevice: keyZ.optional(),
    properties: properties ?? record.nullishToEmpty(),
    status: zod.nullToUndefined(statusZ),
  });

export interface Device<
  Properties extends z.ZodType<record.Unknown> = typeof record.unknownZ,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
> extends Omit<
  z.infer<ReturnType<typeof deviceZ>>,
  "properties" | "make" | "model" | "status"
> {
  properties: z.infer<Properties>;
  make: z.infer<Make>;
  model: z.infer<Model>;
  status?: Status;
}

export const newZ = <
  Properties extends z.ZodType<record.Unknown> = typeof record.unknownZ,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas: DeviceSchemas<Properties, Make, Model> = {},
) =>
  deviceZ(schemas).extend({
    properties: schemas?.properties ?? record.nullishToEmpty(),
  });

export interface New<
  Properties extends z.ZodType<record.Unknown> = typeof record.unknownZ,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
> extends Omit<z.input<ReturnType<typeof newZ>>, "properties"> {
  properties: z.infer<Properties>;
  make: z.infer<Make>;
  model: z.infer<Model>;
}
