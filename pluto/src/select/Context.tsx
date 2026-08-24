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
  /**
   * True when re-selecting an already-selected item changes nothing: the frame
   * forbids emptying the selection and a select closes no dialog. Items use it
   * to drop interaction feedback that would promise a change.
   */
  reselectNoop?: boolean;
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
}

// The membership and presence stores are created monomorphically at record.Key; Select
// layers its own K back on with an assertion at each use. The stores stay honestly
// typed — the untyped seam lives in these two accessors.
const Members = Store.createMembership("Selection");
const Hover = Store.createPresence("Selection.Hover");
const Focus = Store.createPresence("Selection.Focus");
const Head = Store.createPresence("Selection.Head");
const Sole = Store.createPresence("Selection.Sole");
const [ReselectNoop, useReselectNoopContext] = context.create<boolean>({
  defaultValue: false,
  displayName: "Select.ReselectNoop",
});
const members = <K extends record.Key>(): Store.Membership<K> =>
  Members as unknown as Store.Membership<K>;
const hover = <K extends record.Key>(): Store.Presence<K> =>
  Hover as unknown as Store.Presence<K>;

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
  children,
}: ContextProps<K>): ReactElement => {
  const M = members<K>();
  const H = hover<K>();
  return (
    <M.Context value={value} onItem={onSelect} setValue={setSelected} clear={clear}>
      <Focus.Context value={focusOf(value)}>
        <Head.Context value={headOf(value)}>
          <Sole.Context value={soleOf(value)}>
            <ReselectNoop value={reselectNoop}>
              <H.Context value={hoverValue}>{children}</H.Context>
            </ReselectNoop>
          </Sole.Context>
        </Head.Context>
      </Focus.Context>
    </M.Context>
  );
};

/**
 * useReselectNoop returns whether the enclosing frame turns a click on an
 * already-selected item into a no-op. See {@link ContextProps.reselectNoop}.
 */
export const useReselectNoop = (): boolean => useReselectNoopContext();

/** useContext returns the enclosing selection's imperative handle. */
export const useContext = <K extends record.Key = record.Key>(): ContextValue<K> => {
  const membersCtx = members<K>().useContext();
  const hoverCtx = hover<K>().useContext();
  return useMemo<ContextValue<K>>(
    () => ({
      onSelect: membersCtx.onItem,
      setSelected: membersCtx.setValue,
      clear: membersCtx.clear,
      subscribe: membersCtx.subscribe,
      getState: () => ({
        value: membersCtx.getValue(),
        hover: hoverCtx.getPresent(),
      }),
    }),
    [membersCtx, hoverCtx],
  );
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
  const { member, onItem } = members<K>().useItem(key);
  const focused = Focus.useIsPresent(key);
  const head = Head.useIsPresent(key);
  const sole = Sole.useIsPresent(key);
  const hovered = hover<K>().useIsPresent(key);
  return useMemo(
    () => ({ selected: member, focused, head, sole, hovered, onSelect: onItem }),
    [member, focused, head, sole, hovered, onItem],
  );
};

/**
 * useSelectedAmong returns the selected key among the given keys, or undefined when none
 * of them is selected. It subscribes only to the given keys, so consumers stay isolated
 * from changes to the rest of the selection. When more than one of the keys is selected,
 * the earliest in the selection's order wins.
 */
export const useSelectedAmong = <K extends record.Key = record.Key>(
  keys: K[],
): K | undefined => members<K>().useMemberAmong(keys);

/** useSelected returns the currently selected keys. */
export const useSelected = <K extends record.Key = record.Key>(): K[] =>
  members<K>().useMembers();

/** useClear returns a callback that clears the enclosing selection. */
export const useClear = (): (() => void) => Members.useContext().clear;
