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
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { UnexpectedError } from "@synnaxlabs/client";
import { z } from "zod";

import { WorkerMessage } from "@/core/aether/message";
import { PsuedoSetStateArg } from "@/core/hooks/useStateRef";
import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { useTypedWorker } from "@/core/worker/Context";

export interface AetherCreateReturn {
  setState: (state: any, transfer?: Transferable[]) => void;
  delete: () => void;
}

export interface AetherContextValue {
  path: string[];
  register: (type: string, path: string[], receive: StateHandler) => AetherCreateReturn;
}

export const AetherContext = createContext<AetherContextValue | null>(null);

export interface UseAetherReturn<S extends z.ZodTypeAny> {
  key: string;
  path: string[];
  state: [
    z.output<S>,
    (state: PsuedoSetStateArg<z.output<S>>, transfer?: Transferable[]) => void
  ];
}

export const useAetherContext = (): AetherContextValue => {
  const ctx = useContext(AetherContext);
  if (ctx == null) throw new Error("useBobContext must be used within a BobProvider");
  return ctx;
};

export const useAether = <S extends z.ZodTypeAny>(
  type: string,
  initialState: z.input<S>,
  schema: S,
  key?: string,
  initialTransfer: Transferable[] = []
): UseAetherReturn<S> => {
  const oKey = useUniqueKey(key);
  const { register, path: ctxPath } = useAetherContext();
  const path = useMemo(() => [...ctxPath, oKey], [ctxPath]);

  const [internalState, setInternalState] = useState<z.output<S>>(() =>
    schema.parse(initialState)
  );

  const transferred = useRef<Transferable[]>([]);

  const comms = useRef<AetherCreateReturn | null>(null);

  const setState = useCallback(
    (next: PsuedoSetStateArg<z.output<S>>, transfer: Transferable[] = []): void => {
      if (comms.current == null) throw new UnexpectedError("Unexpected message");
      const untransferred = transfer.filter((t) => !transferred.current.includes(t));
      const { setState } = comms.current;
      transferred.current = transferred.current.concat(untransferred);
      if (typeof next === "function")
        setInternalState((prev) => {
          const nextS = (next as (prev: z.output<S>) => z.output<S>)(prev);
          setState(nextS, untransferred);
          return nextS;
        });
      else {
        setInternalState(next);
        setState(next, untransferred);
      }
    },
    [path, type]
  );

  const handleReceive = useCallback(
    (state: any) => {
      const parsed = schema.parse(state);
      setInternalState(parsed);
    },
    [schema]
  );

  if (comms.current == null) {
    comms.current = register(type, path, handleReceive);
    comms.current.setState(initialState, initialTransfer);
  }

  useEffect(() => {
    return () => {
      if (comms.current == null) throw new UnexpectedError("Unexpected message");
      comms.current.delete();
    };
  }, []);

  return { key: oKey, path, state: [internalState, setState] };
};

export interface AetherProviderProps extends PropsWithChildren {
  workerKey: string;
}

type StateHandler = (state: any) => void;

interface RegisteredComponent {
  path: string[];
  handler: StateHandler;
}

export const AetherProvider = ({
  workerKey,
  children,
}: AetherProviderProps): ReactElement => {
  const worker = useTypedWorker<WorkerMessage>(workerKey);
  const registry = useRef<Map<string, RegisteredComponent>>(new Map());

  const register: AetherContextValue["register"] = useCallback(
    (type, path, handler) => {
      const key = path[path.length - 1];
      registry.current.set(key, { path, handler });
      return {
        setState: (state, transfer) =>
          worker.send({ variant: "update", path, state, type }, transfer),
        delete: () => worker.send({ variant: "delete", path }),
      };
    },
    [worker, registry]
  );

  useEffect(() => {
    worker.handle((msg) => {
      if (msg.variant !== "backward") throw new UnexpectedError("Unexpected message");
      const { key, state } = msg;
      const component = registry.current.get(key);
      if (component == null) throw new UnexpectedError("Unexpected message");
      component.handler(state);
    });
  }, [worker]);

  const value = useMemo<AetherContextValue>(() => ({ register, path: [] }), [register]);

  return <AetherContext.Provider value={value}>{children}</AetherContext.Provider>;
};

export interface AetherCompositeProps extends PropsWithChildren {
  path: string[];
}

export const AetherComposite = ({
  children,
  path,
}: AetherCompositeProps): JSX.Element => {
  const ctx = useAetherContext();
  const value = useMemo<AetherContextValue>(() => ({ ...ctx, path }), [ctx, path]);
  return <AetherContext.Provider value={value}>{children}</AetherContext.Provider>;
};

export const Aether = {
  Provider: AetherProvider,
  Composite: AetherComposite,
  use: useAether,
};
