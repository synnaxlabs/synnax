// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type observe, record, status } from "@synnaxlabs/x";
import { z } from "zod";

import { type Key as RackKey } from "@/rack/payload";
import { parseWithoutKeyConversion } from "@/util/parseWithoutKeyConversion";

export const keyZ = z.union([
  z.string(),
  z.bigint().transform((k) => k.toString()),
  z.number().transform((k) => k.toString()),
]);
export type Key = z.infer<typeof keyZ>;

export const statusDetailsZ = <DataSchema extends z.ZodType>(data: DataSchema) =>
  z.object({ task: keyZ, running: z.boolean(), data, cmd: z.string().optional() });

export type StatusDetails<DataSchema extends z.ZodType> = z.infer<
  ReturnType<typeof statusDetailsZ<DataSchema>>
>;

export const statusZ = <DataSchema extends z.ZodType>(data: DataSchema) =>
  status.statusZ(statusDetailsZ(data));

export interface Status<StatusData extends z.ZodType = z.ZodUnknown> extends z.infer<
  ReturnType<typeof statusZ<StatusData>>
> {}

const newStatusDetailsZ = <DataSchema extends z.ZodType>(data: DataSchema) =>
  statusDetailsZ(data).partial({ task: true });

export const newStatusZ = <DataSchema extends z.ZodType>(data: DataSchema) =>
  status.statusZ(newStatusDetailsZ(data)).partial({ key: true, name: true });

export interface NewStatus<DataSchema extends z.ZodType = z.ZodType> extends z.infer<
  ReturnType<typeof newStatusZ<DataSchema>>
> {}

export interface Schemas<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  type: Type;
  config: Config;
  statusData: StatusData;
}

export const taskZ = <S extends Schemas = Schemas>(
  schemas = {
    type: z.string() as unknown as S["type"],
    config: record.nullishToEmpty() as S["config"],
    statusData: z.unknown() as S["statusData"],
  },
) =>
  z.object({
    key: keyZ,
    name: z.string(),
    type: schemas.type,
    internal: z.boolean().optional(),
    config: schemas.config,
    status: statusZ(schemas.statusData).optional().nullable(),
    snapshot: z.boolean().optional(),
  });

export interface Payload<S extends Schemas = Schemas> {
  key: Key;
  name: string;
  type: z.infer<S["type"]>;
  config: z.infer<S["config"]>;
  status?: Status<S["statusData"]>;
  snapshot?: boolean;
  internal?: boolean;
}

export const newZ = <S extends Schemas = Schemas>(schemas?: S) =>
  taskZ(schemas)
    .omit({ key: true, status: true })
    .extend({
      key: keyZ.transform((k) => k.toString()).optional(),
      config: schemas?.config ?? record.nullishToEmpty(),
      status: newStatusZ(schemas?.statusData ?? z.unknown())
        .optional()
        .nullable(),
    });

export interface New<S extends Schemas = Schemas> {
  key?: Key;
  name: string;
  type: z.infer<S["type"]>;
  config: z.infer<S["config"]>;
  status?: NewStatus<S["statusData"]>;
}

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

export interface StateObservable<
  StatusData extends z.ZodType,
> extends observe.ObservableAsyncCloseable<Status<StatusData>> {}

export interface CommandObservable extends observe.ObservableAsyncCloseable<Command> {}

export const rackKey = (key: Key): RackKey => Number(BigInt(key) >> 32n);

export const newKey = (rackKey: RackKey, taskKey: number = 0): Key =>
  ((BigInt(rackKey) << 32n) + BigInt(taskKey)).toString();
