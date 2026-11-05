// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type compare,
  type location,
  type record,
  type state as xstate,
  unique,
} from "@synnaxlabs/x";
import { type FocusEvent, type KeyboardEvent, type ReactElement } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { List } from "@/list";
import { Select } from "@/select";
import { state } from "@/state";
import { findNode, flatten, type Node, type Shape } from "@/tree/base";
import { Context } from "@/tree/Context";
import { Triggers } from "@/triggers";

export const HAUL_TYPE = "tree_item";

export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, undefined>;

export const createHaulItem = (key: string): HaulItem => ({ type: HAUL_TYPE, key });

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter(isHaulItem);

export const canDropHaulItem = Haul.canDropOfType<HaulItem>(HAUL_TYPE);

export interface HandleExpandProps<K extends record.Key = string> {
  current: K[];
  action: "expand" | "contract";
  clicked: K;
}

/**
 * What folds and unfolds a node: a click anywhere on its row, which also selects it,
 * or a click on its caret alone, which leaves the selection as it is.
 */
export type ToggleOn = "row" | "caret";

export interface UseProps<K extends record.Key = string> {
  onExpand?: (props: HandleExpandProps<K>) => void;
  toggleOn?: ToggleOn;
  selected?: K[];
  sort?: compare.Comparator<Node<K>>;
  onSelectedChange?: xstate.Setter<K[]>;
  initialExpanded?: K[];
  nodes: Node<K>[];
}

export interface UseReturn<K extends record.Key = string> {
  selected: K[];
  onSelect: Select.UseMultipleProps<K>["onChange"];
  expanded: K[];
  expand: (key: K) => void;
  contract: (...keys: K[]) => void;
  clearExpanded: () => void;
  /** Folds an expanded node or unfolds a collapsed one. */
  toggle: (key: K) => void;
  toggleOn: ToggleOn;
  shape: Shape<K>;
}

const SHIFT_TRIGGERS: Triggers.Trigger[] = [["Shift"]];

export const use = <K extends record.Key = string>({
  onExpand,
  nodes,
  initialExpanded = [],
  selected: propsSelected,
  onSelectedChange,
  sort,
  toggleOn = "row",
}: UseProps<K>): UseReturn<K> => {
  const [expanded, setExpanded, expandedRef] =
    useCombinedStateAndRef<K[]>(initialExpanded);
  const [selected, setSelected] = state.usePassthrough<K[]>({
    initial: [],
    value: propsSelected,
    onChange: onSelectedChange,
  });
  const shape = flatten<K>({ nodes, expanded, sort });
  const nodesRef = useSyncedRef(nodes);

  const shiftRef = Triggers.useHeldRef({ triggers: SHIFT_TRIGGERS });

  // A caller can show a shape other than this hook's, such as a filtered one, so the
  // node is found in the whole tree.
  const handleToggle = (key: K): void => {
    const n = findNode({ tree: nodesRef.current, key });
    if (n?.children == null) return;
    const currentlyExpanded = expandedRef.current;
    const action = currentlyExpanded.includes(key) ? "contract" : "expand";
    const nextExpanded =
      action === "contract"
        ? currentlyExpanded.filter((k) => k !== key)
        : [...currentlyExpanded, key];
    setExpanded(nextExpanded);
    onExpand?.({ current: nextExpanded, action, clicked: key });
  };

  const handleSelect: Select.UseMultipleProps<K>["onChange"] = (
    keys: K[],
    { clicked }: Select.UseOnChangeExtra<K>,
  ): void => {
    setSelected((p): K[] => {
      if (keys.length === 0 && p.length > 0) return p.slice(0, 1);
      return keys;
    });
    if (toggleOn === "caret" || clicked == null || shiftRef.current.held) return;
    handleToggle(clicked);
  };

  const handleExpand = (key: K): void => {
    setExpanded((expanded) => unique.unique([...expanded, key]));
    onExpand?.({ current: expandedRef.current, action: "expand", clicked: key });
  };

  const handleContract = (...keys: K[]): void => {
    setExpanded((expanded) => expanded.filter((k) => !keys.includes(k)));
    const current = expandedRef.current;
    keys.forEach((key) => onExpand?.({ current, action: "contract", clicked: key }));
  };

  const clearExpanded = () => setExpanded([]);

  return {
    selected,
    expanded,
    contract: handleContract,
    expand: handleExpand,
    clearExpanded,
    toggle: handleToggle,
    toggleOn,
    shape,
    onSelect: handleSelect,
  };
};

export interface ItemRenderProps<
  K extends record.Key = string,
> extends List.ItemRenderProps<K> {}

export interface TreeProps<K extends record.Key, E extends record.Keyed<K>>
  extends
    Omit<
      Select.FrameProps<K, E>,
      "children" | "ref" | "virtualizer" | "data" | "onChange" | "cursor" | "moveX"
    >,
    Omit<List.ScrollProps, "children" | "onSelect">,
    Pick<List.ItemsProps<K>, "emptyContent">,
    UseReturn<K> {
  children: Component.RenderProp<ItemRenderProps<K>>;
  showRules?: boolean;
  shape: Shape<K>;
}

const ITEM_HEIGHT = 27;

// A consumer's handlers run first, so its preventDefault keeps a key from the tree.
const Scroll = ({
  onKeyDown,
  onKeyUp,
  onBlur,
  ...rest
}: List.ScrollProps): ReactElement => {
  const keyboard = Select.useKeyboard();
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    onKeyDown?.(e);
    keyboard.onKeyDown(e);
  };
  const handleKeyUp = (e: KeyboardEvent<HTMLDivElement>) => {
    onKeyUp?.(e);
    keyboard.onKeyUp(e);
  };
  const handleBlur = (e: FocusEvent<HTMLDivElement>) => {
    onBlur?.(e);
    keyboard.onBlur();
  };
  return (
    <List.Scroll
      tabIndex={0}
      {...rest}
      onKeyDown={handleKeyDown}
      onKeyUp={handleKeyUp}
      onBlur={handleBlur}
    />
  );
};

/**
 * Renders the nodes of a {@link Shape}. The tree takes focus for its items, and its
 * keys move the selection as in any `Select.Frame` whose `cursor` is `"selection"`.
 * Right also expands a node or moves into it, and Left collapses a node or moves to its
 * parent.
 */
export const Tree = <K extends record.Key, E extends record.Keyed<K>>({
  shape,
  children,
  selected,
  onSelect,
  getItem,
  subscribe,
  className,
  contract: _,
  expand: __,
  expanded: ___,
  className: ____,
  clearExpanded: _____,
  toggle,
  toggleOn,
  showRules = false,
  virtual = true,
  itemHeight = ITEM_HEIGHT,
  overscan,
  onFetchMore,
  allowNone,
  autoSelectOnNone,
  emptyContent,
  pinned,
  ...rest
}: TreeProps<K, E>): ReactElement => {
  const { keys, nodes } = shape;
  // Reads keys at click time, so an expand keeps unchanged items cached.
  const keysRef = useSyncedRef(keys);
  const toggleAt = (index: number) => toggle(keysRef.current[index]);
  const contextValue = { nodes, toggle: toggleOn === "caret" ? toggleAt : undefined };
  const moveX = (key: K, direction: location.X): K | undefined => {
    const at = keys.indexOf(key);
    if (at === -1) return undefined;
    const { depth, expanded, hasChildren } = nodes[at];
    const opens = direction === "right";
    if (hasChildren && expanded !== opens) {
      toggle(key);
      return undefined;
    }
    if (opens)
      return at < keys.length - 1 && nodes[at + 1].depth > depth
        ? keys[at + 1]
        : undefined;
    for (let i = at - 1; i >= 0; i--) if (nodes[i].depth < depth) return keys[i];
    return undefined;
  };
  return (
    <Context value={contextValue}>
      <Select.Frame
        multiple
        value={selected}
        replaceOnSingle
        data={keys}
        onChange={onSelect}
        getItem={getItem}
        subscribe={subscribe}
        itemHeight={itemHeight}
        overscan={overscan}
        onFetchMore={onFetchMore}
        virtual={virtual}
        pinned={pinned}
        allowNone={allowNone}
        autoSelectOnNone={autoSelectOnNone}
        cursor="selection"
        moveX={moveX}
      >
        <Scroll
          full="y"
          role="tree"
          className={CSS.cls(
            CSS.B("tree"),
            className,
            showRules && CSS.M("show-rules"),
          )}
          {...rest}
        >
          <List.Items<K, E> emptyContent={emptyContent}>{children}</List.Items>
        </Scroll>
      </Select.Frame>
    </Context>
  );
};
