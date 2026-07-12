// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useMemo } from "react";

import { Store } from "@/store";

type Value<K extends record.Key = record.Key> = Store.MembershipValue<K>;

// Focus is only defined for ordered multi-selections: the key heading the value
// array is the focused key. A scalar selection carries no ordering, so nothing is
// focused.
const head = <K extends record.Key>(value: Value<K>): K | undefined =>
  Array.isArray(value) ? value[0] : undefined;

interface SelectionState<K extends record.Key = record.Key> {
  value: Value<K>;
  hover?: K;
}

interface ContextValue<K extends record.Key = record.Key> {
  onSelect: (key: K) => void;
  setSelected: (keys: K[]) => void;
  clear: () => void;
  subscribe: (listener: () => void, key?: K) => () => void;
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

const Members = Store.createMembership("Selection");
const Hover = Store.createPresence("Selection.Hover");
const Focus = Store.createPresence("Selection.Focus");

/**
 * Context distributes a controlled selection to keyed item consumers. Membership, focus,
 * and hover are held in independent stores, so an item re-renders only for the dimensions
 * it reads via useItemState. Focus tracks the head of an ordered multi-selection.
 */
export const Context = <K extends record.Key = record.Key>({
  value,
  onSelect,
  setSelected,
  clear,
  hover,
  children,
}: ContextProps<K>): ReactElement => (
  <Members.Context value={value} onItem={onSelect} setValue={setSelected} clear={clear}>
    <Focus.Context value={head(value)}>
      <Hover.Context value={hover}>{children}</Hover.Context>
    </Focus.Context>
  </Members.Context>
);

/** useContext returns the enclosing selection's imperative handle. */
export const useContext = <K extends record.Key = record.Key>(): ContextValue<K> => {
  const members = Members.useContext<K>();
  const hover = Hover.useContext<K>();
  return useMemo<ContextValue<K>>(
    () => ({
      onSelect: members.onItem,
      setSelected: members.setValue,
      clear: members.clear,
      subscribe: members.subscribe,
      getState: () => ({ value: members.getValue(), hover: hover.getPresent() }),
    }),
    [members, hover],
  );
};

/**
 * useItemState subscribes a single keyed item to the enclosing Context, re-rendering only
 * when that key's selected, focused, or hovered state flips.
 */
export const useItemState = <K extends record.Key>(key: K): UseItemStateReturn => {
  const { member, onItem } = Members.useItem(key);
  const focused = Focus.useIsPresent(key);
  const hovered = Hover.useIsPresent(key);
  return useMemo(
    () => ({ selected: member, focused, hovered, onSelect: onItem }),
    [member, focused, hovered, onItem],
  );
};

/**
 * useSelectedAmong returns the selected key among the given keys, or undefined when none
 * of them is selected. It subscribes only to the given keys, so consumers stay isolated
 * from changes to the rest of the selection. When more than one of the keys is selected,
 * the earliest in the selection's order wins.
 */
export const useSelectedAmong = Members.useMemberAmong;

/** useSelected returns the currently selected keys. */
export const useSelected = Members.useMembers;

/** useClear returns a callback that clears the enclosing selection. */
export const useClear = (): (() => void) => Members.useContext().clear;
