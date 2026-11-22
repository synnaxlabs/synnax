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
  status,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { type destructor } from "@synnaxlabs/x";
import { useEffect, useState, useSyncExternalStore } from "react";

import { Flux } from "@/flux";
import { Synnax } from "@/synnax";

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

// The status of the calculation that writes the channel, null when no calculation
// writes it or the calculation has none, and undefined until the cache can tell.
const cachedStatusOf = (
  client: Client,
  channel: channel.Key,
): calculation.Status | null | undefined => {
  const output = client.calculations.outputs.get(channel);
  if (output == null) return undefined;
  if (output.hidden || output.calculation == null) return null;
  const st = client.statuses.ownedBy(calculation.resourceID(output.calculation));
  if (st == null) return undefined;
  return calculation.statusZ.parse(st);
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
  retrieve: async ({ client, query: { channel } }) => {
    const [output] = await client.calculations.outputs.retrieve([channel]);
    if (output.hidden || output.calculation == null) return null;
    const owner = calculation.resourceID(output.calculation);
    const [st] = await client.statuses.retrieveOwned([owner]);
    return st == null ? null : calculation.statusZ.parse(st);
  },
  onChange: ({ client, query: { channel } }, handler) => {
    const notify = () => handler(cachedStatusOf(client, channel) ?? null);
    const unsubscribeOutputs = client.calculations.outputs.subscribe(notify, channel);
    const unsubscribeStatuses = client.statuses.store.subscribe((event) => {
      const output = client.calculations.outputs.get(channel);
      const calc = output == null || output.hidden ? null : output.calculation;
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
  equal: (a, b) =>
    a?.variant === b?.variant &&
    a?.message === b?.message &&
    a?.description === b?.description,
  retryUnreachable: true,
});

type Outputs = calculation.Client["outputs"];

const LOOKUP_FAILED = "Failed to load channels";

const LOOKUP_FAILED_REASON =
  "Synnax could not check which channels a calculation writes. Check the connection " +
  "to the Core, then open the picker again.";

const LOOKING_UP = status.create({
  variant: "loading",
  message: "Finding calculated channels",
});

// Reports a failed lookup.
type Report = (failure: status.Status) => void;

interface Lookups {
  /**
   * Records that the filter read the channel, and looks it up when it has no answer.
   * The reads of a tick form one pass over the list, and its lookups one request.
   */
  read: (outputs: Outputs, report: Report, key: channel.Key, answered: boolean) => void;
  /** Sets whether the picker's list is open, which a failure is reported only while. */
  setOpen: (open: boolean) => void;
  subscribe: (onChange: () => void) => destructor.Destructor;
  /**
   * Loading while a lookup runs, else the failure of the last one. Undefined when the
   * last pass read no channel without an answer.
   */
  status: () => status.Status | undefined;
}

const failureOf = (error: unknown): status.Status => {
  const failure = status.fromException(error, LOOKUP_FAILED);
  return { ...failure, description: `${LOOKUP_FAILED_REASON} ${failure.description}` };
};

// The filter reads while the list renders, so each change lands after the render.
const createLookups = (): Lookups => {
  const queued = new Set<channel.Key>();
  const looking = new Set<channel.Key>();
  const listeners = new Set<() => void>();
  let running = 0;
  let failure: status.Status | undefined;
  // A failure episode, which a lookup that answers ends, is reported once.
  let reported = false;
  let open = false;
  let pass: Set<channel.Key> | undefined;
  let unanswered = new Set<channel.Key>();
  const notify = () => listeners.forEach((onChange) => onChange());
  const run = async (outputs: Outputs, report: Report) => {
    const keys = [...queued];
    queued.clear();
    running++;
    notify();
    try {
      await outputs.retrieve(keys);
      reported = false;
    } catch (e) {
      failure = failureOf(e);
      if (!reported && open) {
        report(failure);
        reported = true;
      }
    }
    keys.forEach((key) => looking.delete(key));
    running--;
    notify();
  };
  const endPass = () => {
    unanswered = pass ?? new Set();
    pass = undefined;
    notify();
  };
  return {
    read: (outputs, report, key, answered) => {
      if (pass == null) {
        pass = new Set();
        queueMicrotask(endPass);
      }
      if (answered) return;
      pass.add(key);
      if (looking.has(key)) return;
      looking.add(key);
      if (queued.size === 0) queueMicrotask(() => void run(outputs, report));
      queued.add(key);
    },
    setOpen: (next) => {
      open = next;
    },
    subscribe: (onChange) => {
      listeners.add(onChange);
      return () => listeners.delete(onChange);
    },
    status: () => {
      if (unanswered.size === 0) return undefined;
      return running > 0 ? LOOKING_UP : (failure ?? LOOKING_UP);
    },
  };
};

// version is the version of outputs that the filter reads, so a new answer makes a new
// filter.
const createNonOutputFilter =
  (
    outputs: Outputs | undefined,
    _version: number,
    read: (key: channel.Key, answered: boolean) => void,
  ) =>
  (ch: channel.Channel): boolean => {
    const output = outputs?.get(ch.key);
    read(ch.key, output != null);
    return output != null && !calculation.isOutput(output);
  };

/** The return value of {@link useNonOutputFilter}. */
export interface NonOutputFilter {
  /**
   * Keeps the channels that no calculation writes. It looks up each channel it has no
   * answer for, and leaves the channel out until the answer loads. Each new answer
   * gives a new filter, so a list that holds it filters again, and a channel still
   * without an answer joins the next lookup.
   */
  filter: (ch: channel.Channel) => boolean;
  /**
   * Loading while a lookup runs, and the failure of the last lookup if it failed,
   * while the list holds a channel without an answer. A picker shows it in place of
   * the list's status. Undefined once every channel the list holds has an answer.
   */
  status?: status.Status;
}

/** Arguments for {@link useNonOutputFilter}. */
export interface UseNonOutputFilterArgs {
  /**
   * Whether the list of the picker that uses the filter is open. A failed lookup is
   * reported only while it is, once until a lookup answers.
   */
  open: boolean;
}

/**
 * @returns a filter that keeps the channels that no calculation writes, and the status
 * of its lookups.
 */
export const useNonOutputFilter = ({
  open,
}: UseNonOutputFilterArgs): NonOutputFilter => {
  const outputs = Synnax.use()?.calculations.outputs;
  const addStatus = Status.useAdder();
  const [lookups] = useState(createLookups);
  const version = useSyncExternalStore(
    (onChange) => outputs?.subscribe(() => onChange()) ?? (() => {}),
    () => outputs?.version() ?? 0,
  );
  const lookupStatus = useSyncExternalStore(lookups.subscribe, lookups.status);
  // A lookup can fail after the list closes, so it reads whether the list is open then.
  useEffect(() => lookups.setOpen(open), [lookups, open]);
  const read = (key: channel.Key, answered: boolean) => {
    if (outputs != null) lookups.read(outputs, addStatus, key, answered);
  };
  return {
    filter: createNonOutputFilter(outputs, version, read),
    status: lookupStatus,
  };
};
