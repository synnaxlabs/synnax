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
import { Store } from "@/store";

type Value<K extends record.Key = record.Key> = K | K[] | undefined;

interface SelectionState<K extends record.Key = record.Key> {
  value: Value<K>;
  hover?: K;
}

const isSelected = <K extends record.Key>(value: Value<K>, key: K): boolean => {
  if (value === undefined) return false;
  if (Array.isArray(value)) return value.includes(key);
  return value === key;
};

// Focus is only defined for ordered multi-selections: the key heading the value
// array is the focused key. A scalar selection carries no ordering, so nothing is
// focused.
const isFocused = <K extends record.Key>(value: Value<K>, key: K): boolean =>
  Array.isArray(value) && value.length > 0 && value[0] === key;

const head = <K extends record.Key>(value: Value<K>): K | undefined =>
  Array.isArray(value) ? value[0] : undefined;

interface ContextValue<K extends record.Key = record.Key> extends Pick<
  Store.UseKeyedListenersReturn<K>,
  "subscribe"
> {
  onSelect: (key: K) => void;
  setSelected: (keys: K[]) => void;
  clear: () => void;
  getState: () => SelectionState<K>;
}

export interface ContextProps<K extends record.Key = record.Key>
  extends
    PropsWithChildren,
    Partial<Pick<ContextValue<K>, "onSelect" | "setSelected" | "clear">>,
    SelectionState<K> {}

export interface UseItemStateReturn {
  selected: boolean;
  /**
   * focused is true when the key heads an ordered multi-selection: the value is an
   * array and this key is its first element. Always false for scalar selections.
   */
  focused: boolean;
  hovered: boolean;
  onSelect: () => void;
}

/**
 * CreateReturn is a created selection context: a provider that publishes a controlled
 * selection plus hooks bound to its own React context. Nesting is independent, so a
 * created selection never reads or writes the selection of another.
 */
export interface CreateReturn {
  /**
   * Context distributes a controlled selection to keyed item consumers. It diffs value
   * changes and notifies only the listeners whose keys entered or left the selection,
   * so item re-renders stay surgical via useItemState.
   */
  Context: <K extends record.Key = record.Key>(props: ContextProps<K>) => ReactElement;
  /** useContext returns the enclosing selection's value, throwing nothing if absent. */
  useContext: <K extends record.Key = record.Key>() => ContextValue<K>;
  /**
   * useItemState subscribes a single keyed item to the enclosing Context, re-rendering
   * only when that key's selected, focused, or hovered state flips.
   */
  useItemState: <K extends record.Key>(key: K) => UseItemStateReturn;
  /**
   * useSelectedAmong returns the selected key among the given keys, or undefined when
   * none of them is selected. It subscribes only to the given keys, so consumers stay
   * isolated from changes to the rest of the selection. When more than one of the keys
   * is selected, the earliest in the selection's order wins.
   */
  useSelectedAmong: <K extends record.Key>(keys: K[]) => K | undefined;
  /** useSelected returns the currently selected keys. */
  useSelected: <K extends record.Key>() => K[];
  /** useClear returns a callback that clears the enclosing selection. */
  useClear: () => () => void;
}

const SELECTED_FLAG = 1;
const HOVERED_FLAG = 2;
const FOCUSED_FLAG = 4;

const NOOP = () => {};

/**
 * create mints a typed selection context: a Context provider that publishes a controlled
 * selection and hooks that read it. Call once per use case so each selection carries its
 * own React context and nests independently of other selections.
 *
 * @param name identifies the selection in the context display name.
 */
export const create = (name: string): CreateReturn => {
  const [BaseContext, useCtx] = context.create<ContextValue>({
    defaultValue: {
      clear: NOOP,
      getState: () => ({ value: undefined, hover: undefined }),
      onSelect: NOOP,
      setSelected: NOOP,
      subscribe: () => NOOP,
    },
    displayName: `${name}.Context`,
  });

  const Context = <K extends record.Key = record.Key>({
    value,
    onSelect = NOOP,
    clear = NOOP,
    setSelected = NOOP,
    children,
    hover,
  }: ContextProps<K>): ReactElement => {
    const valueRef = useRef(value);
    const hoverRef = useRef(hover);

    const { notifyListeners, subscribe } = Store.useKeyedListeners<K>();

    const getState = useCallback(
      () => ({ value: valueRef.current, hover: hoverRef.current }),
      [],
    );
    const ctx = useMemo<ContextValue<K>>(
      () => ({ onSelect, setSelected, clear, subscribe, getState }),
      [getState, onSelect, setSelected, clear, subscribe],
    );
    useLayoutEffect(() => {
      const changed = new Set(array.toArray(valueRef.current)).symmetricDifference(
        new Set(array.toArray(value)),
      );
      // A reorder can move focus without changing membership, so head changes are
      // diffed separately.
      const prevHead = head(valueRef.current);
      const nextHead = head(value);
      if (prevHead !== nextHead) {
        if (prevHead != null) changed.add(prevHead);
        if (nextHead != null) changed.add(nextHead);
      }
      valueRef.current = value;
      const prevHover = hoverRef.current;
      if (prevHover !== hover) {
        if (prevHover != null) changed.add(prevHover);
        if (hover != null) changed.add(hover);
        hoverRef.current = hover;
      }
      if (changed.size > 0) notifyListeners(Array.from(changed));
    }, [value, hover, notifyListeners]);

    return (
      <BaseContext value={ctx as unknown as ContextValue<record.Key>}>
        {children}
      </BaseContext>
    );
  };

  const useContext = <K extends record.Key = record.Key>(): ContextValue<K> =>
    useCtx() as unknown as ContextValue<K>;

  const useItemState = <K extends record.Key>(key: K): UseItemStateReturn => {
    const { getState, onSelect, subscribe } = useContext();
    const handleSelect = useCallback(() => onSelect(key), [key, onSelect]);
    const itemState = useSyncExternalStore(
      useCallback((onStoreChange) => subscribe(onStoreChange, key), [key, subscribe]),
      useCallback((): number => {
        const state = getState();
        let flags = 0;
        if (isSelected(state.value, key)) flags |= SELECTED_FLAG;
        if (state.hover === key) flags |= HOVERED_FLAG;
        if (isFocused(state.value, key)) flags |= FOCUSED_FLAG;
        return flags;
      }, [key, getState]),
      useCallback((): number => 0, []),
    );
    return useMemo(
      () => ({
        selected: (itemState & SELECTED_FLAG) !== 0,
        hovered: (itemState & HOVERED_FLAG) !== 0,
        focused: (itemState & FOCUSED_FLAG) !== 0,
        onSelect: handleSelect,
      }),
      [itemState, handleSelect],
    );
  };

  const useSelectedAmong = <K extends record.Key>(keys: K[]): K | undefined => {
    const { getState, subscribe } = useContext<K>();
    return useSyncExternalStore(
      useCallback(
        (onStoreChange) => {
          const destructors = keys.map((key) => subscribe(() => onStoreChange(), key));
          return () => destructors.forEach((d) => d());
        },
        [keys, subscribe],
      ),
      useCallback((): K | undefined => {
        const { value } = getState();
        if (value === undefined) return undefined;
        return array.toArray(value).find((key) => keys.includes(key));
      }, [keys, getState]),
      useCallback((): K | undefined => undefined, []),
    );
  };

  const useSelected = <K extends record.Key>(): K[] => {
    const { getState, subscribe } = useContext<K>();
    const res = useSyncExternalStore(
      subscribe,
      () => getState().value,
      () => null,
    );
    return useMemo((): K[] => (res == null ? [] : array.toArray(res)), [res]);
  };

  const useClear = () => useContext().clear;

  return { Context, useContext, useItemState, useSelectedAmong, useSelected, useClear };
};

export const {
  Context,
  useContext,
  useItemState,
  useSelectedAmong,
  useSelected,
  useClear,
} = create("Selection");
