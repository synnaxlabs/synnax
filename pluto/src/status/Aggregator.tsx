// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  use,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { Aether } from "@/aether";
import { useSyncedRef } from "@/hooks";
import { status } from "@/status/aether";

const StatusesContext = createContext<status.Spec[]>([]);

export interface Adder extends status.Adder {}

const AdderContext = createContext<Adder>(() => {});

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
      const spec = { time: TimeStamp.now(), key: id.id(), ...status };
      setState((state) => ({ ...state, statuses: [spec, ...state.statuses] }));
    },
    [setState],
  );

  return (
    <StatusesContext value={statuses}>
      <AdderContext value={handleAdd}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </AdderContext>
    </StatusesContext>
  );
};

export const useAdder = () => use(AdderContext);

export interface ExceptionHandler extends status.ExceptionHandler {}

export const useExceptionHandler = (): ExceptionHandler => {
  const add = useAdder();
  return useCallback(
    (exc: unknown, message?: string): void => {
      if (!(exc instanceof Error)) throw exc;
      add({
        variant: "error",
        message: message ?? exc.message,
        description: message != null ? exc.message : undefined,
      });
    },
    [add],
  );
};

export interface UseNotificationsProps {
  expiration?: TimeSpan;
}

export interface NotificationSpec extends status.Spec {
  count: number;
}

export interface UseNotificationsReturn {
  statuses: NotificationSpec[];
  silence: (key: string) => void;
}

const DEFAULT_EXPIRATION = TimeSpan.seconds(7);

export const useNotifications = ({
  expiration = DEFAULT_EXPIRATION,
}: UseNotificationsProps = {}): UseNotificationsReturn => {
  const statuses = use(StatusesContext);
  const [threshold, setThreshold] = useState<TimeStamp>(TimeStamp.now());
  const [silenced, setSilenced] = useState<Set<string>>(new Set());

  const filtered = useMemo(() => {
    const fresh = statuses.filter(
      ({ key, time }) => time.after(threshold) && !silenced.has(key),
    );
    return filterDuplicates(fresh);
  }, [statuses, threshold, silenced]);

  useEffect(() => {
    if (filtered.length === 0) return;
    const lastTime = filtered.reduce(
      (latest, { time }) => (time.after(latest) ? time : latest),
      threshold,
    );
    if (lastTime.beforeEq(threshold)) return;
    const timeout = setTimeout(() => setThreshold(lastTime), expiration.milliseconds);
    return () => clearTimeout(timeout);
  }, [filtered, expiration, threshold]);

  const statusesRef = useSyncedRef(statuses);

  const silence: UseNotificationsReturn["silence"] = useCallback(
    (key) => {
      const status = statusesRef.current.find((s) => s.key === key);
      if (status == null) return;
      const duplicates = statusesRef.current
        .filter(
          ({ message, variant }) =>
            message === status.message && variant === status.variant,
        )
        .map(({ key }) => key);
      // create a new set to trigger a rerender
      setSilenced((prev) => {
        const next = new Set(prev);
        let changed = false;
        duplicates.forEach((key) => {
          if (next.has(key)) return;
          next.add(key);
          changed = true;
        });
        return changed ? next : prev;
      });
    },
    [statusesRef, setSilenced],
  );

  return { statuses: filtered, silence };
};

const filterDuplicates = (statuses: status.Spec[]): NotificationSpec[] => {
  const map = new Map<string, NotificationSpec>();
  statuses.forEach((status) => {
    const { message, variant } = status;
    const key = JSON.stringify({ message, variant });
    const existing = map.get(key);
    if (existing == null) {
      map.set(key, { ...status, count: 1 });
      return;
    }
    map.set(key, {
      ...existing,
      count: existing.count + 1,
      time: status.time.after(existing.time) ? status.time : existing.time,
    });
  });
  return Array.from(map.values());
};
