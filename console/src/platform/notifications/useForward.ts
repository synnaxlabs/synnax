// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type status, type Synnax as Client } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { useEffect } from "react";

/**
 * What a status is for, which decides if it notifies. A reply answers a command, an
 * alarm is a status that no feature owns, and a state is the condition of a thing.
 */
export type Kind = "reply" | "alarm" | "state";

/** Parameters for {@link Rule.inUse}. */
export interface InUseParams {
  client: Client;
  /** The key of the thing, which is the part of the status key after its type. */
  key: string;
}

/** How the statuses on the keys that one feature owns notify. */
export interface Rule {
  /** @returns the kind of a status on one of the feature's keys. */
  kind: (status: status.Status) => Kind;
  /**
   * @returns whether the thing that a state is about is in use. Runs only when a
   * problem starts on the key. Leave it out when the thing is always in use.
   */
  inUse?: (params: InUseParams) => Promise<boolean>;
  /** Whether a warning notifies only if its problem is still open after 5 s. */
  warningsDelayed?: boolean;
}

/** The rule of each feature that owns status keys, by the ontology type of its keys. */
export type Rules = Partial<Record<ontology.ResourceType, Rule>>;

const WARNING_DELAY = TimeSpan.seconds(5);
const PROBLEM_VARIANTS: status.Variant[] = ["warning", "error"];
const ROUTINE_VARIANTS: status.Variant[] = ["success", "loading", "disabled"];

interface Owner {
  rule: Rule;
  key: string;
}

const getOwner = (rules: Rules, key: status.Key): Owner | null => {
  const separator = key.indexOf(":");
  if (separator < 0) return null;
  const type = ontology.resourceTypeZ.safeParse(key.slice(0, separator));
  const rule = type.success ? rules[type.data] : undefined;
  if (rule == null) return null;
  return { rule, key: key.slice(separator + 1) };
};

interface Problem {
  status: status.Status;
  /** Undefined until the check resolves. */
  inUse?: boolean;
  /** True until the delay of a rule with delayed warnings ends. */
  delayed: boolean;
  shown: boolean;
  timer?: ReturnType<typeof setTimeout>;
}

interface ForwarderParams {
  client: Client;
  rules: Rules;
  add: Status.Adder;
}

interface Forwarder {
  write: (status: status.Status) => void;
  /** Closes the problem of a deleted status. */
  remove: (key: status.Key) => void;
  stop: () => void;
}

const createForwarder = ({ client, rules, add }: ForwarderParams): Forwarder => {
  const problems = new Map<status.Key, Problem>();

  const show = (problem: Problem): void => {
    if (problem.inUse !== true) return;
    const held = problem.delayed && problem.status.variant === "warning";
    if (!problem.shown && held) return;
    problem.shown = true;
    add(problem.status);
  };

  const close = (key: status.Key): void => {
    clearTimeout(problems.get(key)?.timer);
    problems.delete(key);
  };

  // A newer status replaces the notification on its key, so an older one must not show.
  const retire = (key: status.Key): void => {
    if (problems.get(key)?.shown === false) close(key);
  };

  const open = (stat: status.Status, { rule, key }: Owner): void => {
    const problem: Problem = {
      status: stat,
      delayed: rule.warningsDelayed === true,
      shown: false,
    };
    problems.set(stat.key, problem);
    if (problem.delayed)
      problem.timer = setTimeout(() => {
        problem.delayed = false;
        if (!problem.shown) show(problem);
      }, WARNING_DELAY.milliseconds);
    if (rule.inUse == null) {
      problem.inUse = true;
      show(problem);
      return;
    }
    void rule.inUse({ client, key }).then(
      (inUse) => {
        if (problems.get(stat.key) !== problem) return;
        problem.inUse = inUse;
        show(problem);
      },
      // The check only filters noise, so a status whose check fails still shows.
      (exc: unknown) => {
        if (problems.get(stat.key) !== problem) return;
        console.error(`failed to check if the thing of ${stat.key} is in use`, exc);
        problem.inUse = true;
        show(problem);
      },
    );
  };

  const write = (stat: status.Status): void => {
    const owner = getOwner(rules, stat.key);
    const kind = owner?.rule.kind(stat) ?? "alarm";
    if (kind !== "state") retire(stat.key);
    if (owner == null || kind === "alarm") {
      add(stat);
      return;
    }
    const bad = PROBLEM_VARIANTS.includes(stat.variant);
    if (!bad) close(stat.key);
    if (kind === "reply") {
      if (!ROUTINE_VARIANTS.includes(stat.variant)) add(stat);
      return;
    }
    if (!bad) return;
    const problem = problems.get(stat.key);
    if (problem == null) {
      open(stat, owner);
      return;
    }
    problem.status = stat;
    show(problem);
  };

  const stop = (): void => {
    problems.forEach(({ timer }) => clearTimeout(timer));
    problems.clear();
  };

  return { write, remove: close, stop };
};

/**
 * Notifies the status writes that the Core streams, by kind. A reply notifies at once
 * unless it is a success, loading, or disabled status. An alarm notifies at once. A
 * state notifies only when it is a warning or an error and its thing is in use. A
 * delayed warning notifies only if its key is still bad 5 s after the problem starts.
 * Mount it once, in the main window.
 * @param rules - The rules of the features that own status keys.
 */
export const useForward = (rules: Rules): void => {
  const client = Synnax.use();
  const add = Status.useAdder();
  useEffect(() => {
    if (client == null) return;
    const forwarder = createForwarder({ client, rules, add });
    const removeWriteListener = client.statuses.onWrite(forwarder.write);
    const removeDeleteListener = client.statuses.onDelete(forwarder.remove);
    return () => {
      removeWriteListener();
      removeDeleteListener();
      forwarder.stop();
    };
  }, [client, rules, add]);
};
