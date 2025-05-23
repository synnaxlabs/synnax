// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { deep, type SenderHandler } from "@synnaxlabs/x";
import { compare } from "@synnaxlabs/x/compare";
import {
  createContext,
  memo,
  type PropsWithChildren,
  type ReactElement,
  use as reactUse,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import { type MainMessage, type WorkerMessage } from "@/aether/message";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useMemoCompare } from "@/memo";
import { state } from "@/state";
import { prettyParse } from "@/util/zod";
import { Worker } from "@/worker";

export interface CreateReturn {
  setState: (state: any, transfer?: Transferable[]) => void;
  delete: () => void;
}

export interface ContextValue {
  path: string[];
  create: (type: string, path: string[], onReceive?: StateHandler) => CreateReturn;
}

const ZERO_CONTEXT_VALUE = {
  path: [],
  create: () => ({ setState: () => {}, delete: () => {} }),
};

const Context = createContext<ContextValue>(ZERO_CONTEXT_VALUE);

export interface ProviderProps extends PropsWithChildren {
  workerKey: string;
  worker?: SenderHandler<MainMessage, WorkerMessage>;
}

export const Provider = ({
  workerKey,
  worker: propsWorker,
  children,
}: ProviderProps): ReactElement => {
  const contextWorker = Worker.use<MainMessage, WorkerMessage>(workerKey);
  const registry = useRef<Map<string, RegisteredComponent>>(new Map());
  const [ready, setReady] = useState(false);
  const worker = useMemo(
    () => propsWorker ?? contextWorker,
    [propsWorker, contextWorker],
  );

  const create: ContextValue["create"] = useCallback(
    (type, path, handler) => {
      const key = path.at(-1);
      if (key == null)
        throw new ValidationError(
          `[Aether.Provider] - received zero length path when registering component of type ${type}`,
        );
      if (type.length === 0)
        console.warn(
          `[Aether.Provider] - received zero length type when registering component at ${path.join(".")} This is probably a bad idea.`,
        );
      registry.current.set(key, { path, handler });
      return {
        setState: (state: any, transfer: Transferable[] = []): void => {
          if (worker == null) console.warn("aether - no worker");
          worker?.send({ variant: "update", path, state, type }, transfer);
        },
        delete: () => {
          if (worker == null) console.warn("aether - no worker");
          worker?.send({ variant: "delete", path, type });
        },
      };
    },
    [worker, registry],
  );

  const [error, setError] = useState<Error | null>(null);
  if (error != null) throw error;

  useEffect(() => {
    worker?.handle((msg) => {
      const { variant } = msg;
      if (variant == "error") {
        const {
          error: { message, stack, name },
        } = msg;
        const err = new Error(message);
        err.stack = stack;
        err.name = name;
        setError(err);
        return;
      }
      const { key, state } = msg;
      const component = registry.current.get(key);
      if (component == null)
        throw new UnexpectedError(
          `[Aether.Provider] - received worker update message for unregistered component with key ${key}`,
        );
      if (component.handler == null)
        throw new UnexpectedError(
          `[Aether.Provider] - received worker update message for component with key ${key} that has no handler`,
        );
      component.handler(state);
    });
    setReady(true);
  }, [worker]);

  const value = useMemo<ContextValue>(() => ({ create, path: ["root"] }), [create]);

  return <Context value={value}>{ready && children}</Context>;
};

export const useContext = () => reactUse(Context);

export interface UseLifecycleReturn<S extends z.ZodTypeAny> {
  path: string[];
  setState: (state: z.input<S>, transfer?: Transferable[]) => void;
}

interface UseLifecycleProps<S extends z.ZodTypeAny> {
  type: string;
  schema: S;
  aetherKey?: string;
  initialState: z.input<S>;
  initialTransfer?: Transferable[];
  onReceive?: StateHandler;
}

/**
 * A low-level hook that manages the lifecycle of an Aether component. This hook handles
 * component creation, state updates, and cleanup in the Aether worker architecture.
 *
 * @template S - A Zod schema type that defines the shape and validation of the component's state
 *
 * @param props - Configuration options for the lifecycle hook
 * @param props.type - A string identifier for the component type. Must correspond to a registered
 *                     component type in the worker thread's registry.
 * @param props.schema - A Zod schema that defines the shape and validation of the component's state.
 *                       Used to validate state during transfer between main and worker threads.
 * @param props.aetherKey - Optional unique identifier for the component. If not provided,
 *                         a unique key will be generated automatically.
 * @param props.initialState - The initial state value that conforms to the provided schema.
 * @param props.initialTransfer - Optional array of Transferable objects to be transferred
 *                               along with the initial state.
 * @param props.onReceive - Optional callback function that handles state updates received
 *                         from the worker thread.
 *
 * @returns An object containing:
 *          - path: string[] - The full path to this component in the Aether component tree
 *          - setState: (state: z.input<S>, transfer?: Transferable[]) => void - A function
 *            to update the component's state and optionally transfer objects to the worker
 *
 * @example
 * ```typescript
 * const MyComponent = () => {
 *   const { path, setState } = useLifecycle({
 *     type: "MyComponentType",
 *     schema: z.object({ count: z.number() }),
 *     initialState: { count: 0 },
 *     onReceive: (state) => console.log("Received state:", state)
 *   });
 *
 *   return <button onClick={() => setState({ count: count + 1 })}>Increment</button>;
 * };
 * ```
 *
 * @remarks
 * - This hook must be used within an Aether.Provider context
 * - The hook ensures proper cleanup on component unmount
 * - State updates are validated against the provided schema before being sent to the worker
 * - The hook uses synchronous initialization to ensure parent components are created
 *   before their children
 * - Changes to initialState after the first render will not affect the component's state.
 *   Use setState to update the state instead.
 */
export const useLifecycle = <S extends z.ZodTypeAny>({
  type,
  aetherKey,
  initialState,
  schema,
  initialTransfer = [],
  onReceive,
}: UseLifecycleProps<S>): UseLifecycleReturn<S> => {
  const key = useUniqueKey(aetherKey);
  const comms = useRef<CreateReturn | null>(null);
  const ctx = useContext();
  const path = useMemoCompare(
    () => [...ctx.path, key],
    ([a], [b]) => compare.primitiveArrays(a, b) === 0,
    [ctx.path, key] as [string[], string],
  );

  const setState = useCallback(
    (state: z.input<S>, transfer: Transferable[] = []) =>
      comms.current?.setState(prettyParse(schema, state), transfer),
    [],
  );

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
    return () => {
      comms.current?.delete();
      comms.current = null;
    };
  }, [type, path, onReceive, setState]);

  // Destroy the component on unmount.
  useLayoutEffect(
    () => () => {
      comms.current?.delete();
      comms.current = null;
    },
    [],
  );

  return useMemo(() => ({ setState, path }), [setState, key, path]);
};

export interface CProps {
  aetherKey?: string;
}

export interface UseProps<S extends z.ZodTypeAny>
  extends Omit<UseLifecycleProps<S>, "onReceive"> {
  onAetherChange?: (state: z.output<S>) => void;
}

interface ComponentContext {
  path: string[];
}

export type UseReturn<S extends z.ZodTypeAny> = [
  ComponentContext,
  z.output<S>,
  (state: state.SetArg<z.input<S>>, transfer?: Transferable[]) => void,
];

export interface UseUnidirectionalProps<S extends z.ZodTypeAny>
  extends Pick<UseLifecycleProps<S>, "schema" | "aetherKey"> {
  type: string;
  state: z.input<S>;
}

/***
 * A simpler version of {@link use} that assumes the caller only wants to propagate
 * state to the aether component, and not receive state from the aether component.
 */
export const useUnidirectional = <S extends z.ZodTypeAny>({
  state,
  ...rest
}: UseUnidirectionalProps<S>): ComponentContext => {
  const { path, setState } = useLifecycle({ ...rest, initialState: state });
  const ref = useRef<z.output<S> | null>(null);
  if (!deep.equal(ref.current, state)) {
    ref.current = state;
    setState(state);
  }
  return { path };
};

/**
 * Use creates a new aether component with a unique key and type.
 *
 * @param props.type - The type of the component. Aether will use this to instantiate
 * the correct component type on the worker thread. A constructor for this type must
 * exist on the registry passed into aether.render on the worker thread.
 * @param props.aetherKey - A unique key for the component that is used to identify it
 * in the aether component tree. We recommend using `Aether.wrap` to create unique keys.
 * @param props.schema - The zod schema for the component's state. Used to validate state
 * changes on transfer to and from the worker thread.
 * @param props.initialState - The initial state for the component. Note that this state
 * is only used on the first render, and any changes to this value will not be reflected
 * in the component's state. To alter the component's state, use the setState function
 * returned by the hook.
 * @returns A triplet with the following values:
 * 1. An object containing metadata about the created component. This object contains a
 *    path property, which is an array of strings representing the path to the component
 *    in the aether component tree.
 * 2. The component's current state. This is synchronized with the worker thread, and
 * can be updated by both the worker and the main thread.
 * 3. A function that can be used to update the component's state. This function takes
 * in both the next state and an optional array of transferable objects. The next state
 * can be either a new state, or a function that takes in the current state and returns
 * the next state. This function is impure, and will update the component's state on the
 * worker thread.
 */
export const use = <S extends z.ZodTypeAny>(props: UseProps<S>): UseReturn<S> => {
  const { type, schema, initialState, onAetherChange } = props;
  const [internalState, setInternalState] = useState<z.output<S>>(() =>
    prettyParse(schema, initialState),
  );
  const onAetherChangeRef = useRef(onAetherChange);

  // Update the internal component state when we receive communications from the
  // aether.
  const handleReceive = useCallback(
    (rawState: any) => {
      const state = prettyParse(schema, rawState);
      setInternalState(state);
      onAetherChangeRef.current?.(state);
    },
    [schema, setInternalState],
  );

  const { path, setState: setAetherState } = useLifecycle({
    ...props,
    onReceive: handleReceive,
  });

  const setState = useCallback(
    (
      next: state.SetArg<z.input<S> | z.output<S>>,
      transfer: Transferable[] = [],
    ): void => {
      if (state.isSetter(next))
        setInternalState((prev) => {
          const nextS = next(prev);
          setAetherState(nextS, transfer);
          return nextS;
        });
      else {
        setInternalState(next);
        setAetherState(next, transfer);
      }
    },
    [path, type],
  );

  return [{ path }, internalState, setState];
};

type StateHandler = (state: any) => void;

interface RegisteredComponent {
  path: string[];
  handler?: StateHandler;
}

export interface CompositeProps extends PropsWithChildren {
  path: string[];
}

/**
 * Composite establishes all children as child components of the current component. The
 * provide path should match the path returned by the {@link use} hook. Any children
 * of this component will be added to the `children` property of the component on the
 * worker tree.
 *
 * Naturally, composites can be nested within each other. Child components that are
 */
export const Composite = memo(({ children, path }: CompositeProps): ReactElement => {
  const ctx = useContext();
  const value = useMemo<ContextValue>(() => ({ ...ctx, path }), [ctx, path]);
  return <Context value={value}>{children}</Context>;
});
Composite.displayName = "AetherComposite";
