// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type compare, type record, type state as xstate, unique } from "@synnaxlabs/x";
import {
  type ReactElement,
  type Ref,
  useCallback,
  useImperativeHandle,
  useMemo,
  useRef,
} from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { List } from "@/list";
import { Select } from "@/select";
import { state } from "@/state";
import { flatten, getNodeShape, type Node, type Shape } from "@/tree/base";
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
  /**
   * Scrolls the node with the given key into view. The tree must have rendered the
   * node first.
   * @throws {Error} if the node is not in the rendered tree.
   */
  scrollTo: ScrollTo<K>;
  /** Connects scrollTo to the rendered {@link Tree}. */
  scrollToRef: Ref<ScrollTo<K>>;
}

export type ScrollTo<K extends record.Key = string> = (key: K) => void;

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
  const shape = useMemo(
    () => flatten<K>({ nodes, expanded, sort }),
    [nodes, expanded, sort],
  );
  const shapeRef = useSyncedRef(shape);

  const shiftRef = Triggers.useHeldRef({ triggers: SHIFT_TRIGGERS });

  const handleToggle = useCallback(
    (key: K): void => {
      const n = getNodeShape(shapeRef.current, key);
      if (n == null || !n.hasChildren) return;
      const currentlyExpanded = expandedRef.current;
      const action = currentlyExpanded.includes(key) ? "contract" : "expand";
      const nextExpanded =
        action === "contract"
          ? currentlyExpanded.filter((k) => k !== key)
          : [...currentlyExpanded, key];
      setExpanded(nextExpanded);
      onExpand?.({ current: nextExpanded, action, clicked: key });
    },
    [onExpand, setExpanded],
  );

  const handleSelect: Select.UseMultipleProps<K>["onChange"] = useCallback(
    (keys: K[], { clicked }: Select.UseOnChangeExtra<K>): void => {
      setSelected((p): K[] => {
        if (keys.length === 0 && p.length > 0) return p.slice(0, 1);
        return keys;
      });
      if (toggleOn === "caret" || clicked == null || shiftRef.current.held) return;
      handleToggle(clicked);
    },
    [toggleOn, handleToggle, setSelected],
  );

  const handleExpand = useCallback(
    (key: K): void => {
      setExpanded((expanded) => unique.unique([...expanded, key]));
      onExpand?.({ current: expanded, action: "expand", clicked: key });
    },
    [setExpanded],
  );

  const handleContract = useCallback(
    (...keys: K[]): void => {
      setExpanded((expanded) => expanded.filter((k) => !keys.includes(k)));
      // Call onExpand for each contracted key
      keys.forEach((key) => {
        onExpand?.({ current: expanded, action: "contract", clicked: key });
      });
    },
    [setExpanded],
  );

  const clearExpanded = useCallback(() => setExpanded([]), [setExpanded]);

  const scrollToRef = useRef<ScrollTo<K>>(null);
  const scrollTo = useCallback((key: K): void => {
    if (scrollToRef.current == null) throw new Error("the tree is not mounted");
    scrollToRef.current(key);
  }, []);

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
    scrollTo,
    scrollToRef,
  };
};

export interface ItemRenderProps<
  K extends record.Key = string,
> extends List.ItemRenderProps<K> {}

export interface TreeProps<K extends record.Key, E extends record.Keyed<K>>
  extends
    Omit<
      Select.FrameProps<K, E>,
      "children" | "ref" | "virtualizer" | "data" | "onChange"
    >,
    Omit<List.ScrollProps, "children" | "onSelect">,
    Pick<List.ItemsProps<K>, "emptyContent">,
    UseReturn<K> {
  children: Component.RenderProp<ItemRenderProps<K>>;
  showRules?: boolean;
  shape: Shape<K>;
}

const ITEM_HEIGHT = 27;

interface ScrollerProps<K extends record.Key> {
  keys: K[];
  ref: Ref<ScrollTo<K>>;
}

const Scroller = <K extends record.Key>({ keys, ref }: ScrollerProps<K>): null => {
  const { scrollToIndex } = List.useScroller();
  useImperativeHandle(
    ref,
    () => (key: K) => {
      const index = keys.indexOf(key);
      if (index === -1) throw new Error(`node ${String(key)} is not in the tree`);
      scrollToIndex(index);
    },
    [keys, scrollToIndex],
  );
  return null;
};

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
  scrollTo: ______,
  scrollToRef,
  ...rest
}: TreeProps<K, E>): ReactElement => {
  const { keys, nodes } = shape;
  const contextValue = useMemo(
    () => ({
      nodes,
      toggle: toggleOn === "caret" ? (i: number) => toggle(keys[i]) : undefined,
    }),
    [nodes, keys, toggle, toggleOn],
  );
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
        allowNone={allowNone}
        autoSelectOnNone={autoSelectOnNone}
      >
        <Scroller keys={keys} ref={scrollToRef} />
        <List.Scroll
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
        </List.Scroll>
      </Select.Frame>
    </Context>
  );
};
