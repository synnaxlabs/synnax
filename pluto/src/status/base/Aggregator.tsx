// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status as cstatus } from "@synnaxlabs/client";
import { type CrudeTimeSpan, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import type z from "zod";

import { Aether } from "@/aether";
import { context } from "@/context";
import { useSyncedRef } from "@/hooks";
import { status } from "@/status/aether";

const [Context, useContext] = context.create<cstatus.Status[]>({
  defaultValue: [],
  displayName: "Status.Context",
});

/** Adds a status to the enclosing {@link Aggregator}. */
export interface Adder extends status.Adder {}

const [AdderContext, useAdder] = context.create<Adder>({
  defaultValue: () => {},
  displayName: "Status.AdderContext",
});
export { useAdder };

/** Props for {@link Aggregator}. */
export interface AggregatorProps extends PropsWithChildren {
  /** Statuses kept before the oldest are dropped. Defaults to 500. */
  maxHistory?: number;
}

const TRUNCATE_FACTOR = 0.9;

/**
 * Collects statuses from its subtree and from the aether worker thread, and hands them
 * to {@link useNotifications} and the status list. Mount one near the root of the app.
 */
export const Aggregator = ({ children, maxHistory = 500 }: AggregatorProps) => {
  const [{ path }, { statuses }, setState] = Aether.use({
    type: status.Aggregator.TYPE,
    schema: status.aggregatorStateZ,
    initialState: { statuses: [] },
  });
  if (statuses.length > maxHistory) {
    const slice = Math.floor(maxHistory * TRUNCATE_FACTOR);
    setState((state) => ({ ...state, statuses: statuses.slice(0, slice) }));
  }
  const handleAdd: Adder = useCallback(
    (status) => {
      const spec = { time: TimeStamp.now(), key: id.create(), ...status };
      setState((state) => ({
        ...state,
        statuses: [spec, ...state.statuses.filter((s) => s.key != spec.key)],
      }));
    },
    [setState],
  );

  return (
    <Context value={statuses}>
      <AdderContext value={handleAdd}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </AdderContext>
    </Context>
  );
};

export interface ErrorHandler extends status.ErrorHandler {}

export interface AsyncErrorHandler extends status.AsyncErrorHandler {}

/**
 * @returns a handler that turns a caught error into an error status on the enclosing
 * {@link Aggregator}. Use it in place of `console.error` in a UI path.
 *
 * @example handleError(err, "failed to save the range");
 */
export const useErrorHandler = (): ErrorHandler => {
  const add = useAdder();
  return useMemo(() => status.createErrorHandler(add), [add]);
};

/**
 * @returns a handler that runs an async function and reports a rejection as an error
 * status. Use it wherever an effect or a click handler would otherwise float a promise.
 */
export const useAsyncErrorHandler = (): AsyncErrorHandler => {
  const add = useAdder();
  return useMemo(() => status.createAsyncErrorHandler(add), [add]);
};

/** A status shown as a notification, with how many identical ones it stands for. */
export type NotificationSpec<Details extends z.ZodType = z.ZodNever> =
  cstatus.Status<Details> & {
    count: number;
  };

/** Return value for {@link useNotifications}. */
export interface UseNotificationsReturn<Details extends z.ZodType = z.ZodNever> {
  statuses: NotificationSpec<Details>[];
  /** Hides one notification for good. */
  silence: (key: string) => void;
  silenceAll: () => void;
}

const DEFAULT_EXPIRATION = TimeSpan.seconds(7);
const DEFAULT_EXPIRATION_POLL = TimeSpan.seconds(1);

interface UseNotificationsProps {
  expiration?: CrudeTimeSpan;
  poll?: CrudeTimeSpan;
}

/**
 * @returns the statuses recent enough to show as notifications, with identical ones
 * folded into a single entry carrying a count. They drop off on their own after the
 * expiration.
 */
export const useNotifications = ({
  expiration = DEFAULT_EXPIRATION,
  poll = DEFAULT_EXPIRATION_POLL,
}: UseNotificationsProps = {}): UseNotificationsReturn => {
  const statuses = useContext();
  const [silencedKeys, setSilencedKeys] = useState<Set<string>>(new Set());
  const [now, setNow] = useState(() => TimeStamp.now());
  const pollMs = new TimeSpan(poll).milliseconds;

  useEffect(() => {
    const interval = setInterval(() => setNow(TimeStamp.now()), pollMs);
    return () => clearInterval(interval);
  }, [pollMs]);

  const filtered = useMemo(() => {
    const threshold = now.sub(expiration);

    const active = statuses.filter(
      ({ key, time }) => time.afterEq(threshold) && !silencedKeys.has(key),
    );

    const grouped = active.reduce((acc, status) => {
      const key = `${status.variant}:${status.message}`;
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
  }, [statuses, expiration, silencedKeys, now]);

  const statusesRef = useSyncedRef(statuses);

  const silence = useCallback((key: string) => {
    setSilencedKeys((prev) => {
      const next = new Set<string>();
      const existing = statusesRef.current.find(({ key: k }) => k === key);
      if (!prev.has(key)) next.add(key);
      if (existing != null) {
        const silenced = statusesRef.current.filter(
          ({ message, variant }) =>
            message === existing.message && variant === existing.variant,
        );
        silenced.forEach((status) => next.add(status.key));
      }
      if (next.size == 0) return prev;
      return new Set([...next, ...prev]);
    });
  }, []);

  const silenceAll = useCallback(() => {
    setSilencedKeys((prev) => {
      const next = new Set(prev);
      statusesRef.current.forEach(({ key }) => next.add(key));
      return next;
    });
  }, []);

  return { statuses: filtered, silence, silenceAll };
};
