// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useContext as reactUseContext,
  useCallback,
  useMemo,
  useState,
} from "react";

import { Optional, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/aether";
import { status } from "@/status/aether";

interface ContextValue extends z.infer<typeof status.aggregatorStateZ> {
  add: (status: Optional<status.Spec, "time">) => void;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  statuses: [],
  add: () => {},
};

const Context = createContext<ContextValue>(ZERO_CONTEXT_VALUE);

export const useContext = reactUseContext;

export interface AggregatorProps extends PropsWithChildren {}

export const Aggregator = Aether.wrap<AggregatorProps>(
  status.Aggregator.TYPE,
  ({ aetherKey, children }): ReactElement => {
    const [{ path }, { statuses }, setState] = Aether.use({
      aetherKey,
      type: status.Aggregator.TYPE,
      schema: status.aggregatorStateZ,
      initialState: { statuses: [] },
    });

    const handleAdd: ContextValue["add"] = useCallback(
      (status) => {
        const spec: status.Spec = { time: TimeStamp.now(), ...status };
        setState((state) => ({ ...state, statuses: [...state.statuses, spec] }));
      },
      [setState]
    );

    const value = useMemo<ContextValue>(
      () => ({ statuses, add: handleAdd }),
      [statuses, handleAdd]
    );

    return (
      <Context.Provider value={value}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Context.Provider>
    );
  }
);

export const useAggregator = (): ContextValue["add"] => useContext(Context).add;

export interface UseNotificationsProps {
  expiration?: TimeSpan;
}

const DEFAULT_EXPIRATION = TimeSpan.seconds(5);

export const useNotifications = ({
  expiration = DEFAULT_EXPIRATION,
}): ContextValue["statuses"] => {
  const { statuses } = useContext(Context);

  const [threshold, setThreshold] = useState<TimeStamp>(() => TimeStamp.now());

  return statuses.filter((status) => {
    const new_ = status.time.after(threshold);
    if (new_) setTimeout(() => setThreshold(status.time), expiration.milliseconds);
    return new_;
  });
};
