// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor, type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";
import { Store } from "@/store";

type Value<K extends record.Key = record.Key> = Store.MembershipValue<K>;

// Focus is only defined for ordered multi-selections: the key heading the value array
// is the focused key. A scalar selection carries no ordering, so nothing is focused.
const focusOf = <K extends record.Key>(value: Value<K>): K | undefined =>
  Array.isArray(value) ? value[0] : undefined;

const headOf = <K extends record.Key>(value: Value<K>): K | undefined =>
  Array.isArray(value) ? value[0] : value;

const soleOf = <K extends record.Key>(value: Value<K>): K | undefined => {
  if (!Array.isArray(value)) return value;
  return value.length === 1 ? value[0] : undefined;
};

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

/** Props for {@link Context}. */
export interface ContextProps<K extends record.Key = record.Key>
  extends
    PropsWithChildren,
    Partial<Pick<ContextValue<K>, "onSelect" | "setSelected" | "clear">>,
    SelectionState<K> {
  /** True when clicking the sole selected item changes nothing. */
  reselectNoop?: boolean;
  /** Moves the hover onto the item under the pointer. */
  setHover?: (key: K) => void;
}

/** Return value for {@link useItemState}. */
export interface UseItemStateReturn {
  selected: boolean;
  /**
   * focused is true when the key heads an ordered multi-selection: the value is an
   * array and this key is its first element. Always false for scalar selections.
   */
  focused: boolean;
  head: boolean;
  /** True when the key is the entire selection: its sole member. */
  sole: boolean;
  hovered: boolean;
  onSelect: () => void;
  /** Moves the hover onto the item, for when the pointer moves onto it. */
  onHover: () => void;
}

// The membership and presence stores are typed at record.Key, so Select asserts its own
// K wherever a key enters or leaves them. The stores stay honestly typed.
const Members = Store.createMembership("Selection");
const Hover = Store.createPresence("Selection.Hover");
const Focus = Store.createPresence("Selection.Focus");
const Head = Store.createPresence("Selection.Head");
const Sole = Store.createPresence("Selection.Sole");
const [ReselectNoop, useReselectNoopContext] = context.create<boolean>({
  defaultValue: false,
  displayName: "Select.ReselectNoop",
});
const [SetHover, useSetHover] = context.create<(key: record.Key) => void>({
  defaultValue: destructor.NOOP,
  displayName: "Select.SetHover",
});

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
  hover: hoverValue,
  reselectNoop = false,
  setHover = destructor.NOOP,
  children,
}: ContextProps<K>): ReactElement => (
  <Members.Context
    value={value}
    onItem={onSelect as Store.MembershipProps["onItem"]}
    setValue={setSelected as Store.MembershipProps["setValue"]}
    clear={clear}
  >
    <Focus.Context value={focusOf(value)}>
      <Head.Context value={headOf(value)}>
        <Sole.Context value={soleOf(value)}>
          <ReselectNoop value={reselectNoop}>
            <SetHover value={setHover as (key: record.Key) => void}>
              <Hover.Context value={hoverValue}>{children}</Hover.Context>
            </SetHover>
          </ReselectNoop>
        </Sole.Context>
      </Head.Context>
    </Focus.Context>
  </Members.Context>
);

/**
 * useReselectNoop returns whether the enclosing frame turns a click on an
 * already-selected item into a no-op. See {@link ContextProps.reselectNoop}.
 */
export const useReselectNoop = (): boolean => useReselectNoopContext();

/** useContext returns the enclosing selection's imperative handle. */
export const useContext = <K extends record.Key = record.Key>(): ContextValue<K> => {
  const membersCtx = Members.useContext();
  const hoverCtx = Hover.useContext();
  return {
    onSelect: membersCtx.onItem,
    setSelected: membersCtx.setValue,
    clear: membersCtx.clear,
    subscribe: membersCtx.subscribe,
    getState: () => ({
      value: membersCtx.getValue() as Value<K>,
      hover: hoverCtx.getPresent() as K | undefined,
    }),
  };
};

/**
 * useItemState subscribes a single keyed item to the enclosing Context, re-rendering only
 * when that key's selected, focused, or hovered state flips.
 */
/**
 * Reads one key's place in the enclosing selection. The caller re-renders only when
 * the dimensions it reads change, so a long list stays cheap.
 */
export const useItemState = <K extends record.Key>(key: K): UseItemStateReturn => {
  const { member, onItem } = Members.useItem(key);
  const focused = Focus.useIsPresent(key);
  const head = Head.useIsPresent(key);
  const sole = Sole.useIsPresent(key);
  const hovered = Hover.useIsPresent(key);
  const setHover = useSetHover();
  return {
    selected: member,
    focused,
    head,
    sole,
    hovered,
    onSelect: onItem,
    onHover: () => setHover(key),
  };
};

/**
 * useSelectedAmong returns the selected key among the given keys, or undefined when none
 * of them is selected. It subscribes only to the given keys, so consumers stay isolated
 * from changes to the rest of the selection. When more than one of the keys is selected,
 * the earliest in the selection's order wins.
 */
export const useSelectedAmong = <K extends record.Key = record.Key>(
  keys: K[],
): K | undefined => Members.useMemberAmong(keys) as K | undefined;

/** useSelected returns the currently selected keys. */
export const useSelected = <K extends record.Key = record.Key>(): K[] =>
  Members.useMembers() as K[];

/** useClear returns a callback that clears the enclosing selection. */
export const useClear = (): (() => void) => Members.useContext().clear;
