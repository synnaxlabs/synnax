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

export interface ExceptionHandler extends status.ExceptionHandler {
  (func: () => Promise<void>, message?: string): void;
}

const errorToStatus = (exc: unknown, message?: string): status.CrudeSpec => {
  if (!(exc instanceof Error)) throw exc;
  return {
    variant: "error",
    message: message ?? exc.message,
    description: message != null ? exc.message : undefined,
  };
};

export const useExceptionHandler = (): ExceptionHandler => {
  const add = useAdder();
  return useCallback(
    (excOrFunc: unknown | (() => Promise<void>), message?: string): void => {
      if (typeof excOrFunc === "function") {
        void (async () => {
          try {
            await excOrFunc();
          } catch (exc) {
            add(errorToStatus(exc, message));
          }
        })();
        return;
      }
      add(errorToStatus(excOrFunc, message));
    },
    [add],
  );
};

export interface NotificationSpec extends status.Spec {
  count: number;
}

export interface UseNotificationsReturn {
  statuses: NotificationSpec[];
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
  const statuses = use(StatusesContext);
  const [silencedKeys, setSilencedKeys] = useState<Set<string>>(new Set());
  const [now, setNow] = useState(() => TimeStamp.now());

  useEffect(() => {
    const interval = setInterval(() => setNow(TimeStamp.now()), poll.milliseconds);
    return () => clearInterval(interval);
  }, [poll.milliseconds]);

  const { statuses: filtered } = useMemo(() => {
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

    return { statuses: Array.from(grouped.values()) };
  }, [statuses, expiration, silencedKeys, now]);

  const silence = useCallback((key: string) => {
    setSilencedKeys((prev) => {
      const next = new Set(prev);
      next.add(key);
      return next;
    });
  }, []);

  return { statuses: filtered, silence };
};
