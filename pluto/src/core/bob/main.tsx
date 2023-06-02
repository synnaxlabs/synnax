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
  useState,
} from "react";

import { WorkerMessage } from "@/core/bob/message";
import { PsuedoSetStateArg } from "@/core/hooks/useStateRef";
import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { useTypedWorker } from "@/worker/Context";

export interface BobContextValue {
  path: string[];
  bootstrap: (state: any, transfer?: Transferable[]) => void;
  setState: (
    path: string[],
    type: string,
    state: any,
    transfer?: Transferable[]
  ) => void;
  delete: (path: string[]) => void;
}

export const BobContext = createContext<BobContextValue | null>(null);

export interface UseBobComponentReturn<S extends unknown> {
  key: string;
  path: string[];
  state: [S, (state: PsuedoSetStateArg<S>, transfer?: Transferable[]) => void];
}

export const useBobContext = (): BobContextValue => {
  const ctx = useContext(BobContext);
  if (ctx == null) throw new Error("useBobContext must be used within a BobProvider");
  return ctx;
};

export const useBobComponent = <S extends unknown>(
  type: string,
  initialState: S,
  key?: string,
  initialTransfer: Transferable[] = []
): UseBobComponentReturn<S> => {
  const oKey = useUniqueKey(key);
  const ctx = useBobContext();
  const path = [...ctx.path, oKey];
  const [state, _setState] = useState(() => {
    ctx.setState(path, type, initialState, initialTransfer);
    return initialState;
  });
  const setState = useCallback(
    (next: PsuedoSetStateArg<S>, transfer: Transferable[] = []): void => {
      if (typeof next === "function") {
        _setState((prev) => {
          const nextS = (next as (prev: S) => S)(prev);
          ctx.setState(path, type, nextS, transfer);
          return nextS;
        });
      } else {
        _setState(next);
        ctx.setState(path, type, next, transfer);
      }
    },
    [ctx, path, type]
  );
  useEffect(() => () => ctx.delete(path), []);
  return { key: oKey, path, state: [state, setState] };
};

export const useBobBootstrap = <P extends unknown>(): ((
  state: P,
  transfer?: Transferable[]
) => void) => {
  const ctx = useBobContext();
  return useCallback(
    (state: P, transfer: Transferable[] = []): void => ctx.bootstrap(state, transfer),
    [ctx]
  );
};

export interface BobProviderProps extends PropsWithChildren {
  workerKey: string;
}

export const BobProvider = ({ workerKey, children }: BobProviderProps): JSX.Element => {
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

  const bootstrap = useCallback(
    (state: any, transfer: Transferable[] = []): void =>
      worker.send({ variant: "bootstrap", data: state }, transfer),
    []
  );

  return (
    <BobContext.Provider value={{ path: [], setState, delete: delete_, bootstrap }}>
      {children}
    </BobContext.Provider>
  );
};

export interface BobCompositeProps extends PropsWithChildren {
  path: string[];
}

export const BobComposite = ({ children, path }: BobCompositeProps): JSX.Element => {
  const ctx = useBobContext();
  return <BobContext.Provider value={{ ...ctx, path }}>{children}</BobContext.Provider>;
};

export const Bob = {
  Provider: BobProvider,
  Composite: BobComposite,
  useComponent: useBobComponent,
  useBootstrap: useBobBootstrap,
};
