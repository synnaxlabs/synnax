// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { context } from "@/context";
import { useKeyedListeners, type UseKeyedListenersReturn } from "@/store/store";

interface ContextValue<K extends record.Key = record.Key> extends Pick<
  UseKeyedListenersReturn<K>,
  "subscribe"
> {
  getPresent: () => K | undefined;
}

export interface PresenceProps<
  K extends record.Key = record.Key,
> extends PropsWithChildren {
  value?: K;
}

/**
 * Presence is a created presence context: a provider that publishes at most one present
 * key plus hooks bound to its own React context. Nesting is independent, so a created
 * presence never reads or writes the presence of another.
 */
export interface Presence<K extends record.Key = record.Key> {
  /**
   * Context distributes the single present key to keyed consumers. On change it notifies
   * only the previous and next holders, so at most two items re-render.
   */
  Context: (props: PresenceProps<K>) => ReactElement;
  useContext: () => ContextValue<K>;
  /**
   * useIsPresent subscribes a single keyed item to the enclosing Provider, re-rendering
   * only when this key gains or loses presence.
   */
  useIsPresent: (key: K) => boolean;
  /** usePresent returns the currently present key, or undefined when none is present. */
  usePresent: () => K | undefined;
}

const NOOP = () => {};

/**
 * createPresence mints a typed presence context: a Provider that publishes at most one
 * present key and hooks that read it. Presence is a single-valued membership: the hovered
 * item, the active item, the keyboard cursor. Call once per use case so each presence
 * carries its own React context and nests independently of other presences.
 *
 * @param name identifies the presence in the context display name.
 */
export const createPresence = (name: string): Presence => {
  const [BaseContext, useCtx] = context.create<ContextValue>({
    defaultValue: { getPresent: () => undefined, subscribe: () => NOOP },
    displayName: `${name}.Context`,
  });

  const Context = ({ value, children }: PresenceProps): ReactElement => {
    const valueRef = useRef(value);
    const { notifyListeners, subscribe } = useKeyedListeners();
    const getPresent = useCallback(() => valueRef.current, []);
    const ctx = useMemo<ContextValue>(
      () => ({ getPresent, subscribe }),
      [getPresent, subscribe],
    );
    useLayoutEffect(() => {
      if (valueRef.current === value) return;
      const changed: record.Key[] = [];
      if (valueRef.current != null) changed.push(valueRef.current);
      if (value != null) changed.push(value);
      valueRef.current = value;
      if (changed.length > 0) notifyListeners(changed);
    }, [value, notifyListeners]);
    return <BaseContext value={ctx}>{children}</BaseContext>;
  };

  const useContext = (): ContextValue => useCtx();

  const useIsPresent = (key: record.Key): boolean => {
    const { getPresent, subscribe } = useContext();
    return useSyncExternalStore(
      useCallback((onStoreChange) => subscribe(onStoreChange, key), [key, subscribe]),
      useCallback((): boolean => getPresent() === key, [key, getPresent]),
      useCallback((): boolean => false, []),
    );
  };

  const usePresent = (): record.Key | undefined => {
    const { getPresent, subscribe } = useContext();
    return useSyncExternalStore(
      subscribe,
      () => getPresent(),
      () => undefined,
    );
  };

  return { Context, useContext, useIsPresent, usePresent };
};
