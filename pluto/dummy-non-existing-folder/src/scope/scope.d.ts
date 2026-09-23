import { type optional, type record } from "@synnaxlabs/x";
import { type FC, type PropsWithChildren } from "react";
export interface ProviderProps<K extends record.Key> extends PropsWithChildren {
    value: K;
}
/**
 * Hook is a hook whose key argument has been made optional by {@link Instance.bindHook}.
 * The key is resolved from the surrounding scope unless overridden. If the hook has no
 * arguments other than the key, it may be called with no arguments at all.
 */
export type Hook<K extends record.Key, Args extends record.Keyed<K>, R> = optional.Arg<optional.Optional<Args, "key">, R>;
/** Instance is a created scope: a Provider plus hooks bound to a single key type. */
export interface Instance<K extends record.Key> {
    /** Provider sets the active key for all descendants. */
    Provider: FC<ProviderProps<K>>;
    /**
     * use resolves the active key. An explicitly passed override takes precedence over
     * the surrounding scope. It throws if neither an override nor a Provider is present.
     */
    use: (override?: K) => K;
    /** useOptional behaves like use but returns undefined instead of throwing. */
    useOptional: (override?: K) => K | undefined;
    /**
     * require narrows an already-resolved key, throwing if it is nullish. Use inside a
     * callback to enforce a key where the use hook cannot be called.
     */
    require: (key: K | undefined) => K;
    /**
     * bindHook lifts a hook that requires a key into one whose key is optional, sourcing
     * it from the surrounding scope. All non-key arguments are forwarded unchanged.
     */
    bindHook: <Args extends record.Keyed<K>, R>(hook: (args: Args) => R) => Hook<K, Args, R>;
}
/**
 * create mints a typed scope: a Provider that publishes an active key and hooks that
 * read it. Call once per domain so each scope carries its own branded key type and
 * nests independently of other domains.
 * @param name identifies the scope in error messages and the context display name.
 */
export declare const create: <K extends record.Key>(name: string) => Instance<K>;
//# sourceMappingURL=scope.d.ts.map