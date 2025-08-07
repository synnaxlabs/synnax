// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, type observe, record, status } from "@synnaxlabs/x";
import { z } from "zod";

import { type Key as RackKey } from "@/hardware/rack/payload";
import { decodeJSONString } from "@/util/decodeJSONString";
import { parseWithoutKeyConversion } from "@/util/parseWithoutKeyConversion";

export const keyZ = z.union([
  z.string(),
  z.bigint().transform((k) => k.toString()),
  z.number().transform((k) => k.toString()),
]);
export type Key = z.infer<typeof keyZ>;

export const statusDetailsZ = <D extends z.ZodType>(data: D) =>
  z.object({ task: keyZ, running: z.boolean(), data });

export type StatusDetails<D extends z.ZodType> = z.infer<
  ReturnType<typeof statusDetailsZ<D>>
>;

export const statusZ = <D extends z.ZodType>(data: D) =>
  status.statusZ(statusDetailsZ(data));

export type Status<StatusData extends z.ZodType = z.ZodUnknown> = z.infer<
  ReturnType<typeof statusZ<StatusData>>
>;

export const taskZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodUnknown,
>(
  schemas: Schemas<Type, Config, StatusData> = {
    typeSchema: z.string() as unknown as Type,
    configSchema: z.unknown() as unknown as Config,
    statusDataSchema: z.unknown() as unknown as StatusData,
  },
) =>
  z.object({
    key: keyZ,
    name: z.string(),
    type: schemas.typeSchema,
    internal: z.boolean().optional(),
    config: z.string().transform(decodeJSONString).or(schemas.configSchema),
    status: statusZ(schemas.statusDataSchema).optional().nullable(),
    snapshot: z.boolean().optional(),
  });

export interface Schemas<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  typeSchema: Type;
  configSchema: Config;
  statusDataSchema: StatusData;
}

export type Payload<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> = {
  key: Key;
  name: string;
  type: z.infer<Type>;
  config: z.infer<Config>;
  status?: Status<StatusData>;
  snapshot?: boolean;
  internal?: boolean;
};

export const newZ = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodUnknown,
>(
  schemas?: Schemas<Type, Config, StatusData>,
) =>
  taskZ(schemas)
    .omit({ key: true })
    .extend({
      key: keyZ.transform((k) => k.toString()).optional(),
      config: z.unknown().transform((c) => binary.JSON_CODEC.encodeString(c)),
    });

export type New<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
> = {
  key?: Key;
  name: string;
  type: z.infer<Type>;
  config: z.infer<Config>;
};

export const commandZ = z.object({
  task: keyZ,
  type: z.string(),
  key: z.string(),
  args: record.unknownZ
    .or(z.string().transform(parseWithoutKeyConversion))
    .or(z.array(z.unknown()))
    .or(z.null())
    .optional() as z.ZodOptional<z.ZodType<record.Unknown>>,
});

export interface Command extends Omit<z.infer<typeof commandZ>, "args"> {
  args?: record.Unknown;
}

export interface StateObservable<StatusData extends z.ZodType>
  extends observe.ObservableAsyncCloseable<Status<StatusData>> {}

export interface CommandObservable extends observe.ObservableAsyncCloseable<Command> {}

export const rackKey = (key: Key): RackKey => Number(BigInt(key) >> 32n);
