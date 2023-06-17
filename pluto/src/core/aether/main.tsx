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
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { WorkerMessage } from "@/core/aether/message";
import { PsuedoSetStateArg } from "@/core/hooks/useStateRef";
import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { useTypedWorker } from "@/core/worker/Context";

export interface AetherContextValue {
  path: string[];
  setState: (
    path: string[],
    type: string,
    state: any,
    transfer?: Transferable[]
  ) => void;
  delete: (path: string[]) => void;
}

export const AetherContext = createContext<AetherContextValue | null>(null);

export interface UseAetherReturn<S extends unknown> {
  key: string;
  path: string[];
  state: [S, (state: PsuedoSetStateArg<S>, transfer?: Transferable[]) => void];
}

export const useAetherContext = (): AetherContextValue => {
  const ctx = useContext(AetherContext);
  if (ctx == null) throw new Error("useBobContext must be used within a BobProvider");
  return ctx;
};

export const useAether = <S extends unknown>(
  type: string,
  initialState: S,
  key?: string,
  initialTransfer: Transferable[] = []
): UseAetherReturn<S> => {
  const oKey = useUniqueKey(key);
  const ctx = useAetherContext();
  const path = useMemo(() => [...ctx.path, oKey], [ctx]);

  const [internalState, setInternalState] = useState(initialState);

  const transferred = useRef<Transferable[]>([]);

  const setState = useCallback(
    (next: PsuedoSetStateArg<S>, transfer: Transferable[] = []): void => {
      const untransferred = transfer.filter((t) => !transferred.current.includes(t));
      transferred.current = transferred.current.concat(untransferred);
      if (typeof next === "function")
        setInternalState((prev) => {
          const nextS = (next as (prev: S) => S)(prev);
          ctx.setState(path, type, nextS, untransferred);
          return nextS;
        });
      else {
        setInternalState(next);
        ctx.setState(path, type, next, untransferred);
      }
    },
    [ctx, path, type]
  );

  useEffect(
    () => setState(initialState, initialTransfer),
    [setState, initialState, initialState]
  );

  useEffect(() => () => ctx.delete(path), [path]);

  return { key: oKey, path, state: [internalState, setState] };
};

export interface AetherProviderProps extends PropsWithChildren {
  workerKey: string;
}

export const AetherProvider = ({
  workerKey,
  children,
}: AetherProviderProps): JSX.Element => {
  const worker = useTypedWorker<WorkerMessage>(workerKey);
  const setState = useCallback(
    (path: string[], type: string, state: any, transfer: Transferable[] = []): void =>
      worker.send({ variant: "setState", path, type, state }, transfer),
    [worker]
  );

  const delete_ = useCallback(
    (path: string[]): void => worker.send({ variant: "delete", path }),
    [worker]
  );

  return (
    <AetherContext.Provider value={{ path: [], setState, delete: delete_ }}>
      {children}
    </AetherContext.Provider>
  );
};

export interface BobCompositeProps extends PropsWithChildren {
  path: string[];
}

export const AetherComposite = ({ children, path }: BobCompositeProps): JSX.Element => (
  <AetherContext.Provider value={{ ...useAetherContext(), path }}>
    {children}
  </AetherContext.Provider>
);

export const Aether = {
  Provider: AetherProvider,
  Composite: AetherComposite,
  use: useAether,
};
