import { type CrudeTimeSpan, type destructor, type state } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";
import { type CallersFromSchema, type EmptyMethodsSchema, type MethodsSchema } from "./aether/aether";
import { type MainComms } from "./aether/message";
import { type RawSetArg, Store } from "./store";
/** Value supplied by the Aether context to descendants of {@link Provider}. */
export interface ContextValue {
    /** Path of the nearest enclosing {@link Composite}, or `["root"]` at the top level.
     * Components append their key to derive their own path. */
    path: readonly string[];
    /** Store shared by every component below the {@link Provider}. */
    store: Store;
}
export interface ProviderProps extends PropsWithChildren {
    /** URL of the worker script to spawn. Ignored when {@link worker} is set. */
    workerURL?: string | URL;
    /** Opt out of spawning a worker. Aether mounts, but all worker-bound operations
     * no-op. */
    workerEnabled?: boolean;
    /** Pre-built comms (e.g. from {@link aether.createMockPair} in tests). Takes
     * precedence over {@link workerURL} and {@link workerEnabled}. */
    worker?: MainComms;
    /** Default timeout for async method invocations. Defaults to 5s. */
    invokeTimeout?: CrudeTimeSpan;
}
/** Roots an Aether tree: owns a {@link Store}, spawns or wraps the worker, and supplies
 * the context that descendant {@link use} / {@link useLifecycle} / {@link Composite}
 * calls read. Re-throws worker-reported errors so the nearest React error boundary can
 * catch them. */
export declare const Provider: ({ children, ...config }: ProviderProps) => ReactElement;
/** Output of {@link useLifecycle}: the component's path, a typed setState, the methods
 * registry, and the subscribe / getSnapshot pair consumed by `useSyncExternalStore`. */
export interface UseLifecycleReturn<StateSchema extends z.ZodType<state.State, state.State>, Methods extends MethodsSchema = EmptyMethodsSchema> {
    path: readonly string[];
    setState: (state: RawSetArg<StateSchema>, transfer?: Transferable[]) => void;
    methods: CallersFromSchema<Methods>;
    subscribe: (listener: () => void) => destructor.Destructor;
    getSnapshot: () => z.infer<StateSchema>;
}
type StateHandler<T = unknown> = (state: T) => void;
interface UseLifecycleProps<StateSchema extends z.ZodType, Methods extends MethodsSchema = EmptyMethodsSchema> {
    /** Component type, matched against the worker-side registry. */
    type: string;
    /** Zod schema validating both `initialState` and worker-pushed state. */
    schema: StateSchema;
    /** Key for the component, generated if omitted. Read on the mounting render only;
     * later changes are ignored. Remount under a React `key` to get a new identity. */
    aetherKey?: string;
    initialState: z.input<StateSchema>;
    /** Optional `Transferable`s included with the initial update message. */
    initialTransfer?: Transferable[];
    /** Fired on worker-pushed state changes only (not on local setState). */
    onAetherChange?: StateHandler<z.infer<StateSchema>>;
    methods?: Methods;
}
/** Registers a component with the enclosing {@link Provider}'s {@link Store} and
 * returns the operations needed to drive it. Identity — store, enclosing path, and key —
 * is fixed on the mounting render; moving a component under a different
 * {@link Composite} needs a remount. Lower-level than {@link use} — does not subscribe
 * React to state changes. Most callers want {@link use} or {@link useUnidirectional}
 * instead. */
export declare const useLifecycle: <StateSchema extends z.ZodType<state.State, state.State>, Methods extends MethodsSchema = EmptyMethodsSchema>({ type, aetherKey, initialState, schema, initialTransfer, onAetherChange, methods: methodsSchema, }: UseLifecycleProps<StateSchema, Methods>) => UseLifecycleReturn<StateSchema, Methods>;
/** Mixin for React props of components that participate in the Aether tree. */
export interface ComponentProps {
    /** Optional override for the component's aether key. Stable for the component's
     * lifetime; usually omitted to auto-generate. */
    aetherKey?: string;
}
export interface UseProps<StateSchema extends z.ZodType, Methods extends MethodsSchema = EmptyMethodsSchema> extends Omit<UseLifecycleProps<StateSchema, Methods>, "onReceive"> {
    onAetherChange?: (state: z.infer<StateSchema>) => void;
}
interface ComponentContext {
    path: readonly string[];
}
/** Tuple returned by {@link use}: `[ctx, state, setState, methods]`. */
export type UseReturn<StateSchema extends z.ZodType<state.State, state.State>, Methods extends MethodsSchema = EmptyMethodsSchema> = [
    ComponentContext,
    z.infer<StateSchema>,
    (state: RawSetArg<StateSchema>, transfer?: Transferable[]) => void,
    CallersFromSchema<Methods>
];
export interface UseUnidirectionalProps<StateSchema extends z.ZodType, Methods extends MethodsSchema = EmptyMethodsSchema> extends Pick<UseLifecycleProps<StateSchema, Methods>, "schema" | "aetherKey" | "methods"> {
    type: string;
    /** Source-of-truth state owned by the caller. Push-only: changes here propagate to
     * the worker, but worker pushes do not flow back. */
    state: z.input<StateSchema>;
}
export interface UseUnidirectionalReturn<Methods extends MethodsSchema = EmptyMethodsSchema> extends ComponentContext {
    methods: CallersFromSchema<Methods>;
}
/** One-way binding for components whose state lives in React. Pushes `state` to the
 * worker on every change (compared with `deep.equal`); worker-side updates do not
 * propagate back. Use {@link use} for bidirectional state. */
export declare const useUnidirectional: <StateSchema extends z.ZodType<state.State, state.State>, Methods extends MethodsSchema = EmptyMethodsSchema>({ state, ...rest }: UseUnidirectionalProps<StateSchema, Methods>) => UseUnidirectionalReturn<Methods>;
/** Bidirectional binding: registers the component, subscribes React to its state via
 * `useSyncExternalStore`, and returns the current state, a setter, and the methods
 * registry. */
export declare const use: <StateSchema extends z.ZodType<state.State, state.State>, Methods extends MethodsSchema = EmptyMethodsSchema>(props: UseProps<StateSchema, Methods>) => UseReturn<StateSchema, Methods>;
export interface CompositeProps extends PropsWithChildren {
    path: readonly string[];
}
/** Pushes a new aether path into the context so descendants register as children of the
 * composite component identified by `path`. */
export declare const Composite: import("react").MemoExoticComponent<({ children, path }: CompositeProps) => ReactElement>;
export {};
//# sourceMappingURL=main.d.ts.map