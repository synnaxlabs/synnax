// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { compare, deep, type errors, type SenderHandler } from "@synnaxlabs/x";
import {
  memo,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import {
  type CallersFromSchema,
  type EmptyMethodsSchema,
  isFireAndForget,
  type MethodsSchema,
} from "@/aether/aether/aether";
import { type AetherMessage, type MainMessage } from "@/aether/message";
import { context } from "@/context";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useMemoCompare } from "@/memo";
import { type state } from "@/state";
import { prettyParse } from "@/util/zod";
import { Worker } from "@/worker";

export type { CallersFromSchema, EmptyMethodsSchema, MethodsSchema };

type RawSetArg<S extends z.ZodType<state.State>> =
  | (z.input<S> | z.infer<S>)
  | ((prev: z.infer<S>) => z.input<S> | z.infer<S>);

/** Default invoke timeout in milliseconds */
const DEFAULT_INVOKE_TIMEOUT = 5000;

const reconstructError = (payload: errors.NativePayload): Error => {
  const err = new Error(payload.message);
  err.name = payload.name;
  err.stack = payload.stack;
  return err;
};

interface PendingRequest {
  resolve: (value: unknown) => void;
  reject: (error: Error) => void;
  controller: AbortController;
}

/** Manages pending invoke requests with AbortController-based cancellation. */
class InvokeTracker {
  private pending = new Map<string, PendingRequest>();
  private counters = new Map<string, number>();

  nextkey(key: string): string {
    const counter = this.counters.get(key) ?? 0;
    this.counters.set(key, counter + 1);
    return `${key}-${counter}`;
  }

  track(
    key: string,
    resolve: (value: unknown) => void,
    reject: (error: Error) => void,
    signal: AbortSignal,
  ): void {
    const controller = new AbortController();
    // Link the external signal to our internal controller
    signal.addEventListener("abort", () => controller.abort(signal.reason), {
      signal: controller.signal,
    });
    controller.signal.addEventListener("abort", () => {
      this.pending.delete(key);
      reject(controller.signal.reason);
    });
    this.pending.set(key, { resolve, reject, controller });
  }

  resolve(key: string, result: unknown, error?: errors.NativePayload): boolean {
    const pending = this.pending.get(key);
    if (pending == null) return false;
    this.pending.delete(key);
    if (error != null) pending.reject(reconstructError(error));
    else pending.resolve(result);
    return true;
  }

  abort(reason: Error): void {
    this.pending.forEach(({ controller }) => controller.abort(reason));
    this.pending.clear();
  }

  clearCounter(key: string): void {
    this.counters.delete(key);
  }
}

/**
 * return value of the create function in the Aether context.
 * Provides methods to manage the lifecycle of an Aether component.
 */
export interface CreateReturn {
  /**
   * Updates the state of the component and optionally transfers objects to the worker thread.
   * @param state - The new state to set on the component
   * @param transfer - Optional array of Transferable objects to be transferred to the worker
   */
  setState: (state: state.State, transfer?: Transferable[]) => void;

  /**
   * Deletes the component from the Aether tree, triggering cleanup
   * on the worker thread.
   */
  delete: () => void;

  /**
   * Invokes a method on the worker component (fire-and-forget).
   * @param method - The name of the method to invoke
   * @param args - Arguments to pass to the method (spread to handler)
   */
  invokeMethod: (method: string, args: unknown[]) => void;

  /**
   * Invokes a method on the worker component and waits for the response.
   * @param method - The name of the method to invoke
   * @param args - Arguments to pass to the method (spread to handler)
   * @param signal - Optional AbortSignal for cancellation/timeout (default: 5s timeout)
   * @returns A Promise that resolves with the method's return value
   */
  invokeMethodAsync: <R>(
    method: string,
    args: unknown[],
    signal?: AbortSignal,
  ) => Promise<R>;
}

/**
 * Interface representing the value provided by the Aether context.
 * Used to create and manage Aether components in the component tree.
 */
export interface ContextValue {
  /** The current path in the Aether component tree */
  path: string[];
  /**
   * Creates a new Aether component in the tree
   * @param type - The type identifier for the component
   * @param path - The path where the component should be created
   * @param onReceive - Optional callback for handling state updates from the worker
   * @returns An object with methods to manage the component's lifecycle
   */
  create: (type: string, path: string[], onReceive?: StateHandler) => CreateReturn;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: {
    create: () => ({
      setState: () => {},
      delete: () => {},
      invokeMethod: () => {},
      invokeMethodAsync: () => Promise.reject(new Error("No Aether provider")),
    }),
    path: [],
  },
  displayName: "Aether.Context",
});

/**
 * Props for the Aether Provider component that establishes the Aether context.
 */
export interface ProviderProps extends PropsWithChildren {
  /** Unique identifier for the worker instance */
  workerKey: string;

  /** Optional worker handler for managing communication between main and worker threads */
  worker?: SenderHandler<MainMessage, AetherMessage>;
}

export const Provider = ({
  workerKey,
  worker: propsWorker,
  children,
}: ProviderProps): ReactElement => {
  const contextWorker = Worker.use<MainMessage, AetherMessage>(workerKey);
  const registry = useRef<Map<string, RegisteredComponent>>(new Map());
  const [ready, setReady] = useState(false);
  const worker = useMemo(
    () => propsWorker ?? contextWorker,
    [propsWorker, contextWorker],
  );
  const invokeTracker = useRef(new InvokeTracker());

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
      const controller = new AbortController();
      return {
        setState: (state: state.State, transfer: Transferable[] = []): void => {
          if (worker == null) console.warn("aether - no worker");
          worker?.send({ variant: "update", path, state, type }, transfer);
        },
        delete: () => {
          controller.abort(new Error("Component deleted"));
          invokeTracker.current.clearCounter(key);
          if (worker == null) console.warn("aether - no worker");
          worker?.send({ variant: "delete", path, type });
        },
        invokeMethod: (method: string, args: unknown[]): void => {
          worker?.send({
            variant: "invoke_request",
            key: "",
            path,
            method,
            args,
            expectsResponse: false,
          });
        },
        invokeMethodAsync: <R,>(
          method: string,
          args: unknown[],
          signal: AbortSignal = AbortSignal.timeout(DEFAULT_INVOKE_TIMEOUT),
        ): Promise<R> =>
          new Promise((resolve, reject) => {
            if (worker == null) return reject(new Error("aether - no worker"));
            if (controller.signal.aborted)
              return reject(new Error("Component deleted"));
            const invokeKey = invokeTracker.current.nextkey(key);
            invokeTracker.current.track(
              invokeKey,
              resolve as (v: unknown) => void,
              reject,
              AbortSignal.any([signal, controller.signal]),
            );
            worker.send({
              variant: "invoke_request",
              key: invokeKey,
              path,
              method,
              args,
              expectsResponse: true,
            });
          }),
      };
    },
    [worker],
  );

  const [error, setError] = useState<Error | null>(null);
  if (error != null) throw error;

  useEffect(() => {
    worker?.handle((msg) => {
      const { variant } = msg;
      if (variant === "error") {
        const err = reconstructError(msg.error);
        setError((prev) => {
          if (prev != null) {
            console.error(
              "[aether] - received new error after error was already set, but before previous error was thrown.",
            );
            console.error(err);
          }
          return err;
        });
        return;
      }
      if (variant === "invoke_response") {
        invokeTracker.current.resolve(msg.key, msg.result, msg.error);
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

export interface UseLifecycleReturn<
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
> {
  path: string[];
  setState: (state: RawSetArg<S>, transfer?: Transferable[]) => void;
  /** Typed methods object for invoking methods on the worker component */
  methods: CallersFromSchema<M>;
}

interface UseLifecycleProps<
  S extends z.ZodType,
  M extends MethodsSchema = EmptyMethodsSchema,
> {
  type: string;
  schema: S;
  aetherKey?: string;
  initialState: z.input<S>;
  initialTransfer?: Transferable[];
  onReceive?: StateHandler<z.infer<S>>;
  /** Methods schema for Main → Worker invoke calls */
  methods?: M;
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
export const useLifecycle = <
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
>({
  type,
  aetherKey,
  initialState,
  schema,
  initialTransfer = [],
  onReceive,
  methods: methodsSchema,
}: UseLifecycleProps<S, M>): UseLifecycleReturn<S, M> => {
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
    [schema],
  );

  const methods = useMemo(() => {
    if (methodsSchema == null) return {} as CallersFromSchema<M>;
    const callers: Record<string, (...args: unknown[]) => unknown> = {};
    for (const method of Object.keys(methodsSchema))
      callers[method] = isFireAndForget(methodsSchema[method])
        ? (...args: unknown[]) => comms.current?.invokeMethod(method, args)
        : (...args: unknown[]) => comms.current?.invokeMethodAsync(method, args);

    return callers as CallersFromSchema<M>;
  }, [methodsSchema]);

  // We run the first effect synchronously so that parent components are created
  // before their children. This is impossible to do with effect hooks.
  if (comms.current == null) {
    comms.current = ctx.create(type, path, onReceive as StateHandler<unknown>);
    comms.current.setState(prettyParse(schema, initialState), initialTransfer);
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
    comms.current = ctx.create(type, path, onReceive as StateHandler);
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

  return useMemo(
    () => ({ setState, path, methods }),
    [setState, path, methods],
  ) as UseLifecycleReturn<S, M>;
};

/** Props for components that use Aether functionality */
export interface ComponentProps {
  /** Optional unique identifier for the Aether component */
  aetherKey?: string;
}

/** Props for the use hook that manages Aether component lifecycle */
export interface UseProps<
  S extends z.ZodType,
  M extends MethodsSchema = EmptyMethodsSchema,
> extends Omit<UseLifecycleProps<S, M>, "onReceive"> {
  /** Optional callback for handling state changes from the Aether component */
  onAetherChange?: (state: z.infer<S>) => void;
}

interface ComponentContext {
  path: string[];
}

/**
 * Return type for the use hook, providing access to component context, state, state setter, and methods
 */
export type UseReturn<
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
> = [
  ComponentContext,
  z.infer<S>,
  (state: RawSetArg<S>, transfer?: Transferable[]) => void,
  CallersFromSchema<M>,
];

/**
 * Props for the useUnidirectional hook that only propagates state to the Aether component
 */
export interface UseUnidirectionalProps<
  S extends z.ZodType,
  M extends MethodsSchema = EmptyMethodsSchema,
> extends Pick<UseLifecycleProps<S, M>, "schema" | "aetherKey" | "methods"> {
  /** The type identifier for the Aether component */
  type: string;
  /** The current state to propagate to the Aether component */
  state: z.input<S>;
}

export interface UseUnidirectionalReturn<M extends MethodsSchema = EmptyMethodsSchema>
  extends ComponentContext {
  methods: CallersFromSchema<M>;
}

/***
 * A simpler version of {@link use} that assumes the caller only wants to propagate
 * state to the aether component, and not receive state from the aether component.
 */
export const useUnidirectional = <
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
>({
  state,
  ...rest
}: UseUnidirectionalProps<S, M>): UseUnidirectionalReturn<M> => {
  const { path, setState, methods } = useLifecycle<S, M>({
    ...rest,
    initialState: state,
  });
  const ref = useRef<z.input<S> | z.infer<S> | null>(null);
  if (!deep.equal(ref.current, state)) {
    ref.current = state;
    setState(state);
  }
  return { path, methods };
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
export const use = <
  S extends z.ZodType<state.State>,
  M extends MethodsSchema = EmptyMethodsSchema,
>(
  props: UseProps<S, M>,
): UseReturn<S, M> => {
  const { type, schema, initialState, onAetherChange } = props;
  const [internalState, setInternalState] = useState<z.infer<S>>(() =>
    prettyParse(schema, initialState),
  );
  const onAetherChangeRef = useRef(onAetherChange);

  // Update the internal component state when we receive communications from the
  // aether.
  const handleReceive = useCallback(
    (rawState: z.infer<S>) => {
      const state = prettyParse(schema, rawState);
      setInternalState(state);
      onAetherChangeRef.current?.(state);
    },
    [schema],
  );

  const {
    path,
    setState: setAetherState,
    methods,
  } = useLifecycle({
    ...props,
    onReceive: handleReceive,
  });

  const setState = useCallback(
    (next: RawSetArg<S>, transfer: Transferable[] = []): void => {
      if (typeof next === "function")
        setInternalState((prev) => {
          const nextS = next(prev);
          // This makes our setter impure, so it's something we should be wary of causing
          // unexpected behavior in the the future.
          setAetherState(nextS, transfer);
          return nextS;
        });
      else {
        setInternalState(prettyParse(schema, next));
        setAetherState(next, transfer);
      }
    },
    [path, type, setInternalState],
  );

  return [{ path }, internalState, setState, methods];
};

type StateHandler<S = unknown> = (state: S) => void;

interface RegisteredComponent {
  path: string[];
  handler?: StateHandler;
}

/**
 * Props for the Composite component that establishes child components in the Aether tree
 */
export interface CompositeProps extends PropsWithChildren {
  /**
   * The path in the Aether component tree where children should be established
   */
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
