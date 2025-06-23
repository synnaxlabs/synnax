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

export const useCalculationStateSynchronizer = (
  onChange: (state: channel.CalculationState) => void,
): void =>
  Query.useParsedListener(
    channel.CALCULATION_STATE_CHANNEL_NAME,
    channel.calculationStateZ,
    onChange,
  );

export const useForm = Query.createForm<channel.Key, typeof channel.channelZ>({
  name: "Channel",
  schema: channel.channelZ,
  queryFn: async ({ client, params: key }) => {
    if (key == null) return null;
    return await client.channels.retrieve(key);
  },
  mutationFn: async ({ client, values }) => await client.channels.create(values),
  listeners: [],
});
