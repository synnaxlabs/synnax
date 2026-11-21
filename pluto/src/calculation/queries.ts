// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, type channel, type Synnax as Client } from "@synnaxlabs/client";
import { useSyncExternalStore } from "react";

import { Flux } from "@/flux";
import { Synnax } from "@/synnax";

export type RetrieveOutputQuery = {
  /** The key of the channel. */
  channel: channel.Key;
};

/**
 * Answers with the calculation that writes the channel of the query, or with none. The
 * lookups of a tick share one request.
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
});

/**
 * @returns whether a calculation writes the channel. False until the answer loads, and
 * for the key 0, which names no channel.
 */
export const useIsOutput = (channel: channel.Key): boolean =>
  useResultOutput(channel === 0 ? null : { channel }).data?.calculation != null;

// The status of the calculation that writes the channel, null when no calculation
// writes it or the calculation has none, and undefined until the cache can tell.
const cachedStatusOf = (
  client: Client,
  channel: channel.Key,
): calculation.Status | null | undefined => {
  const output = client.calculations.outputs.get(channel);
  if (output == null) return undefined;
  if (output.calculation == null) return null;
  const st = client.statuses.ownedBy(calculation.resourceID(output.calculation));
  if (st == null) return undefined;
  return calculation.statusZ.parse(st);
};

/**
 * Answers with the status of the calculation that writes the channel of the query, or
 * null when no calculation writes it or the calculation has no status. The lookups of
 * a tick share one request for the outputs and one for the statuses. Compared by
 * variant and message, so a heartbeat that changes neither is silenced.
 */
export const { useResult: useResultStatus } = Flux.createRetrieve<
  RetrieveOutputQuery,
  calculation.Status | null
>({
  name: "calculation status",
  retrieve: async ({ client, query: { channel } }) => {
    const [output] = await client.calculations.outputs.retrieve([channel]);
    if (output.calculation == null) return null;
    const owner = calculation.resourceID(output.calculation);
    const [st] = await client.statuses.retrieveOwned([owner]);
    return st == null ? null : calculation.statusZ.parse(st);
  },
  onChange: ({ client, query: { channel } }, handler) => {
    const notify = () => handler(cachedStatusOf(client, channel) ?? null);
    const unsubscribeOutputs = client.calculations.outputs.subscribe(notify, channel);
    const unsubscribeStatuses = client.statuses.store.subscribe((event) => {
      const calc = client.calculations.outputs.get(channel)?.calculation;
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
  getCached: ({ client, query: { channel } }) => cachedStatusOf(client, channel),
  equal: (a, b) => a?.variant === b?.variant && a?.message === b?.message,
});

type Outputs = calculation.Client["outputs"];

// version is the version of outputs that the filter reads, so a new answer makes a new
// filter.
const createNonOutputFilter =
  (outputs: Outputs | undefined, _version: number) =>
  (ch: channel.Channel): boolean => {
    if (outputs == null) return false;
    const output = outputs.get(ch.key);
    if (output == null) {
      // The lookups of a tick share one request.
      outputs.retrieve([ch.key]).catch(console.error);
      return false;
    }
    return output.calculation == null;
  };

/**
 * @returns a filter that keeps the channels that no calculation writes. It looks up
 * each channel it reads, and leaves a channel out until its answer loads. Each new
 * answer gives a new filter, so a list that holds it filters again.
 */
export const useNonOutputFilter = (): ((ch: channel.Channel) => boolean) => {
  const outputs = Synnax.use()?.calculations.outputs;
  const version = useSyncExternalStore(
    (onChange) => outputs?.subscribe(() => onChange()) ?? (() => {}),
    () => outputs?.version() ?? 0,
  );
  return createNonOutputFilter(outputs, version);
};
