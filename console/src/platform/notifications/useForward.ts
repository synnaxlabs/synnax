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
import { destructor, TimeSpan } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

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
   * @returns whether the thing that a state is about is in use. Runs when a new problem
   * starts on the key and on each change from {@link onUseChange}. Leave it out when
   * the thing is always in use.
   */
  inUse?: (params: InUseParams) => Promise<boolean>;
  /**
   * Calls the handler on each change that can change whether a thing is in use, such as
   * a device that becomes configured. A problem that starts while its thing is not in
   * use never notifies, but it joins the open problems when its thing is in use.
   * @returns a destructor that stops the calls.
   */
  onUseChange?: (client: Client, handler: () => void) => destructor.Destructor;
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

/**
 * A warning or error that is open, from a thing in use or a status that no feature
 * owns. A reply is never a problem.
 */
export interface Problem {
  /** The latest bad status of the problem. */
  status: status.Status;
  /** How many bad statuses of the problem the Console got. */
  count: number;
  /** The key of the status whose problem covers this one, such as the task's rack. */
  coveredBy: status.Key | null;
}

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
// Writers stamp times on their own clocks, so a later write can have an earlier time.
const isSameWrite = (a: status.Status, b: status.Status): boolean =>
  a.time.equals(b.time) && a.variant === b.variant && a.message === b.message;

const identify = ({ since }: status.Status): string =>
  since?.valueOf().toString() ?? "";

interface Entry {
  /** The since of the problem. */
  identity: string;
  /** The latest bad status of the problem. */
  status: status.Status;
  rule?: Rule;
  /** Whether the latest write on the key is bad. */
  open: boolean;
  /** Whether the latest write arrived inside the covering problem. */
  covered: boolean;
  /**
   * Superseded lasts from a newer status on the key, such as a reply, to the next
   * write.
   */
  decision: "waiting" | "superseded" | "shown" | "covered" | "hidden";
  /** Whether the thing was in use when the problem started, by the first check. */
  usedAtStart?: boolean;
  /** Whether the thing is in use by the latest check. Undefined until a check ends. */
  inUse?: boolean;
  /** True while a warning waits out the delay of its rule. */
  delayed: boolean;
  timer?: ReturnType<typeof setTimeout>;
  count: number;
  /** Counts the use checks, since checks can end out of order. */
  checks: number;
  /** Whether the use of the thing changed while the problem was closed. */
  useStale?: boolean;
  /** The status that the load returned, until the next write on the key. */
  loaded?: status.Status;
}

interface ForwarderParams {
  client: Client;
  rules: Rules;
  add: Status.Adder;
  setProblems: (problems: Problem[]) => void;
}

interface Forwarder {
  /** Holds each write until {@link load} runs. */
  write: (status: status.Status) => void;
  /** Holds each delete until {@link load} runs. */
  remove: (key: status.Key) => void;
  /** Takes the problems that are open at load as seen, then handles held events. */
  load: (open: status.Status[]) => void;
  /**
   * Checks again if the thing of each problem that the rule checked is in use. Calls
   * before the next microtask share one check, since one fetch can set many things.
   */
  recheck: (rule: Rule) => void;
  stop: () => void;
}

const createForwarder = ({
  client,
  rules,
  add,
  setProblems,
}: ForwarderParams): Forwarder => {
  const problems = new Map<status.Key, Entry>();
  let held: Array<() => void> | null = [];
  let stopped = false;

  const listed = ({ open, inUse, rule, status: stat }: Entry): boolean =>
    open && inUse === true && rule?.kind(stat) !== "reply";

  const publish = (): void =>
    setProblems(
      [...problems.values()].filter(listed).map(({ status: stat, count, rule }) => ({
        status: stat,
        count,
        coveredBy: rule?.coveredBy?.(stat) ?? null,
      })),
    );

  const isCovered = (rule: Rule | undefined, stat: status.Status): boolean => {
    const coveringKey = rule?.coveredBy?.(stat);
    if (coveringKey == null) return false;
    const covering = problems.get(coveringKey);
    if (covering?.open !== true) return false;
    const { since } = stat;
    const coveringSince = covering.status.since;
    return since == null || coveringSince == null || since.afterEq(coveringSince);
  };

  const notify = ({ status: stat, identity }: Entry): void =>
    add({ ...stat, problem: identity });

  const show = (problem: Entry): void => {
    const { decision, open, usedAtStart, inUse } = problem;
    if (decision !== "waiting" || !open) return;
    if (usedAtStart !== true || inUse !== true) return;
    if (problem.delayed && problem.status.variant === "warning") return;
    clearTimeout(problem.timer);
    problem.decision = problem.covered ? "covered" : "shown";
    if (problem.decision === "shown") notify(problem);
  };

  const wait = (problem: Entry): void => {
    clearTimeout(problem.timer);
    problem.delayed = true;
    problem.timer = setTimeout(() => {
      problem.delayed = false;
      show(problem);
    }, WARNING_DELAY.milliseconds);
  };

  const check = (problem: Entry, { rule, key }: Owner): void => {
    if (rule.inUse == null) {
      problem.usedAtStart = true;
      problem.inUse = true;
      show(problem);
      return;
    }
    const { status: stat } = problem;
    const checks = ++problem.checks;
    const settle = (inUse: boolean): void => {
      const wasListed = listed(problem);
      if (checks === 1) problem.usedAtStart = inUse;
      if (checks === problem.checks) {
        problem.inUse = inUse;
        if (!inUse) problem.decision = "hidden";
      }
      show(problem);
      if (listed(problem) !== wasListed) publish();
    };
    void rule.inUse({ client, key }).then(
      (inUse) => {
        if (problems.get(stat.key) === problem) settle(inUse);
      },
      // The check only filters noise, so a status whose check fails still shows.
      (exc: unknown) => {
        if (problems.get(stat.key) !== problem) return;
        console.error(`failed to check if the thing of ${stat.key} is in use`, exc);
        settle(true);
      },
    );
  };

  const track = (stat: status.Status, owner: Owner | null): Entry => {
    clearTimeout(problems.get(stat.key)?.timer);
    const problem: Entry = {
      identity: identify(stat),
      status: stat,
      rule: owner?.rule,
      open: true,
      covered: isCovered(owner?.rule, stat),
      decision: "waiting",
      delayed: false,
      count: 1,
      checks: 0,
    };
    problems.set(stat.key, problem);
    return problem;
  };

  const start = (stat: status.Status, owner: Owner | null, kind: Kind): void => {
    const problem = track(stat, owner);
    if (owner == null || kind !== "state") {
      problem.inUse = true;
      problem.decision = "shown";
      notify(problem);
      return;
    }
    if (owner.rule.warningsDelayed === true) wait(problem);
    check(problem, owner);
  };

  const repeat = (problem: Entry, stat: status.Status, owner: Owner | null): void => {
    const reopened = !problem.open;
    const restarted = reopened || problem.decision === "superseded";
    if (problem.decision === "superseded") problem.decision = "waiting";
    problem.status = stat;
    problem.open = true;
    problem.count++;
    problem.covered = isCovered(problem.rule, stat);
    if (reopened && problem.useStale === true && owner != null) {
      problem.useStale = false;
      check(problem, owner);
    }
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
    } else if (restarted && delays) wait(problem);
    show(problem);
  };

  // A problem that waits to notify must not notify after a newer status on its key.
  const retire = (key: status.Key): void => {
    const problem = problems.get(key);
    if (problem?.decision !== "waiting") return;
    clearTimeout(problem.timer);
    problem.decision = "superseded";
  };

  const handle = (stat: status.Status): void => {
    const before = problems.get(stat.key);
    const wasListed = before != null && listed(before);
    route(stat);
    const after = problems.get(stat.key);
    if (wasListed || (after != null && listed(after))) publish();
  };

  const route = (stat: status.Status): void => {
    const problem = problems.get(stat.key);
    const loaded = problem?.loaded;
    if (problem != null) problem.loaded = undefined;
    // The stream can also deliver the write that the load returned.
    if (loaded != null && isSameWrite(loaded, stat)) return;
    const owner = getOwner(rules, stat.key);
    const kind = owner?.rule.kind(stat) ?? "alarm";
    if (kind !== "state") retire(stat.key);
    const bad = PROBLEM_VARIANTS.includes(stat.variant);
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
    if (problem?.identity === identify(stat)) repeat(problem, stat, owner);
    else start(stat, owner, kind);
  };

  const drop = (key: status.Key): void => {
    const problem = problems.get(key);
    if (problem == null) return;
    clearTimeout(problem.timer);
    problems.delete(key);
    if (listed(problem)) publish();
  };

  const run = (event: () => void): void => {
    if (held == null) event();
    else held.push(event);
  };

  const write = (stat: status.Status): void => run(() => handle(stat));

  const remove = (key: status.Key): void => run(() => drop(key));

  const load = (open: status.Status[]): void => {
    if (stopped || held == null) return;
    open.forEach((stat) => {
      const owner = getOwner(rules, stat.key);
      const problem = track(stat, owner);
      problem.decision = "hidden";
      problem.loaded = stat;
      // A reply gets the check too, since a later state can join its problem.
      if (owner == null || owner.rule.kind(stat) === "alarm") problem.inUse = true;
      else check(problem, owner);
    });
    publish();
    const events = held;
    held = null;
    events.forEach((event) => event());
  };

  const rechecking = new Set<Rule>();
  const recheck = (rule: Rule): void => {
    if (rechecking.has(rule)) return;
    rechecking.add(rule);
    queueMicrotask(() => {
      rechecking.delete(rule);
      problems.forEach((problem, key) => {
        const owner = getOwner(rules, key);
        if (owner?.rule !== rule || problem.checks === 0) return;
        // A closed problem checks again only when it reopens.
        if (problem.open) check(problem, owner);
        else problem.useStale = true;
      });
    });
  };

  const stop = (): void => {
    stopped = true;
    problems.forEach(({ timer }) => clearTimeout(timer));
    problems.clear();
  };

  return { write, remove, load, recheck, stop };
};

/**
 * Runs the forwarder of {@link Provider}.
 * @returns the open problems.
 */
export const useForward = (rules: Rules): Problem[] => {
  const client = Synnax.use();
  const add = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const [problems, setProblems] = useState<Problem[]>([]);
  useEffect(() => {
    if (client == null) return;
    const forwarder = createForwarder({ client, rules, add, setProblems });
    const removeWriteListener = client.statuses.onWrite(forwarder.write);
    const removeDeleteListener = client.statuses.onDelete(forwarder.remove);
    const removeUseListeners = Object.values(rules).map((rule) =>
      rule?.onUseChange?.(client, () => forwarder.recheck(rule)),
    );
    client.statuses
      .retrieve({ variants: PROBLEM_VARIANTS })
      .then(forwarder.load, (exc: unknown) => {
        handleError(exc, "Failed to retrieve open problems");
        forwarder.load([]);
      });
    return () => {
      destructor.unwind(
        removeWriteListener,
        removeDeleteListener,
        ...removeUseListeners,
      );
      forwarder.stop();
      setProblems([]);
    };
  }, [client, rules, add, handleError]);
  return problems;
};
