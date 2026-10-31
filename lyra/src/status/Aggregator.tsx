// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type PropsWithChildren, useState } from "react";
import type z from "zod";

import { context } from "@/context";
import { useNow, useSyncedRef } from "@/hooks";
import {
  type Adder,
  type AsyncErrorHandler,
  createAsyncErrorHandler,
  createErrorHandler,
  type ErrorHandler,
  type Log,
} from "@/status/errorHandler";
import { create, type Status } from "@/status/status";

interface Received {
  id: string;
  /** The latest status. */
  status: Status;
  /** When the aggregator got the first status. Notifications expire from this time. */
  time: TimeStamp;
  /** How many statuses joined. */
  count: number;
}

// A key holds one notification for its problem and one for its other statuses.
const sameSlot = (a: Status, b: Status): boolean =>
  a.key === b.key && (a.problem == null) === (b.problem == null);

const sameProblem = (a: Status, b: Status): boolean =>
  a.problem != null && a.problem === b.problem;

const [Context, useContext] = context.create<Received[]>({
  defaultValue: [],
  displayName: "Status.Context",
});

const [AdderContext, useAdder] = context.create<Adder>({
  defaultValue: () => {},
  displayName: "Status.AdderContext",
});
export { useAdder };

interface Handlers {
  error: ErrorHandler;
  async: AsyncErrorHandler;
}

const createHandlers = (add: Adder, log: Log): Handlers => ({
  error: createErrorHandler(add, log),
  async: createAsyncErrorHandler(add, log),
});

// Reads console.error at call time, so a test's replacement or a session recorder's
// wrapper applies without new handlers.
const printError: Log = (message) => console.error(message);

const [HandlersContext, useHandlers] = context.create<Handlers>({
  defaultValue: createHandlers(() => {}, printError),
  displayName: "Status.HandlersContext",
});

/** Props for {@link Aggregator}. */
export interface AggregatorProps extends PropsWithChildren {
  /** Statuses kept before the oldest are dropped. Problems stay. Defaults to 500. */
  maxHistory?: number;
  /** Prints each error reported through an error handler. Defaults to console.error. */
  log?: Log;
}

const TRUNCATE_FACTOR = 0.9;

const isProblem = ({ status }: Received): boolean => status.problem != null;

// A problem notifies only once while its status stays in the history, so the history
// keeps every problem. A key holds one problem at most, which bounds them.
const trim = (received: Received[], size: number): Received[] => {
  const dropped = new Set(received.filter((r) => !isProblem(r)).slice(size));
  return received.filter((r) => !dropped.has(r));
};

/**
 * Collects statuses from its subtree and hands them to {@link useNotifications} and the
 * status list. Mount one near the root of the app. Each key holds one notification for
 * its problem and one for its other statuses, and a new status replaces the one in its
 * place. A warning or error of the same problem joins its notification instead: the
 * count goes up, and the notification does not show again.
 */
export const Aggregator = ({
  children,
  maxHistory = 500,
  log = printError,
}: AggregatorProps) => {
  const [received, setReceived] = useState<Received[]>([]);
  if (received.filter((r) => !isProblem(r)).length > maxHistory)
    setReceived(trim(received, Math.floor(maxHistory * TRUNCATE_FACTOR)));
  const handleAdd: Adder = (spec) => {
    const status = create(spec);
    setReceived((prev) => {
      const held = prev.find((r) => sameSlot(r.status, status));
      if (held != null && sameProblem(held.status, status))
        return prev.map((r) => (r === held ? { ...r, status, count: r.count + 1 } : r));
      const next = { id: id.create(), status, time: TimeStamp.now(), count: 1 };
      return [next, ...prev.filter((r) => r !== held)];
    });
  };
  const handlers = createHandlers(handleAdd, log);
  return (
    <Context value={received}>
      <AdderContext value={handleAdd}>
        <HandlersContext value={handlers}>{children}</HandlersContext>
      </AdderContext>
    </Context>
  );
};

/**
 * @returns a handler that turns a caught error into an error status on the enclosing
 * {@link Aggregator}. Use it in place of `console.error` in a UI path.
 *
 * @example handleError(err, "Failed to save range");
 */
export const useErrorHandler = (): ErrorHandler => useHandlers().error;

/**
 * @returns a handler that runs an async function and reports a rejection as an error
 * status. Use it wherever an effect or a click handler would otherwise float a promise.
 */
export const useAsyncErrorHandler = (): AsyncErrorHandler => useHandlers().async;

/** A status shown as a notification, with how many identical ones it stands for. */
export type NotificationSpec<Details extends z.ZodType = z.ZodNever> =
  Status<Details> & {
    /** Identifies the notification for {@link UseNotificationsReturn.silence}. */
    id: string;
    count: number;
  };

/** Return value for {@link useNotifications}. */
export interface UseNotificationsReturn<Details extends z.ZodType = z.ZodNever> {
  statuses: NotificationSpec<Details>[];
  /**
   * Closes the notification with the id. Later statuses of the same problem stay
   * closed. The next problem on the key shows.
   */
  silence: (id: string) => void;
  /** Closes every notification that shows now. */
  silenceAll: () => void;
}

const groupKey = ({ variant, name, message, description }: Status): string =>
  JSON.stringify([variant, name, message, description]);

const DEFAULT_EXPIRATION = TimeSpan.seconds(7);
const DEFAULT_EXPIRATION_POLL = TimeSpan.seconds(1);

interface UseNotificationsProps {
  expiration?: CrudeTimeSpan;
  poll?: CrudeTimeSpan;
}

// A `{}` default makes a new object on each call, so the compiler cannot cache the
// values read from it.
const NO_PROPS: UseNotificationsProps = {};

const group = (active: Received[]): NotificationSpec[] => {
  const grouped = active.reduce((acc, { id, status, count }) => {
    const key = groupKey(status);
    const existing = acc.get(key);
    if (existing == null) {
      acc.set(key, { ...status, id, count });
      return acc;
    }
    acc.set(key, {
      ...existing,
      count: existing.count + count,
      time: status.time.after(existing.time) ? status.time : existing.time,
    });
    return acc;
  }, new Map<string, NotificationSpec>());
  return Array.from(grouped.values());
};

/**
 * @returns the statuses recent enough to show as notifications, with identical ones
 * folded into a single entry carrying a count. They drop off on their own after the
 * expiration, counted from when the aggregator got the first status of each, not from
 * its time.
 */
export const useNotifications = ({
  expiration = DEFAULT_EXPIRATION,
  poll = DEFAULT_EXPIRATION_POLL,
}: UseNotificationsProps = NO_PROPS): UseNotificationsReturn => {
  const received = useContext();
  const [silenced, setSilenced] = useState<Set<string>>(() => new Set());
  const now = useNow((now) => now.add(poll));
  const threshold = now.sub(expiration);
  const active = received.filter(
    (r) => r.time.afterEq(threshold) && !silenced.has(r.id),
  );
  const activeRef = useSyncedRef(active);

  const close = (ids: string[]) =>
    setSilenced((prev) => (ids.length === 0 ? prev : new Set([...prev, ...ids])));

  const silence = (notification: string) => {
    const closed = activeRef.current.find((r) => r.id === notification);
    if (closed == null) return;
    const shared = groupKey(closed.status);
    close(
      activeRef.current
        .filter(({ status }) => groupKey(status) === shared)
        .map((r) => r.id),
    );
  };

  const silenceAll = () => close(activeRef.current.map((r) => r.id));

  return { statuses: group(active), silence, silenceAll };
};
