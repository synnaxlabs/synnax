// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { type Optional, type primitive } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { Query } from "@/query";
import { Sync } from "@/query/sync";

export const useCalculationStatusSynchronizer = (
  onStatusChange: (status: channel.CalculationStatus) => void,
): void =>
  Sync.useListener({
    channel: channel.CALCULATION_STATUS_CHANNEL_NAME,
    onChange: Sync.parsedHandler(channel.calculationStatusZ, async (args) => {
      onStatusChange(args.changed);
    }),
  });

export const formSchema = channel.newZ
  .extend({
    name: z.string().min(1, "Name must not be empty"),
    dataType: DataType.z.transform((v) => v.toString()),
  })
  .refine(
    (v) => !v.isIndex || DataType.z.parse(v.dataType).equals(DataType.TIMESTAMP),
    {
      message: "Index channel must have data type TIMESTAMP",
      path: ["dataType"],
    },
  )
  .refine((v) => v.isIndex || v.index !== 0 || v.virtual, {
    message: "Data channel must have an index",
    path: ["index"],
  })
  .refine((v) => v.virtual || !DataType.z.parse(v.dataType).isVariable, {
    message: "Persisted channels must have a fixed-size data type",
    path: ["dataType"],
  });

export const calculatedFormSchema = formSchema
  .extend({
    expression: z
      .string()
      .min(1, "Expression must not be empty")
      .refine((v) => v.includes("return"), {
        message: "Expression must contain a return statement",
      }),
  })
  .refine((v) => v.requires?.length > 0, {
    message: "Expression must use at least one synnax channel",
    path: ["requires"],
  });

const channelToFormValues = (ch: channel.Channel) => ({
  ...ch.payload,
  dataType: ch.dataType.toString(),
});

export interface QueryParams extends Query.Params {
  key?: channel.Key;
}

interface UseFormArgs<Z extends z.ZodObject>
  extends Optional<
    Pick<Query.UseFormArgs<QueryParams, Z>, "initialValues" | "params" | "afterUpdate">,
    "initialValues"
  > {}

export const ZERO_FORM_VALUES: z.infer<
  typeof formSchema | typeof calculatedFormSchema
> = {
  key: 0,
  name: "",
  index: 0,
  dataType: DataType.FLOAT32.toString(),
  internal: false,
  isIndex: false,
  leaseholder: 0,
  virtual: false,
  expression: "",
  requires: [],
};

const retrieve = async ({
  client,
  params: { key },
}: Query.RetrieveArgs<QueryParams>) => {
  if (key == null) return null;
  return channelToFormValues(await client.channels.retrieve(key));
};

const update = async ({
  client,
  values,
  onChange,
}: Query.UpdateArgs<QueryParams, typeof formSchema | typeof calculatedFormSchema>) => {
  const ch = await client.channels.create(values);
  onChange(channelToFormValues(ch));
};

export const useForm = (args: UseFormArgs<typeof formSchema>) =>
  Query.useForm<QueryParams, typeof formSchema>({
    name: "Channel",
    schema: formSchema,
    initialValues: { ...ZERO_FORM_VALUES },
    ...args,
    retrieve,
    update,
    listeners: [],
  });

export const useCalculatedForm = (args: UseFormArgs<typeof calculatedFormSchema>) =>
  Query.useForm<QueryParams, typeof calculatedFormSchema>({
    name: "CalculatedChannel",
    schema: calculatedFormSchema,
    initialValues: { ...ZERO_FORM_VALUES },
    ...args,
    retrieve,
    update,
    listeners: [],
  });
