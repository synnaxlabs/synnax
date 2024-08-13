// Copyright 2024 Synnax Labs, Inc.
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
  type ReactElement,
  useCallback,
  useContext as reactUseContext,
  useMemo,
  useState,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useSyncedRef } from "@/hooks";
import { status } from "@/status/aether";

interface ContextValue extends z.infer<typeof status.aggregatorStateZ> {
  add: (status: status.CrudeSpec) => void;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  statuses: [],
  add: () => {},
};

const Context = createContext<ContextValue>(ZERO_CONTEXT_VALUE);

export const useContext = reactUseContext;

export interface AggregatorProps extends PropsWithChildren {
  maxHistory?: number;
}

export const Aggregator = Aether.wrap<AggregatorProps>(
  status.Aggregator.TYPE,
  ({ aetherKey, children, maxHistory = 500 }): ReactElement => {
    const [{ path }, { statuses }, setState] = Aether.use({
      aetherKey,
      type: status.Aggregator.TYPE,
      schema: status.aggregatorStateZ,
      initialState: { statuses: [] },
    });

    if (maxHistory != null && statuses.length > maxHistory) {
      const slice = Math.floor(maxHistory * 0.9);
      setState((state) => ({ ...state, statuses: statuses.slice(0, slice) }));
    }

    const handleAdd: ContextValue["add"] = useCallback(
      (status) => {
        const spec: status.Spec = { time: TimeStamp.now(), key: id.id(), ...status };
        setState((state) => ({ ...state, statuses: [spec, ...state.statuses] }));
      },
      [setState],
    );

    const value = useMemo<ContextValue>(
      () => ({ statuses, add: handleAdd }),
      [statuses, handleAdd],
    );

    return (
      <Context.Provider value={value}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Context.Provider>
    );
  },
);

export const useAggregator = (): ContextValue["add"] => useContext(Context).add;

export interface UseNotificationsProps {
  expiration?: TimeSpan;
}

const DEFAULT_EXPIRATION = TimeSpan.seconds(5);

export interface NotificationSpec extends status.Spec {
  count: number;
}

export interface UseNotificationsReturn {
  statuses: NotificationSpec[];
  silence: (key: string) => void;
}

export const useNotifications = (
  props: UseNotificationsProps = {},
): UseNotificationsReturn => {
  const { statuses } = useContext(Context);
  const { expiration = DEFAULT_EXPIRATION } = props;
  const statusesRef = useSyncedRef(statuses);

  const [threshold, setThreshold] = useState<TimeStamp>(() => TimeStamp.now());
  const [silenced, setSilenced] = useState<string[]>([]);

  const filtered = statuses.filter((status) => {
    const new_ = status.time.after(threshold);
    if (new_)
      setTimeout(
        () => setThreshold((p) => (status.time.after(p) ? status.time : p)),
        expiration.milliseconds,
      );
    return new_ && !silenced.includes(status.key);
  });

  const silence = useCallback(
    (key: string) => {
      const s = statusesRef.current.find((s) => s.key === key);
      if (s == null) return;
      const duplicates = findDuplicateStatus(s, statusesRef.current);
      setSilenced((silenced) => [...silenced, ...duplicates.map((s) => s.key)]);
    },
    [setSilenced],
  );

  return {
    statuses: reduceDuplicateStatuses(filtered),
    silence,
  };
};

const reduceDuplicateStatuses = (statuses: status.Spec[]): NotificationSpec[] =>
  statuses.reduce<NotificationSpec[]>((acc, status) => {
    const { message, variant } = status;
    const existing = acc.find((s) => s.message === message && s.variant === variant);
    if (existing != null) {
      existing.count += 1;
      if (existing.time.before(status.time)) existing.time = status.time;
    } else acc.push({ ...status, count: 1 });
    return acc;
  }, []);

const findDuplicateStatus = (
  target: status.Spec,
  statuses: status.Spec[],
): status.Spec[] =>
  statuses.filter((status) => {
    const { message, variant } = status;
    return message === target.message && variant === target.variant;
  });
