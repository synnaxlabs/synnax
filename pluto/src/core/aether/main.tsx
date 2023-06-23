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
  memo,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { z } from "zod";

import { MainMessage, WorkerMessage } from "@/core/aether/message";
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

export type UseAetherReturn<S extends z.ZodTypeAny> = [
  {
    key: string;
    path: string[];
  },
  z.output<S>,
  (state: PsuedoSetStateArg<z.input<S>>, transfer?: Transferable[]) => void
];

export const useAetherContext = (): AetherContextValue => {
  const ctx = useContext(AetherContext);
  if (ctx == null) throw new Error("useBobContext must be used within a BobProvider");
  return ctx;
};

export const useAether = <S extends z.ZodTypeAny>(
  type: string,
  schema: S,
  initialState: z.input<S>,
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

  const commsRef = useRef<AetherCreateReturn | null>(null);

  const setState = useCallback(
    (next: PsuedoSetStateArg<z.input<S>>, transfer: Transferable[] = []): void => {
      const untransferred = transfer.filter((t) => !transferred.current.includes(t));
      const comms = commsRef.current as AetherCreateReturn;
      transferred.current = transferred.current.concat(untransferred);
      if (typeof next === "function")
        setInternalState((prev) => {
          const nextS = schema.parse((next as (prev: z.output<S>) => z.input<S>)(prev));
          comms.setState(schema.parse(nextS), untransferred);
          return nextS;
        });
      else {
        setInternalState(next);
        comms.setState(next, untransferred);
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

  if (commsRef.current == null) {
    commsRef.current = register(type, path, handleReceive);
    commsRef.current.setState(initialState, initialTransfer);
  }

  useEffect(() => {
    return () => {
      if (commsRef.current == null) throw new UnexpectedError("Unexpected message");
      commsRef.current.delete();
    };
  }, []);

  return [{ key: oKey, path }, internalState, setState];
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
  const worker = useTypedWorker<MainMessage, WorkerMessage>(workerKey);
  const registry = useRef<Map<string, RegisteredComponent>>(new Map());

  const register: AetherContextValue["register"] = useCallback(
    (type, path, handler) => {
      const key = path.at(-1);
      if (key == null)
        throw new ValidationError(
          `[Aether.Provider] - received zero length path when registering component of type ${type}`
        );
      if (type.length === 0)
        console.warn(
          `[Aether.Provider] - received zero length type when registering component at ${path.join(
            "."
          )} This is probably a bad idea.`
        );
      registry.current.set(key, { path, handler });
      return {
        setState: (state, transfer) =>
          worker.send({ variant: "update", path, state, type }, transfer),
        delete: () => worker.send({ variant: "delete", path }),
      };
    },
    [worker, registry]
  );

  useEffect(
    () =>
      worker.handle((msg) => {
        const { key, state } = msg;
        const component = registry.current.get(key);
        if (component == null)
          throw new UnexpectedError(
            `[Aether.Provider] - received worker update message for unregistered component with key ${key}`
          );
        component.handler(state);
      }),
    [worker]
  );

  const value = useMemo<AetherContextValue>(() => ({ register, path: [] }), [register]);

  return <AetherContext.Provider value={value}>{children}</AetherContext.Provider>;
};

export interface AetherCompositeProps extends PropsWithChildren {
  path: string[];
}

export const AetherComposite = memo(
  ({ children, path }: AetherCompositeProps): JSX.Element => {
    const ctx = useAetherContext();
    const value = useMemo<AetherContextValue>(() => ({ ...ctx, path }), [ctx, path]);
    return <AetherContext.Provider value={value}>{children}</AetherContext.Provider>;
  }
);
AetherComposite.displayName = "AetherComposite";

export const Aether = {
  Provider: AetherProvider,
  Composite: AetherComposite,
  use: useAether,
};
