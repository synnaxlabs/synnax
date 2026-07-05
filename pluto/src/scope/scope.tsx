// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type optional, type record } from "@synnaxlabs/x";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";

export interface ProviderProps<K extends record.Key> extends PropsWithChildren {
  value: K;
}

/**
 * Hook is a hook whose key argument has been made optional by {@link Instance.bindHook}.
 * The key is resolved from the surrounding scope unless overridden. If the hook has no
 * arguments other than the key, it may be called with no arguments at all.
 */
export type Hook<K extends record.Key, Args extends record.Keyed<K>, R> = optional.Arg<
  optional.Optional<Args, "key">,
  R
>;

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
   * bindHook lifts a hook that requires a key into one whose key is optional, sourcing
   * it from the surrounding scope. All non-key arguments are forwarded unchanged.
   */
  bindHook: <Args extends record.Keyed<K>, R>(
    hook: (args: Args) => R,
  ) => Hook<K, Args, R>;
}

/**
 * create mints a typed scope: a Provider that publishes an active key and hooks that
 * read it. Call once per domain so each scope carries its own branded key type and
 * nests independently of other domains.
 *
 * @param name identifies the scope in error messages and the context display name.
 */
export const create = <K extends record.Key>(name: string): Instance<K> => {
  const [Context, useContextValue] = context.create<K | undefined>({
    displayName: `${name}.Scope`,
    defaultValue: undefined,
  });

  const Provider = ({ value, children }: ProviderProps<K>): ReactElement => (
    <Context value={value}>{children}</Context>
  );

  const useOptional = (override?: K): K | undefined => {
    const fromScope = useContextValue();
    return override ?? fromScope;
  };

  const use = (override?: K): K => {
    const value = useOptional(override);
    if (value == null)
      throw new Error(
        `${name} scope requires a key: pass one explicitly or render within ${name}.Provider`,
      );
    return value;
  };

  const bindHook =
    <Args extends record.Keyed<K>, R>(hook: (args: Args) => R): Hook<K, Args, R> =>
    (args?: optional.Optional<Args, "key">): R => {
      const key = use(args?.key);
      return hook({ ...args, key } as Args);
    };

  return { Provider, use, useOptional, bindHook };
};
