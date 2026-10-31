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
   * @returns whether the thing that a state is about is in use. Runs only when a new
   * problem starts on the key. Leave it out when the thing is always in use.
   */
  inUse?: (params: InUseParams) => Promise<boolean>;
  /** Whether a warning notifies only if its problem is still open after 5 s. */
  warningsDelayed?: boolean;
  /**
   * @returns the key of the status whose open problem covers a problem on this key,
   * such as the rack of a task. A problem that starts inside the covering problem does
   * not notify until a write of it arrives after the covering problem closes.
   */
  coveredBy?: (status: status.Status) => status.Key | null;
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

// Without a since, every problem on a key has one identity.
const identify = ({ since }: status.Status): string =>
  since?.valueOf().toString() ?? "";

interface Problem {
  /** The since of the problem. */
  identity: string;
  /** The latest bad status of the problem. */
  status: status.Status;
  rule?: Rule;
  /** Whether the latest write on the key is bad. */
  open: boolean;
  /** Whether the latest write arrived inside the covering problem. */
  covered: boolean;
  decision: "waiting" | "shown" | "covered" | "hidden";
  /** Undefined until the check resolves. */
  inUse?: boolean;
  /** True while a warning waits out the delay of its rule. */
  delayed: boolean;
  timer?: ReturnType<typeof setTimeout>;
}

interface ForwarderParams {
  client: Client;
  rules: Rules;
  add: Status.Adder;
}

interface Forwarder {
  /** Holds each write until {@link load} runs. */
  write: (status: status.Status) => void;
  /** Holds each delete until {@link load} runs. */
  remove: (key: status.Key) => void;
  /** Takes the problems that are open at load as seen, then handles held events. */
  load: (open: status.Status[]) => void;
  stop: () => void;
}

const createForwarder = ({ client, rules, add }: ForwarderParams): Forwarder => {
  const problems = new Map<status.Key, Problem>();
  let held: Array<() => void> | null = [];
  let stopped = false;

  const isCovered = (rule: Rule | undefined, stat: status.Status): boolean => {
    const coveringKey = rule?.coveredBy?.(stat);
    if (coveringKey == null) return false;
    const covering = problems.get(coveringKey);
    if (covering?.open !== true) return false;
    const { since } = stat;
    const coveringSince = covering.status.since;
    return since == null || coveringSince == null || since.afterEq(coveringSince);
  };

  const notify = ({ status: stat, identity }: Problem): void =>
    add({ ...stat, problem: identity });

  const show = (problem: Problem): void => {
    if (problem.decision !== "waiting" || !problem.open || problem.inUse !== true)
      return;
    if (problem.delayed && problem.status.variant === "warning") return;
    clearTimeout(problem.timer);
    problem.decision = problem.covered ? "covered" : "shown";
    if (problem.decision === "shown") notify(problem);
  };

  const wait = (problem: Problem): void => {
    clearTimeout(problem.timer);
    problem.delayed = true;
    problem.timer = setTimeout(() => {
      problem.delayed = false;
      show(problem);
    }, WARNING_DELAY.milliseconds);
  };

  const start = (stat: status.Status, owner: Owner | null, kind: Kind): void => {
    clearTimeout(problems.get(stat.key)?.timer);
    const problem: Problem = {
      identity: identify(stat),
      status: stat,
      rule: owner?.rule,
      open: true,
      covered: isCovered(owner?.rule, stat),
      decision: "waiting",
      delayed: false,
    };
    problems.set(stat.key, problem);
    if (owner == null || kind !== "state") {
      problem.decision = "shown";
      notify(problem);
      return;
    }
    const { rule, key } = owner;
    if (rule.warningsDelayed === true) wait(problem);
    if (rule.inUse == null) {
      problem.inUse = true;
      show(problem);
      return;
    }
    void rule.inUse({ client, key }).then(
      (inUse) => {
        if (problems.get(stat.key) !== problem) return;
        problem.inUse = inUse;
        if (!inUse) problem.decision = "hidden";
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

  const repeat = (problem: Problem, stat: status.Status): void => {
    const reopened = !problem.open;
    problem.status = stat;
    problem.open = true;
    problem.covered = isCovered(problem.rule, stat);
    if (problem.decision === "shown") {
      notify(problem);
      return;
    }
    if (problem.decision === "hidden") return;
    const delays = problem.rule?.warningsDelayed === true;
    if (problem.decision === "covered") {
      if (problem.covered) return;
      problem.decision = "waiting";
      if (delays) wait(problem);
    } else if (reopened && delays) wait(problem);
    show(problem);
  };

  const drop = (key: status.Key): void => {
    clearTimeout(problems.get(key)?.timer);
    problems.delete(key);
  };

  // A problem that waits to notify must not notify after a newer status on its key.
  const retire = (key: status.Key): void => {
    if (problems.get(key)?.decision === "waiting") drop(key);
  };

  const handle = (stat: status.Status): void => {
    const owner = getOwner(rules, stat.key);
    const kind = owner?.rule.kind(stat) ?? "alarm";
    if (kind !== "state") retire(stat.key);
    const bad = PROBLEM_VARIANTS.includes(stat.variant);
    const problem = problems.get(stat.key);
    if (!bad && problem != null) {
      problem.open = false;
      if (problem.decision === "waiting") clearTimeout(problem.timer);
    }
    // A reply answers one command, so it is not part of a problem.
    if (kind === "reply") {
      if (!ROUTINE_VARIANTS.includes(stat.variant)) add(stat);
      return;
    }
    if (!bad) {
      if (kind === "alarm") add(stat);
      return;
    }
    if (problem?.identity === identify(stat)) repeat(problem, stat);
    else start(stat, owner, kind);
  };

  const run = (event: () => void): void => {
    if (held == null) event();
    else held.push(event);
  };

  const write = (stat: status.Status): void => run(() => handle(stat));

  const remove = (key: status.Key): void => run(() => drop(key));

  const load = (open: status.Status[]): void => {
    if (stopped || held == null) return;
    open.forEach((stat) =>
      problems.set(stat.key, {
        identity: identify(stat),
        status: stat,
        open: true,
        covered: false,
        decision: "hidden",
        delayed: false,
      }),
    );
    const events = held;
    held = null;
    events.forEach((event) => event());
  };

  const stop = (): void => {
    stopped = true;
    problems.forEach(({ timer }) => clearTimeout(timer));
    problems.clear();
  };

  return { write, remove, load, stop };
};

/**
 * Notifies the status writes that the Core streams, by kind. A reply notifies at once
 * unless it is a success, loading, or disabled status. An alarm notifies at once. A
 * state notifies only when it is a warning or an error and its thing is in use. A
 * delayed warning notifies only if its key is still bad 5 s after the problem starts.
 * A problem, which is a key and a since, notifies once. Later writes of the problem add
 * to its count, and the problems open at load never notify. Mount it once, in the main
 * window.
 * @param rules - The rules of the features that own status keys.
 */
export const useForward = (rules: Rules): void => {
  const client = Synnax.use();
  const add = Status.useAdder();
  const handleError = Status.useErrorHandler();
  useEffect(() => {
    if (client == null) return;
    const forwarder = createForwarder({ client, rules, add });
    const removeWriteListener = client.statuses.onWrite(forwarder.write);
    const removeDeleteListener = client.statuses.onDelete(forwarder.remove);
    client.statuses
      .retrieve({ variants: PROBLEM_VARIANTS })
      .then(forwarder.load, (exc: unknown) => {
        handleError(exc, "Failed to retrieve open problems");
        forwarder.load([]);
      });
    return () => {
      removeWriteListener();
      removeDeleteListener();
      forwarder.stop();
    };
  }, [client, rules, add, handleError]);
};
