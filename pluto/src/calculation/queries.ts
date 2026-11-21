// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  calculation,
  type channel,
  isConnectionError,
  status,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { type destructor } from "@synnaxlabs/x";
import { useState, useSyncExternalStore } from "react";

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
  retryUnreachable: true,
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
  retryUnreachable: true,
});

type Outputs = calculation.Client["outputs"];

const LOOKUP_FAILED = "Failed to find calculated channels";

const LOOKING_UP = status.create({
  variant: "loading",
  message: "Finding calculated channels",
});

interface Lookups {
  /**
   * Looks up whether a calculation writes the channel, unless a lookup of it runs. A
   * tick shares one request.
   */
  lookUp: (outputs: Outputs, report: Status.ErrorHandler, key: channel.Key) => void;
  subscribe: (onChange: () => void) => destructor.Destructor;
  /** Loading while a lookup runs, else the failure of the last one, if it failed. */
  status: () => status.Status | undefined;
}

// Resolves to the rejection of the promise, or to null when it fulfills.
const rejectionOf = async (promise: Promise<unknown>): Promise<unknown> =>
  await promise.then(
    () => null,
    (cause: unknown) => cause,
  );

// The filter looks up while the list renders, so each change lands after the render.
const createLookups = (): Lookups => {
  const queued = new Set<channel.Key>();
  const looking = new Set<channel.Key>();
  const listeners = new Set<() => void>();
  let running = 0;
  let failure: status.Status | undefined;
  const notify = () => listeners.forEach((onChange) => onChange());
  const run = async (outputs: Outputs, report: Status.ErrorHandler) => {
    const keys = [...queued];
    queued.clear();
    running++;
    notify();
    const fetch = async () => await outputs.retrieve(keys);
    let error = await rejectionOf(fetch());
    if (isConnectionError(error))
      error = await rejectionOf(
        Flux.retryUnreachableFetch("calculation outputs", fetch),
      );
    failure = undefined;
    if (error != null) {
      report(error, LOOKUP_FAILED);
      failure = status.fromException(error, LOOKUP_FAILED);
    }
    keys.forEach((key) => looking.delete(key));
    running--;
    notify();
  };
  return {
    lookUp: (outputs, report, key) => {
      if (looking.has(key)) return;
      looking.add(key);
      if (queued.size === 0) queueMicrotask(() => void run(outputs, report));
      queued.add(key);
    },
    subscribe: (onChange) => {
      listeners.add(onChange);
      return () => listeners.delete(onChange);
    },
    status: () => (running > 0 ? LOOKING_UP : failure),
  };
};

// version is the version of outputs that the filter reads, so a new answer makes a new
// filter.
const createNonOutputFilter =
  (
    outputs: Outputs | undefined,
    _version: number,
    lookUp: (key: channel.Key) => void,
  ) =>
  (ch: channel.Channel): boolean => {
    const output = outputs?.get(ch.key);
    if (output == null) {
      lookUp(ch.key);
      return false;
    }
    return output.calculation == null;
  };

/** The return value of {@link useNonOutputFilter}. */
export interface NonOutputFilter {
  /**
   * Keeps the channels that no calculation writes. It looks up each channel it has no
   * answer for, and leaves the channel out until the answer loads. Each new answer
   * gives a new filter, so a list that holds it filters again.
   */
  filter: (ch: channel.Channel) => boolean;
  /**
   * Loading while a lookup runs, and the failure of the last lookup if it failed. A
   * picker shows it in place of the list's status, so a channel left out unanswered
   * never reads as no match. Undefined once every lookup answered.
   */
  status?: status.Status;
}

/**
 * @returns a filter that keeps the channels that no calculation writes, and the status
 * of its lookups. A lookup the Core is unreachable for repeats before it fails, and a
 * failure goes to the status handler.
 */
export const useNonOutputFilter = (): NonOutputFilter => {
  const outputs = Synnax.use()?.calculations.outputs;
  const handleError = Status.useErrorHandler();
  const [lookups] = useState(createLookups);
  const version = useSyncExternalStore(
    (onChange) => outputs?.subscribe(() => onChange()) ?? (() => {}),
    () => outputs?.version() ?? 0,
  );
  const lookupStatus = useSyncExternalStore(lookups.subscribe, lookups.status);
  const lookUp = (key: channel.Key) => {
    if (outputs != null) lookups.lookUp(outputs, handleError, key);
  };
  return {
    filter: createNonOutputFilter(outputs, version, lookUp),
    status: lookupStatus,
  };
};
