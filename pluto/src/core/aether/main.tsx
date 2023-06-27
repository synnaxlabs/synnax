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
import { PsuedoSetStateArg, isStateSetter } from "@/core/hooks/useStateRef";
import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { useTypedWorker } from "@/core/worker/Context";

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

export interface UseAetherLifecycleReturn<S extends z.ZodTypeAny> {
  key: string;
  path: string[];
  setState: (state: z.input<S>, transfer?: Transferable[]) => void;
}

interface UseAetherLifecycleRef extends AetherCreateReturn {
  first: boolean;
}

export interface UseAetherProps<S extends z.ZodTypeAny> {
  type: string;
  schema: S;
  key?: string;
  initialState?: z.input<S>;
  initialTransfer?: Transferable[];
  onReceive?: StateHandler;
}

export const useAether = <S extends z.ZodTypeAny>({
  type,
  key: maybeKey,
  initialState,
  schema,
  initialTransfer = [],
  onReceive,
}: UseAetherProps<S>): UseAetherLifecycleReturn<S> => {
  const key = useUniqueKey(maybeKey);
  const comms = useRef<UseAetherLifecycleRef | null>(null);
  const ctx = useAetherContext();
  const path = useMemo(() => [...ctx.path, key], [ctx.path, key]);

  const setState = useCallback((state: z.input<S>, transfer: Transferable[] = []) => {
    if (comms.current == null) return;
    comms.current.setState(schema.parse(state), transfer);
  }, []);

  // Delete the aether component when the component is unmounted.
  useEffect(() => () => comms.current?.delete(), []);

  // We run this effect whenever the identity of the aether component
  // we're using changes i.e. when the type or path changes. We also
  // run the effect when the onReceive callback changes, to make sure
  // that the callback is up to date. We don't run the effect when the
  // initialState or initialTransfer change, because this state is INITIAL.
  useEffect(() => {
    // If we have no comms to the aether component, or
    // we're on the first execution of the effect, do nothing.
    if (comms.current == null) return;
    if (comms.current.first) {
      comms.current.first = false;
      return;
    }
    comms.current.delete();
    comms.current = {
      ...ctx.create(type, path, onReceive),
      first: false,
    };
    setState(initialState, initialTransfer);
  }, [type, path, onReceive, setState]);

  // We run the first effect synchronously so that parent components are created
  // before their children. This is impossible to do with a useEffect or useLayoutEffect
  // hook.
  if (comms.current == null) {
    comms.current = {
      ...ctx.create(type, path, schema.parse(initialState)),
      first: true,
    };
    setState(initialState, initialTransfer);
  }

  return useMemo(
    () => ({ ...(comms.current as UseAetherLifecycleRef), setState, key, path }),
    [comms.current, setState, key, path]
  );
};

export interface UseAetherStateProps<S extends z.ZodTypeAny>
  extends Omit<UseAetherProps<S>, "onReceive"> {}

export const useStatefulAether = <S extends z.ZodTypeAny>({
  type,
  schema,
  initialState,
  key: maybeKey,
  initialTransfer,
}: UseAetherStateProps<S>): UseAetherReturn<S> => {
  const [internalState, setInternalState] = useState<z.output<S>>(() =>
    schema.parse(initialState)
  );

  // Update the internal component state when we receive communications from the
  // aether.
  const handleReceive = useCallback(
    (state: any) => setInternalState(schema.parse(state)),
    [schema]
  );

  const { path, key, ...comms } = useAether({
    type,
    key: maybeKey,
    schema,
    initialState,
    initialTransfer,
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
          comms.setState(nextS, transfer);
          return nextS;
        });
      else {
        setInternalState(next);
        comms.setState(next, transfer);
      }
    },
    [path, type]
  );

  return [{ key, path }, internalState, setState];
};

export interface AetherProviderProps extends PropsWithChildren {
  workerKey: string;
}

type StateHandler = (state: any) => void;

interface RegisteredComponent {
  path: string[];
  handler?: StateHandler;
}

export const AetherProvider = ({
  workerKey,
  children,
}: AetherProviderProps): ReactElement => {
  const worker = useTypedWorker<MainMessage, WorkerMessage>(workerKey);
  const registry = useRef<Map<string, RegisteredComponent>>(new Map());

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

  useEffect(
    () =>
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
      }),
    [worker]
  );

  const value = useMemo<AetherContextValue>(() => ({ create, path: [] }), [create]);

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
  useStateful: useStatefulAether,
  use: useAether,
};
