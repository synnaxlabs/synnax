import { type primitive, state } from "@synnaxlabs/x";
import { type Ref, type RefCallback, type RefObject } from "react";
/**
 * A ref that satisfies the interface of useState, but returns a ref as the first
 * element of the tuple. This is useful when you want to keep a piece of state but don't
 * want its changes to trigger a re-render.
 * @returns a tuple containing the ref and the pseudo-setState function.
 */
export declare const useStateRef: <T extends state.State>(initialValue: state.Initial<T>) => [RefObject<T>, state.Setter<T>];
/**
 * Use synced ref keeps the provided value in sync with the returned ref. This is useful
 * when you want access to a piece of state but don't want it's changes to trigger a
 * re-render.
 * @returns a ref that is kept in sync with the provided value.
 */
export declare const useSyncedRef: <T>(value: T) => RefObject<T>;
/**
 * Holds a ref whose value is built on the first render and kept for the component's
 * life. Use it in place of `useRef(expensive())`, which builds a fresh value on every
 * render and throws it away.
 *
 * @example const store = useInitializerRef(() => new Store());
 */
export declare const useInitializerRef: <T>(initializer: () => T) => RefObject<T>;
/**
 * Combines multiple refs into one. Note that the returned ref callback will not be
 * updated when the provided refs changes. These refs are only set once, and are assumed
 * to be static throughout the lifetime of the component.
 *
 * @returns - A callback ref that will set all of the provided refs.
 */
export declare const useCombinedRefs: <T>(...refs: Array<Ref<T> | null | undefined>) => RefCallback<T>;
/**
 * Keeps a piece of state and a ref to it. The ref is assigned by the setter itself, so
 * it is current as soon as the setter returns, before the re-render it triggers.
 * @param initialState - The initial state, or a function that lazily computes it.
 * @returns a tuple of the state, its setter, and a ref holding the latest value.
 */
export declare const useCombinedStateAndRef: <T extends primitive.Value | object>(initialState: state.Initial<T>) => [T, state.Setter<T>, RefObject<T>];
/**
 * @returns the value this hook was given on the previous render, or undefined on the
 * first. Compare against it to react to a change without storing it in state.
 */
export declare const usePrevious: <T>(value: T) => T | undefined;
//# sourceMappingURL=ref.d.ts.map