// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, type channel } from "@synnaxlabs/client";

import { Flux } from "@/flux";

type RetrieveOutputQuery = {
  /** The key of the channel. */
  channel: channel.Key;
};

/**
 * Answers with the calculation that writes the channel of the query, or with none. The
 * lookups of a tick share one request.
 */
const { useResult: useResultOutput } = Flux.createRetrieve<
  RetrieveOutputQuery,
  calculation.Output
>({
  name: "calculation output",
  retrieve: async ({ client, query: { channel } }) => {
    const [output] = await client.calculations.outputs.retrieve([channel]);
    return output;
  },
  onChange: ({ client, query: { channel } }, handler) =>
    client.calculations.outputs.subscribe(
      (event) => handler(event.variant === "set" ? event.value : undefined),
      channel,
    ),
  getCached: ({ client, query: { channel } }) =>
    client.calculations.outputs.get(channel),
});

/**
 * @returns whether a calculation writes the channel, whether or not the reader can
 * retrieve it, or undefined until the answer loads. False for the key 0, which names
 * no channel.
 */
export const useIsOutput = (channel: channel.Key): boolean | undefined => {
  const output = useResultOutput(channel === 0 ? null : { channel }).data;
  if (channel === 0) return false;
  return output == null ? undefined : calculation.isOutput(output);
};
