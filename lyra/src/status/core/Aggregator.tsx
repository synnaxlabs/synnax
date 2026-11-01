// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan, TimeStamp } from "@synnaxlabs/x";
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
} from "@/status/core/errorHandler";
import { create, type Status } from "@/status/core/status";

interface Received {
  status: Status;
  /** When the aggregator got the status. Notifications expire from this time. */
  time: TimeStamp;
}

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
  /** Statuses kept before the oldest are dropped. Defaults to 500. */
  maxHistory?: number;
  /** Prints each error reported through an error handler. Defaults to console.error. */
  log?: Log;
}

const TRUNCATE_FACTOR = 0.9;

/**
 * Collects statuses from its subtree and hands them to {@link useNotifications} and
 * the status list. Mount one near the root of the app.
 */
export const Aggregator = ({
  children,
  maxHistory = 500,
  log = printError,
}: AggregatorProps) => {
  const [received, setReceived] = useState<Received[]>([]);
  if (received.length > maxHistory)
    setReceived(received.slice(0, Math.floor(maxHistory * TRUNCATE_FACTOR)));
  const handleAdd: Adder = (spec) => {
    const next = { status: create(spec), time: TimeStamp.now() };
    setReceived((prev) => [
      next,
      ...prev.filter(({ status }) => status.key != next.status.key),
    ]);
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
  Status<Details> & { count: number };

/** Return value for {@link useNotifications}. */
export interface UseNotificationsReturn<Details extends z.ZodType = z.ZodNever> {
  statuses: NotificationSpec<Details>[];
  /** Hides one notification for good. */
  silence: (key: string) => void;
  silenceAll: () => void;
}

const identity = ({ variant, name, message, description }: Status): string =>
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

const group = (
  received: Received[],
  threshold: TimeStamp,
  silencedKeys: Set<string>,
): NotificationSpec[] => {
  const active = received
    .filter(
      ({ status, time }) => time.afterEq(threshold) && !silencedKeys.has(status.key),
    )
    .map(({ status }) => status);
  const grouped = active.reduce((acc, status) => {
    const key = identity(status);
    if (!acc.has(key)) {
      acc.set(key, { ...status, count: 1 });
      return acc;
    }
    const existing = acc.get(key)!;
    acc.set(key, {
      ...existing,
      count: existing.count + 1,
      time: status.time.after(existing.time) ? status.time : existing.time,
    });
    return acc;
  }, new Map<string, NotificationSpec>());
  return Array.from(grouped.values());
};

/**
 * @returns the statuses recent enough to show as notifications, with identical ones
 * folded into a single entry carrying a count. They drop off on their own after the
 * expiration, counted from when the aggregator got each status, not from its time.
 */
export const useNotifications = ({
  expiration = DEFAULT_EXPIRATION,
  poll = DEFAULT_EXPIRATION_POLL,
}: UseNotificationsProps = NO_PROPS): UseNotificationsReturn => {
  const received = useContext();
  const [silencedKeys, setSilencedKeys] = useState<Set<string>>(() => new Set());
  const now = useNow((now) => now.add(poll));
  const statusesRef = useSyncedRef(received.map(({ status }) => status));

  const filtered = group(received, now.sub(expiration), silencedKeys);

  const silence = (key: string) => {
    setSilencedKeys((prev) => {
      const next = new Set<string>();
      const existing = statusesRef.current.find(({ key: k }) => k === key);
      if (!prev.has(key)) next.add(key);
      if (existing != null) {
        const id = identity(existing);
        statusesRef.current
          .filter((status) => identity(status) === id)
          .forEach((status) => next.add(status.key));
      }
      if (next.size == 0) return prev;
      return new Set([...next, ...prev]);
    });
  };

  const silenceAll = () => {
    setSilencedKeys((prev) => {
      const next = new Set(prev);
      statusesRef.current.forEach(({ key }) => next.add(key));
      return next;
    });
  };

  return { statuses: filtered, silence, silenceAll };
};
