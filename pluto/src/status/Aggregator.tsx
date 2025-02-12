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

export const Aggregator = ({ children, maxHistory = 500 }: AggregatorProps) => {
  const [{ path }, { statuses }, setState] = Aether.use({
    type: status.Aggregator.TYPE,
    schema: status.aggregatorStateZ,
    initialState: { statuses: [] },
  });
  if (statuses.length > maxHistory) {
    const slice = Math.floor(maxHistory * 0.9);
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
  const [silenced, setSilenced] = useState<string[]>([]);

  const filtered = useMemo(() => {
    const fresh = statuses.filter(({ key, time }) => {
      if (time.beforeEq(threshold)) return false;
      setTimeout(
        () => setThreshold((t) => (time.after(t) ? time : t)),
        expiration.milliseconds,
      );
      return !silenced.includes(key);
    });
    return filterDuplicates(fresh);
  }, [statuses, threshold, setThreshold, expiration, silenced]);

  const silence: UseNotificationsReturn["silence"] = useCallback(
    (key) => {
      const status = statuses.find((s) => s.key === key);
      if (status == null) return;
      const duplicates = statuses
        .filter(
          ({ message, variant }) =>
            message === status.message && variant === status.variant,
        )
        .map(({ key }) => key);
      setSilenced((silenced) => [...silenced, ...duplicates]);
    },
    [statuses, setSilenced],
  );

  return { statuses: filtered, silence };
};

const filterDuplicates = (statuses: status.Spec[]): NotificationSpec[] => {
  const map = new Map<string, NotificationSpec>();
  statuses.forEach((status) => {
    const { message, variant } = status;
    const key = `${message}-${variant}`;
    const existing = map.get(key);
    if (existing == null) {
      map.set(key, { ...status, count: 1 });
      return;
    }
    existing.count += 1;
    if (existing.time.before(status.time)) existing.time = status.time;
  });
  return Array.from(map.values());
};
