// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
  useCallback,
  useMemo,
  useState,
  useSyncExternalStore,
} from "react";

import { context } from "@/context";
import { useInitializerRef } from "@/hooks";
import { Store } from "@/store";

interface FixedItem {
  element: HTMLElement | null;
  hidden: boolean;
}

/**
 * Registry tracks the parts of a select that live outside its data: fixed items, the
 * position of the data block among them, and the label slots triggers render.
 */
export interface Registry {
  setItem: (key: record.Key, item: FixedItem) => void;
  removeItem: (key: record.Key) => void;
  setBlock: (element: HTMLElement | null) => void;
  /**
   * @returns every visible option in page order: fixed items, with the data keys at the
   * data block's position, or after the fixed items when no block is rendered.
   */
  getOrder: (data: record.Key[]) => record.Key[];
  getElement: (key: record.Key) => HTMLElement | null;
  hasItem: (key: record.Key) => boolean;
  /** @returns a number that changes whenever the order may have changed. */
  getVersion: () => number;
  countVisible: () => number;
  setSlot: (key: record.Key, element: HTMLElement | null) => void;
  getSlot: (key: record.Key) => HTMLElement | undefined;
  /** Subscribes to changes for one key, or to every change when key is omitted. */
  subscribe: (listener: () => void, key?: record.Key) => destructor.Destructor;
}

const BLOCK = Symbol("block");

const compareDocument = (a: HTMLElement, b: HTMLElement): number =>
  a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1;

const useRegistry = (): Registry => {
  const { notifyListeners, subscribe } = Store.useKeyedListeners<record.Key>();
  const itemsRef = useInitializerRef(() => new Map<record.Key, FixedItem>());
  const slotsRef = useInitializerRef(() => new Map<record.Key, HTMLElement>());
  const stateRef = useInitializerRef<{ block: HTMLElement | null; version: number }>(
    () => ({ block: null, version: 0 }),
  );
  return useMemo<Registry>(
    () => ({
      setItem: (key, item) => {
        const prev = itemsRef.current.get(key);
        if (prev?.element === item.element && prev?.hidden === item.hidden) return;
        itemsRef.current.set(key, item);
        stateRef.current.version++;
        notifyListeners(key);
      },
      removeItem: (key) => {
        if (!itemsRef.current.delete(key)) return;
        stateRef.current.version++;
        notifyListeners(key);
      },
      setBlock: (element) => {
        stateRef.current.block = element;
        stateRef.current.version++;
        notifyListeners([]);
      },
      getOrder: (data) => {
        const mounted: [HTMLElement, record.Key | typeof BLOCK][] = [];
        const unmounted: record.Key[] = [];
        itemsRef.current.forEach(({ element, hidden }, key) => {
          if (hidden) return;
          if (element == null) unmounted.push(key);
          else mounted.push([element, key]);
        });
        const { block } = stateRef.current;
        if (block != null) mounted.push([block, BLOCK]);
        mounted.sort(([a], [b]) => compareDocument(a, b));
        const order: record.Key[] = [...unmounted];
        mounted.forEach(([, key]) => {
          if (key === BLOCK) order.push(...data);
          else order.push(key);
        });
        if (block == null) order.push(...data);
        return order;
      },
      getElement: (key) => itemsRef.current.get(key)?.element ?? null,
      hasItem: (key) => itemsRef.current.has(key),
      getVersion: () => stateRef.current.version,
      countVisible: () => {
        let count = 0;
        itemsRef.current.forEach(({ hidden }) => {
          if (!hidden) count++;
        });
        return count;
      },
      setSlot: (key, element) => {
        if (element == null) slotsRef.current.delete(key);
        else slotsRef.current.set(key, element);
        notifyListeners(key);
      },
      getSlot: (key) => slotsRef.current.get(key),
      subscribe,
    }),
    [notifyListeners, subscribe],
  );
};

const [RegistryContext, useRegistryContext] = context.create<Registry>({
  displayName: "Select.RegistryContext",
  providerName: "Select.Frame",
});

interface SearchContextValue {
  term: string;
  setTerm: (term: string) => void;
}

const [SearchContext, useSearchContext] = context.create<SearchContextValue>({
  displayName: "Select.SearchContext",
  providerName: "Select.Frame",
});

export { RegistryContext, useRegistry, useRegistryContext, useSearchContext };

/** Shares a search term between a frame's search field and its fixed items. */
export const SearchProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [term, setTerm] = useState("");
  const value = useMemo(() => ({ term, setTerm }), [term]);
  return <SearchContext value={value}>{children}</SearchContext>;
};

/** @returns whether a fixed item with the given key is registered in the frame. */
export const useIsFixed = (key: record.Key | undefined): boolean => {
  const registry = useRegistryContext("Select.useIsFixed");
  return useSyncExternalStore(
    useCallback(
      (listener) => (key == null ? () => {} : registry.subscribe(listener, key)),
      [registry, key],
    ),
    () => key != null && registry.hasItem(key),
    () => false,
  );
};

/** @returns the label slot a trigger registered for the given key. */
export const useSlot = (key: record.Key): HTMLElement | undefined => {
  const registry = useRegistryContext("Select.useSlot");
  return useSyncExternalStore(
    useCallback((listener) => registry.subscribe(listener, key), [registry, key]),
    () => registry.getSlot(key),
    () => undefined,
  );
};

/** @returns a ref that registers its element as the label slot for the given key. */
export const useSlotRef = (key: record.Key | undefined): RefCallback<HTMLElement> => {
  const registry = useRegistryContext("Select.useSlotRef");
  return useCallback(
    (element: HTMLElement | null) => {
      if (key == null) return;
      registry.setSlot(key, element);
      return () => registry.setSlot(key, null);
    },
    [registry, key],
  );
};

/**
 * @returns the number of fixed items the search has not hidden. It re-renders the
 * caller whenever an item registers, unregisters, or changes visibility.
 */
export const useVisibleCount = (): number => {
  const registry = useRegistryContext("Select.useVisibleCount");
  return useSyncExternalStore(
    useCallback((listener) => registry.subscribe(listener), [registry]),
    () => registry.countVisible(),
    () => 0,
  );
};
