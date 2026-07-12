// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type record } from "@synnaxlabs/x";
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

export type MembershipValue<K extends record.Key = record.Key> = K | K[] | undefined;

const contains = <K extends record.Key>(value: MembershipValue<K>, key: K): boolean => {
  if (value === undefined) return false;
  if (Array.isArray(value)) return value.includes(key);
  return value === key;
};

interface ContextValue<K extends record.Key = record.Key> extends Pick<
  UseKeyedListenersReturn<K>,
  "subscribe"
> {
  onItem: (key: K) => void;
  setValue: (keys: K[]) => void;
  clear: () => void;
  getValue: () => MembershipValue<K>;
}

export interface MembershipProps<K extends record.Key = record.Key>
  extends
    PropsWithChildren,
    Partial<Pick<ContextValue<K>, "onItem" | "setValue" | "clear">> {
  value: MembershipValue<K>;
}

export interface MembershipItem {
  member: boolean;
  onItem: () => void;
}

/**
 * Membership is a created set-membership context: a provider that publishes a controlled
 * set of keys plus hooks bound to its own React context. Nesting is independent, so a
 * created membership never reads or writes the set of another.
 */
export interface Membership {
  /**
   * Context distributes the controlled set to keyed consumers. It diffs value changes
   * and notifies only the listeners whose keys entered or left the set, so item
   * re-renders stay surgical via useItem.
   */
  Context: <K extends record.Key = record.Key>(
    props: MembershipProps<K>,
  ) => ReactElement;
  useContext: <K extends record.Key = record.Key>() => ContextValue<K>;
  /**
   * useItem subscribes a single keyed item to the enclosing Provider, re-rendering only
   * when this key's membership flips, and returns a bound onItem action request.
   */
  useItem: <K extends record.Key>(key: K) => MembershipItem;
  /** useIsMember subscribes a single keyed item, re-rendering only when it enters or
   * leaves the set. */
  useIsMember: <K extends record.Key>(key: K) => boolean;
  /** useMembers returns the current members in order. */
  useMembers: <K extends record.Key>() => K[];
  /**
   * useMemberAmong returns the member among the given keys, or undefined when none of
   * them is a member. It subscribes only to the given keys, so consumers stay isolated
   * from changes to the rest of the set. When more than one is a member, the earliest in
   * the set's order wins.
   */
  useMemberAmong: <K extends record.Key>(keys: K[]) => K | undefined;
}

const NOOP = () => {};

/**
 * createMembership mints a typed set-membership context: a Provider that publishes a
 * controlled set of keys and hooks that read it. Call once per use case so each set
 * carries its own React context and nests independently of other sets.
 *
 * @param name identifies the set in the context display name.
 */
export const createMembership = (name: string): Membership => {
  const [BaseContext, useCtx] = context.create<ContextValue>({
    defaultValue: {
      clear: NOOP,
      getValue: () => undefined,
      onItem: NOOP,
      setValue: NOOP,
      subscribe: () => NOOP,
    },
    displayName: `${name}.Context`,
  });

  const Context = <K extends record.Key = record.Key>({
    value,
    onItem = NOOP,
    setValue = NOOP,
    clear = NOOP,
    children,
  }: MembershipProps<K>): ReactElement => {
    const valueRef = useRef(value);
    const { notifyListeners, subscribe } = useKeyedListeners<K>();
    const getValue = useCallback(() => valueRef.current, []);
    const ctx = useMemo<ContextValue<K>>(
      () => ({ onItem, setValue, clear, subscribe, getValue }),
      [getValue, onItem, setValue, clear, subscribe],
    );
    useLayoutEffect(() => {
      const changed = new Set(array.toArray(valueRef.current)).symmetricDifference(
        new Set(array.toArray(value)),
      );
      valueRef.current = value;
      if (changed.size > 0) notifyListeners(Array.from(changed));
    }, [value, notifyListeners]);
    return (
      <BaseContext value={ctx as unknown as ContextValue<record.Key>}>
        {children}
      </BaseContext>
    );
  };

  // The context is stored monomorphically at record.Key (React contexts are invariant);
  // each consumer reapplies its own K here and in the Provider.
  const useContext = <K extends record.Key = record.Key>(): ContextValue<K> =>
    useCtx() as unknown as ContextValue<K>;

  const useIsMember = <K extends record.Key>(key: K): boolean => {
    const { getValue, subscribe } = useContext<K>();
    return useSyncExternalStore(
      useCallback((onStoreChange) => subscribe(onStoreChange, key), [key, subscribe]),
      useCallback((): boolean => contains(getValue(), key), [key, getValue]),
      useCallback((): boolean => false, []),
    );
  };

  const useItem = <K extends record.Key>(key: K): MembershipItem => {
    const { onItem } = useContext<K>();
    const member = useIsMember(key);
    const handleItem = useCallback(() => onItem(key), [key, onItem]);
    return useMemo(() => ({ member, onItem: handleItem }), [member, handleItem]);
  };

  const useMemberAmong = <K extends record.Key>(keys: K[]): K | undefined => {
    const { getValue, subscribe } = useContext<K>();
    return useSyncExternalStore(
      useCallback(
        (onStoreChange) => {
          const destructors = keys.map((key) => subscribe(() => onStoreChange(), key));
          return () => destructors.forEach((d) => d());
        },
        [keys, subscribe],
      ),
      useCallback((): K | undefined => {
        const value = getValue();
        if (value === undefined) return undefined;
        return array.toArray(value).find((key) => keys.includes(key));
      }, [keys, getValue]),
      useCallback((): K | undefined => undefined, []),
    );
  };

  const useMembers = <K extends record.Key>(): K[] => {
    const { getValue, subscribe } = useContext<K>();
    const res = useSyncExternalStore(subscribe, getValue, () => undefined);
    return useMemo((): K[] => (res == null ? [] : array.toArray(res)), [res]);
  };

  return { Context, useContext, useItem, useIsMember, useMembers, useMemberAmong };
};
