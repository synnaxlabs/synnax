// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ComponentType,
  FC,
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
import { Compare } from "@synnaxlabs/x";
import { z } from "zod";

import { useMemoCompare } from "@/core/memo";
import { MainMessage, WorkerMessage } from "@/core/aether/message";
import { PsuedoSetStateArg, isStateSetter } from "@/core/hooks/useStateRef";
import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { Worker } from "@/core/worker";

export interface AetherCreateReturn {
  setState: (state: any, transfer?: Transferable[]) => void;
  delete: () => void;
}

export interface AetherContextValue {
  path: string[];
  create: (
    type: string,
    path: string[],
    onReceive?: StateHandler
  ) => AetherCreateReturn;
}

export const AetherContext = createContext<AetherContextValue | null>(null);

export const AetherProvider = ({
  workerKey,
  children,
}: AetherProviderProps): ReactElement => {
  const worker = Worker.use<MainMessage, WorkerMessage>(workerKey);
  const registry = useRef<Map<string, RegisteredComponent>>(new Map());
  const [ready, setReady] = useState(false);

  const create: AetherContextValue["create"] = useCallback(
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
        setState: (state: any, transfer: Transferable[] = []): void =>
          worker?.send({ variant: "update", path, state, type }, transfer),
        delete: () => worker?.send({ variant: "delete", path }),
      };
    },
    [worker, registry]
  );

  useEffect(() => {
    worker?.handle((msg) => {
      const { key, state } = msg;
      const component = registry.current.get(key);
      if (component == null)
        throw new UnexpectedError(
          `[Aether.Provider] - received worker update message for unregistered component with key ${key}`
        );
      if (component.handler == null)
        throw new UnexpectedError(
          `[Aether.Provider] - received worker update message for component with key ${key} that has no handler`
        );
      component.handler(state);
    });
    setReady(true);
  }, [worker]);

  const value = useMemo<AetherContextValue>(
    () => ({ create, path: ["root"] }),
    [create]
  );

  return (
    <AetherContext.Provider value={value}>{ready && children}</AetherContext.Provider>
  );
};

export const useAetherContext = (): AetherContextValue => {
  const ctx = useContext(AetherContext);
  if (ctx == null) throw new Error("useBobContext must be used within a BobProvider");
  return ctx;
};

export interface UseAetherLifecycleReturn<S extends z.ZodTypeAny> {
  path: string[];
  setState: (state: z.input<S>, transfer?: Transferable[]) => void;
}

export interface UseAetherLifecycleProps<S extends z.ZodTypeAny> {
  type: string;
  schema: S;
  aetherKey: string;
  initialState: z.input<S>;
  initialTransfer?: Transferable[];
  onReceive?: StateHandler;
}

const useAetherLifecycle = <S extends z.ZodTypeAny>({
  type,
  aetherKey: key,
  initialState,
  schema,
  initialTransfer = [],
  onReceive,
}: UseAetherLifecycleProps<S>): UseAetherLifecycleReturn<S> => {
  const comms = useRef<AetherCreateReturn | null>();
  const ctx = useAetherContext();
  const path = useMemoCompare(
    () => [...ctx.path, key],
    ([a], [b]) => Compare.primitiveArrays(a, b) === 0,
    [ctx.path, key] as [string[], string]
  );

  const setState = useCallback((state: z.input<S>, transfer: Transferable[] = []) => {
    comms.current?.setState(schema.parse(state), transfer);
  }, []);

  // We run the first effect synchronously so that parent components are created
  // before their children. This is impossible to do with effect hooks.
  if (comms.current == null) {
    comms.current = ctx.create(type, path, onReceive);
    comms.current.setState(initialState, initialTransfer);
  }

  // We run this effect whenever the identity of the aether component
  // we're using changes i.e. when the type or path changes. We also
  // run the effect when the onReceive callback changes, to make sure
  // that the callback is up to date. We don't run the effect when the
  // initialState or initialTransfer change, because this state is INITIAL.
  const first = useRef(true);
  useEffect(() => {
    // If we're on the first execution of the effect, we've already sent
    // what we need do synchronously, so don't re-execute.
    if (first.current) {
      first.current = false;
      return;
    }
    comms.current = ctx.create(type, path, onReceive);
    setState(initialState, initialTransfer);
    return () => comms.current?.delete();
  }, [type, path, onReceive, setState]);

  return useMemo(() => ({ setState, path }), [setState, key, path]);
};

export interface UseAetherProps<S extends z.ZodTypeAny>
  extends Omit<UseAetherLifecycleProps<S>, "onReceive"> { }

export type UseAetherReturn<S extends z.ZodTypeAny> = [
  {
    path: string[];
  },
  z.output<S>,
  (state: PsuedoSetStateArg<z.input<S>>, transfer?: Transferable[]) => void
];

const useAether = <S extends z.ZodTypeAny>(
  props: UseAetherProps<S>
): UseAetherReturn<S> => {
  const { type, schema, initialState } = props;

  const [internalState, setInternalState] = useState<z.output<S>>(() =>
    schema.parse(initialState)
  );

  // Update the internal component state when we receive communications from the
  // aether.
  const handleReceive = useCallback(
    (state: any) => setInternalState(schema.parse(state)),
    [schema]
  );

  const { path, setState: setAetherState } = useAetherLifecycle({
    ...props,
    onReceive: handleReceive,
  });

  const setState = useCallback(
    (
      next: PsuedoSetStateArg<z.input<S> | z.output<S>>,
      transfer: Transferable[] = []
    ): void => {
      if (isStateSetter(next))
        setInternalState((prev) => {
          const nextS = next(prev);
          // This makes our setter impure, so it's something we should be wary of causing
          // unexpected behaviour in the the future.
          setAetherState(nextS, transfer);
          return nextS;
        });
      else {
        setInternalState(next);
        setAetherState(next, transfer);
      }
    },
    [path, type]
  );

  return [{ path }, internalState, setState];
};

export interface AetherProviderProps extends PropsWithChildren {
  workerKey: string;
}

type StateHandler = (state: any) => void;

interface RegisteredComponent {
  path: string[];
  handler?: StateHandler;
}

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

const wrap = <P extends {}>(
  displayName: string,
  Component: ComponentType<P & { aetherKey: string }>
): FC<P> => {
  Component.displayName = `Aether.wrap(${displayName})`;
  const Wrapped = (props: P): JSX.Element => {
    const key = useUniqueKey();
    return <Component {...props} aetherKey={key} />;
  };
  Wrapped.displayName = displayName;
  return Wrapped;
};

export const Aether = {
  Provider: AetherProvider,
  Composite: AetherComposite,
  use: useAether,
  wrap,
};
