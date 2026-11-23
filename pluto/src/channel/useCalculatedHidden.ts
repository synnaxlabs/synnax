// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { calculation, type channel, status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import type { destructor } from "@synnaxlabs/x";
import { useEffect, useState, useSyncExternalStore } from "react";

import { Synnax } from "@/synnax";

type Outputs = calculation.Client["outputs"];

const LOOKUP_FAILED = "Failed to load channels";

const LOOKUP_FAILED_REASON =
  "Synnax could not check which channels a calculation writes. Open the picker again " +
  "to retry.";

const LOOKING_UP = status.create({
  variant: "loading",
  message: "Finding calculated channels",
});

const failureOf = (error: unknown): status.Status => {
  const failure = status.fromException(error, LOOKUP_FAILED);
  return { ...failure, description: `${LOOKUP_FAILED_REASON} ${failure.description}` };
};

interface Lookups {
  /**
   * Looks up each of keys that no lookup in flight holds, and that no lookup failed for
   * since the picker last opened or a lookup last answered.
   */
  run: (outputs: Outputs, keys: channel.Key[]) => void;
  /**
   * Sets whether the picker is open, and how a failure is reported. A failure is
   * reported only while the picker is open.
   */
  setOpen: (open: boolean, report?: Report) => void;
  subscribe: (onChange: () => void) => destructor.Destructor;
  /** Loading while a lookup runs, else the failure of the last one, if it failed. */
  status: () => status.Status | undefined;
}

type Report = (failure: status.Status) => void;

// A failure episode, which a lookup that answers ends, is reported once.
const createLookups = (): Lookups => {
  const looking = new Set<channel.Key>();
  const failed = new Set<channel.Key>();
  const listeners = new Set<() => void>();
  let running = 0;
  let failure: status.Status | undefined;
  let reported = false;
  let open = false;
  let report: Report | undefined;
  const notify = () => listeners.forEach((onChange) => onChange());
  const run = async (outputs: Outputs, keys: channel.Key[]): Promise<void> => {
    keys.forEach((key) => looking.add(key));
    running++;
    notify();
    let retry: channel.Key[] = [];
    try {
      await outputs.retrieve(keys);
      // The Core answers again, so the channels that failed are looked up again.
      retry = [...failed].filter((key) => !looking.has(key));
      failed.clear();
      failure = undefined;
      reported = false;
    } catch (e) {
      keys.forEach((key) => failed.add(key));
      failure = failureOf(e);
      if (!reported && open && report != null) {
        report(failure);
        reported = true;
      }
    }
    keys.forEach((key) => looking.delete(key));
    running--;
    notify();
    if (retry.length > 0) await run(outputs, retry);
  };
  return {
    run: (outputs, keys) => {
      const fresh = keys.filter((key) => !looking.has(key) && !failed.has(key));
      if (fresh.length > 0) void run(outputs, fresh);
    },
    setOpen: (next, nextReport) => {
      // Each open looks the failed channels up again.
      if (next && !open) failed.clear();
      open = next;
      report = nextReport;
    },
    subscribe: (onChange) => {
      listeners.add(onChange);
      return () => listeners.delete(onChange);
    },
    status: () => (running > 0 ? LOOKING_UP : failure),
  };
};

// version is the version of outputs that the read sees, so a new answer reads again.
const answersOf = (outputs: Outputs, keys: channel.Key[], _version: number) => ({
  unanswered: keys.filter((key) => outputs.get(key) == null),
  kept: keys.filter((key) => {
    const output = outputs.get(key);
    return output != null && !calculation.isOutput(output);
  }),
});

/** The return value of {@link useCalculatedHidden}. */
export interface CalculatedHidden {
  /** The keys that no calculation writes, once each has an answer. */
  data: channel.Key[];
  /**
   * Loading while a lookup runs, and the failure of the last lookup if it failed, while
   * a key has no answer. Undefined once every key has one.
   */
  status?: status.Status;
}

/**
 * Leaves out of keys each channel that a calculation writes, and each channel whose
 * answer has not loaded. It looks up the channels without an answer, and again each
 * time the picker opens. A failed lookup is reported once while the picker is open,
 * until a lookup answers. With hidden false, it keeps every key.
 */
export const useCalculatedHidden = (
  keys: channel.Key[],
  open: boolean,
  hidden: boolean,
): CalculatedHidden => {
  const client = Synnax.use();
  const outputs = hidden ? client?.calculations.outputs : undefined;
  const report = Status.useAdder();
  const [lookups] = useState(createLookups);
  const version = useSyncExternalStore(
    (onChange) => outputs?.subscribe(() => onChange()) ?? (() => {}),
    () => outputs?.version() ?? 0,
  );
  const lookupStatus = useSyncExternalStore(lookups.subscribe, lookups.status);
  // A lookup can fail after the picker closes or unmounts, so it reads whether the
  // picker is open then.
  useEffect(() => {
    lookups.setOpen(open, report);
    return () => lookups.setOpen(false);
  }, [lookups, open, report]);
  const answers = outputs == null ? undefined : answersOf(outputs, keys, version);
  // Keyed on the keys without an answer, so an answer for another channel runs no
  // lookup.
  const pending = answers?.unanswered.join(",") ?? "";
  useEffect(() => {
    if (outputs != null && answers != null && answers.unanswered.length > 0)
      lookups.run(outputs, answers.unanswered);
  }, [lookups, outputs, pending, open]);
  if (answers == null) return { data: keys };
  return {
    data: answers.kept,
    status: answers.unanswered.length === 0 ? undefined : (lookupStatus ?? LOOKING_UP),
  };
};
