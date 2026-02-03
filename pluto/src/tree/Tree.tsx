// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type compare, type record, unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { List } from "@/list";
import { Select } from "@/select";
import { state } from "@/state";
import { flatten, getNodeShape, type Node, type Shape } from "@/tree/base";
import { Context } from "@/tree/Context";
import { Triggers } from "@/triggers";

export const HAUL_TYPE = "tree-item";

export interface HandleExpandProps<K extends record.Key = string> {
  current: K[];
  action: "expand" | "contract";
  clicked: K;
}

export interface UseProps<K extends record.Key = string> {
  onExpand?: (props: HandleExpandProps<K>) => void;
  selected?: K[];
  sort?: compare.Comparator<Node<K>>;
  onSelectedChange?: state.Setter<K[]>;
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
  const nodesRef = useSyncedRef(nodes);
  const shapeRef = useSyncedRef(shape);

  const shiftRef = Triggers.useHeldRef({ triggers: SHIFT_TRIGGERS });

  const handleSelect: Select.UseMultipleProps<K>["onChange"] = useCallback(
    (keys: K[], { clicked }: Select.UseOnChangeExtra<K>): void => {
      setSelected((p): K[] => {
        if (keys.length === 0 && p.length > 0) return p.slice(0, 1);
        return keys;
      });
      if (clicked == null || shiftRef.current.held) return;
      const n = getNodeShape(shapeRef.current, clicked);
      if (n == null || !n.hasChildren) return;
      const currentlyExpanded = expandedRef.current;
      const action = currentlyExpanded.some((key) => key === clicked)
        ? "contract"
        : "expand";
      let nextExpanded: K[];
      if (action === "contract")
        nextExpanded = currentlyExpanded.filter((key) => key !== clicked);
      else nextExpanded = [...currentlyExpanded, clicked];
      setExpanded(nextExpanded);
      onExpand?.({ current: nextExpanded, action, clicked });
    },
    [onExpand, nodesRef, setExpanded, setSelected],
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

  return {
    selected,
    expanded,
    contract: handleContract,
    expand: handleExpand,
    clearExpanded,
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
      "children" | "ref" | "virtualizer" | "data" | "onChange"
    >,
    Omit<List.ItemsProps<K>, "children" | "onSelect">,
    UseReturn<K> {
  children: Component.RenderProp<ItemRenderProps<K>>;
  showRules?: boolean;
  shape: Shape<K>;
}

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
  showRules = false,
  virtual = false,
  ...rest
}: TreeProps<K, E>): ReactElement => {
  const { keys } = shape;
  return (
    <Context value={shape.nodes}>
      <Select.Frame
        multiple
        value={selected}
        replaceOnSingle
        data={keys}
        onChange={onSelect}
        getItem={getItem}
        subscribe={subscribe}
        itemHeight={27}
        virtual={virtual}
      >
        <List.Items<K, E>
          full="y"
          className={CSS(CSS.B("tree"), className, showRules && CSS.M("show-rules"))}
          {...rest}
        >
          {children}
        </List.Items>
      </Select.Frame>
    </Context>
  );
};
