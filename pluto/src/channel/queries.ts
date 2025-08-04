// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { z } from "zod";

import { Flux } from "@/flux";

export const useCalculationStatusSynchronizer = (
  onStatusChange: (status: channel.CalculationStatus) => void,
): void =>
  Flux.useListener({
    channel: channel.CALCULATION_STATUS_CHANNEL_NAME,
    onChange: Flux.parsedHandler(channel.calculationStatusZ, async (args) => {
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
    message: "Expression must use at least one channel",
    path: ["requires"],
  });

const channelToFormValues = (ch: channel.Channel) => ({
  ...ch.payload,
  dataType: ch.dataType.toString(),
});

export interface FluxParams {
  key?: channel.Key;
}

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

const retrieve = async ({ client, params: { key } }: Flux.RetrieveArgs<FluxParams>) => {
  if (key == null) return null;
  return channelToFormValues(await client.channels.retrieve(key));
};

const update = async ({
  client,
  value,
  onChange,
}: Flux.UpdateArgs<
  FluxParams,
  z.infer<typeof formSchema | typeof calculatedFormSchema>
>) => {
  const ch = await client.channels.create(value);
  onChange(channelToFormValues(ch));
};

export const useForm = (args: Flux.UseFormArgs<FluxParams, typeof formSchema>) =>
  Flux.createForm<FluxParams, typeof formSchema>({
    name: "Channel",
    schema: formSchema,
    initialValues: ZERO_FORM_VALUES,
    retrieve,
    update,
    listeners: [],
  })(args);

export const useCalculatedForm = (
  args: Flux.UseFormArgs<FluxParams, typeof calculatedFormSchema>,
) =>
  Flux.createForm<FluxParams, typeof calculatedFormSchema>({
    name: "Calculated Channel",
    schema: calculatedFormSchema,
    initialValues: ZERO_FORM_VALUES,
    retrieve,
    update,
    listeners: [],
  })(args);
