// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";

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

export const useForm = (
  args: Pick<
    Query.UseFormArgs<channel.Key, typeof channel.channelZ>,
    "initialValues" | "params"
  >,
) =>
  Query.useForm<channel.Key, typeof channel.channelZ>({
    name: "Channel",
    schema: channel.channelZ,
    ...args,
    retrieve: async ({ client, params: key }) => {
      if (key == null) return null;
      return await client.channels.retrieve(key);
    },
    update: async ({ client, values }) => await client.channels.create(values),
    listeners: [],
  });
