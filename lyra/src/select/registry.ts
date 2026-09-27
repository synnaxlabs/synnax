// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type record } from "@synnaxlabs/x";
import { useLayoutEffect, useMemo, useReducer, useRef } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { useInitializerRef } from "@/hooks";
import { Store } from "@/store";

interface FixedItem {
  element: HTMLElement | null;
  hidden: boolean;
}

const matches = (text: string, term: string): boolean =>
  term === "" || text.toLowerCase().includes(term.toLowerCase());

const isHidden = (element: HTMLElement | null, term: string): boolean =>
  element != null && !matches(element.textContent, term);

/**
 * Registry tracks the parts of a select that live outside its data: fixed items, the
 * position of the data block among them, and the label elements triggers show.
 */
export interface Registry {
  /** Records a fixed item's element, hiding it when its text misses the search term. */
  setItem: (key: record.Key, element: HTMLElement | null) => void;
  removeItem: (key: record.Key) => void;
  setBlock: (element: HTMLElement | null) => void;
  /**
   * @returns every visible option in page order: fixed items, with the data keys at the
   * data block's position, or after the fixed items when no block is rendered.
   */
  getOrder: (data: record.Key[]) => record.Key[];
  getElement: (key: record.Key) => HTMLElement | null;
  hasItem: (key: record.Key) => boolean;
  /** @returns whether the search term hides the fixed item with the given key. */
  isHidden: (key: record.Key) => boolean;
  countVisible: () => number;
  /** Hides the fixed items whose text does not contain the term. */
  setTerm: (term: string) => void;
  /** @returns the element a fixed item's label renders into, created on first use. */
  getLabel: (key: record.Key) => HTMLElement;
  /** Subscribes to changes for one key, or to every change when key is omitted. */
  subscribe: (listener: () => void, key?: record.Key) => destructor.Destructor;
}

const BLOCK = Symbol("block");

const compareDocument = (a: HTMLElement, b: HTMLElement): number =>
  a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1;

const useRegistry = (): Registry => {
  const { notifyListeners, subscribe } = Store.useKeyedListeners<record.Key>();
  const itemsRef = useInitializerRef(() => new Map<record.Key, FixedItem>());
  const labelsRef = useInitializerRef(() => new Map<record.Key, HTMLElement>());
  const blockRef = useRef<HTMLElement | null>(null);
  const termRef = useRef("");
  return useMemo<Registry>(
    () => ({
      setItem: (key, element) => {
        const prev = itemsRef.current.get(key);
        const hidden = isHidden(element, termRef.current);
        if (prev?.element === element && prev?.hidden === hidden) return;
        itemsRef.current.set(key, { element, hidden });
        notifyListeners(key);
      },
      removeItem: (key) => {
        if (!itemsRef.current.delete(key)) return;
        notifyListeners(key);
      },
      setBlock: (element) => {
        blockRef.current = element;
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
        const block = blockRef.current;
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
      isHidden: (key) => itemsRef.current.get(key)?.hidden ?? false,
      setTerm: (term) => {
        termRef.current = term;
        const changed: record.Key[] = [];
        itemsRef.current.forEach((item, key) => {
          const hidden = isHidden(item.element, term);
          if (hidden === item.hidden) return;
          item.hidden = hidden;
          changed.push(key);
        });
        if (changed.length > 0) notifyListeners(changed);
      },
      countVisible: () => {
        let count = 0;
        itemsRef.current.forEach(({ hidden }) => {
          if (!hidden) count++;
        });
        return count;
      },
      getLabel: (key) => {
        let label = labelsRef.current.get(key);
        if (label == null) {
          label = document.createElement("span");
          label.className = CSS.BE("select", "label");
          labelsRef.current.set(key, label);
        }
        return label;
      },
      subscribe,
    }),
    [notifyListeners, subscribe],
  );
};

const [RegistryContext, useRegistryContext] = context.create<Registry>({
  displayName: "Select.RegistryContext",
  providerName: "Select.Frame",
});

export { RegistryContext, useRegistry, useRegistryContext };

// Items register in layout effects, after their readers render. useSyncExternalStore
// subscribes after paint, so a reader would paint its stale value for one frame.
const useRegistryValue = <T>(
  name: string,
  get: (registry: Registry, key?: record.Key) => T,
  key?: record.Key,
): T => {
  const registry = useRegistryContext(name);
  const [, rerender] = useReducer((n: number) => n + 1, 0);
  const value = get(registry, key);
  const renderedRef = useRef(value);
  useLayoutEffect(() => {
    renderedRef.current = value;
  });
  useLayoutEffect(() => {
    const check = (): void => {
      if (get(registry, key) !== renderedRef.current) rerender();
    };
    check();
    return registry.subscribe(check, key);
  }, [registry, get, key]);
  return value;
};

const isFixed = (registry: Registry, key?: record.Key): boolean =>
  key != null && registry.hasItem(key);

const countVisible = (registry: Registry): number => registry.countVisible();

const isHiddenItem = (registry: Registry, key?: record.Key): boolean =>
  key != null && registry.isHidden(key);

/** @returns whether a fixed item with the given key is registered in the frame. */
export const useIsFixed = (key: record.Key | undefined): boolean =>
  useRegistryValue("Select.useIsFixed", isFixed, key);

/**
 * @returns the number of fixed items the search has not hidden. It re-renders the
 * caller whenever an item registers, unregisters, or changes visibility.
 */
export const useVisibleCount = (): number =>
  useRegistryValue("Select.useVisibleCount", countVisible);

/** @returns whether the search term hides the fixed item with the given key. */
export const useIsHidden = (key: record.Key): boolean =>
  useRegistryValue("Select.useIsHidden", isHiddenItem, key);
