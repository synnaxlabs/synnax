// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, type status as xstatus, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { Aether } from "@/aether";
import { context } from "@/context";
import { useSyncedRef } from "@/hooks";
import { status } from "@/status/aether";

const [Context, useContext] = context.create<xstatus.Status[]>({
  defaultValue: [],
  displayName: "Status.Context",
});

export interface Adder extends status.Adder {}

const [AdderContext, useAdder] = context.create<Adder>({
  defaultValue: () => {},
  displayName: "Status.AdderContext",
});
export { useAdder };

export interface AggregatorProps extends PropsWithChildren {
  maxHistory?: number;
}

const TRUNCATE_FACTOR = 0.9;

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

export const useErrorHandler = (): ErrorHandler => {
  const add = useAdder();
  return useMemo(() => status.createErrorHandler(add), [add]);
};

export const useAsyncErrorHandler = (): AsyncErrorHandler => {
  const add = useAdder();
  return useMemo(() => status.createAsyncErrorHandler(add), [add]);
};

export type NotificationSpec<Details = never> = xstatus.Status<Details> & {
  count: number;
};

export interface UseNotificationsReturn<Details = never> {
  statuses: NotificationSpec<Details>[];
  silence: (key: string) => void;
}

const DEFAULT_EXPIRATION = TimeSpan.seconds(7);
const DEFAULT_EXPIRATION_POLL = TimeSpan.seconds(1);

interface UseNotificationsProps {
  expiration?: TimeSpan;
  poll?: TimeSpan;
}

export const useNotifications = ({
  expiration = DEFAULT_EXPIRATION,
  poll = DEFAULT_EXPIRATION_POLL,
}: UseNotificationsProps = {}): UseNotificationsReturn => {
  const statuses = useContext();
  const [silencedKeys, setSilencedKeys] = useState<Set<string>>(new Set());
  const [now, setNow] = useState(() => TimeStamp.now());

  useEffect(() => {
    const interval = setInterval(() => setNow(TimeStamp.now()), poll.milliseconds);
    return () => clearInterval(interval);
  }, [poll.milliseconds]);

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

  return { statuses: filtered, silence };
};
