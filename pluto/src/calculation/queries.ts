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
 * Answers with how the channel of the query is written: hidden when a calculation that
 * the reader cannot retrieve writes it, and otherwise with the calculation that writes
 * it, or none. The lookups of a tick share one request.
 */
export const { useResult: useResultOutput } = Flux.createRetrieve<
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
  retryUnreachable: true,
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

/**
 * Answers with the status of the calculation that writes the channel of the query, or
 * null when no calculation writes it or the calculation has no status. The lookups of
 * a tick share one request for the outputs and one for the statuses. Compared by
 * variant, message, and description, so a heartbeat that changes only the time is
 * silenced.
 */
export const { useResult: useResultStatus } = Flux.createRetrieve<
  RetrieveOutputQuery,
  calculation.Status | null
>({
  name: "calculation status",
  retrieve: async ({ client, query: { channel } }) =>
    await client.calculations.retrieveStatus(channel),
  onChange: ({ client, query: { channel } }, handler) => {
    const notify = () => handler(client.calculations.getCachedStatus(channel) ?? null);
    const unsubscribeOutputs = client.calculations.outputs.subscribe(notify, channel);
    const unsubscribeStatuses = client.statuses.store.subscribe((event) => {
      const output = client.calculations.outputs.get(channel);
      const calc = output == null ? null : calculation.keyOf(output);
      if (
        calc != null &&
        client.statuses.ownerKeys(event, "calculation")?.includes(calc)
      )
        notify();
    });
    return () => {
      unsubscribeOutputs();
      unsubscribeStatuses();
    };
  },
  getCached: ({ client, query: { channel } }) =>
    client.calculations.getCachedStatus(channel),
  equal: (a, b) =>
    a?.variant === b?.variant &&
    a?.message === b?.message &&
    a?.description === b?.description,
  retryUnreachable: true,
});
