// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tree/Tree.css";

import { type record, unique } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { type Component } from "@/component";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { List } from "@/list";
import { Select } from "@/select";
import { state } from "@/state";
import { flatten, type Node, type Shape } from "@/tree/core";
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
}: UseProps<K>): UseReturn<K> => {
  const [expanded, setExpanded, expandedRef] =
    useCombinedStateAndRef<K[]>(initialExpanded);
  const [selected, setSelected] = state.usePassthrough<K[]>({
    initial: [],
    value: propsSelected,
    onChange: onSelectedChange,
  });
  const data = useMemo(() => flatten<K>({ nodes, expanded }), [nodes, expanded]);
  const nodesRef = useSyncedRef(nodes);

  const shiftRef = Triggers.useHeldRef({ triggers: SHIFT_TRIGGERS });

  const handleSelect: Select.UseMultipleProps<K>["onChange"] = useCallback(
    (keys: K[], { clicked }: Select.UseOnChangeExtra<K>): void => {
      console.log(keys, clicked);
      setSelected(keys);
      const n = nodesRef.current.find((node) => node.key === clicked);
      if (n?.children == null) return;
      if (clicked == null || shiftRef.current.held) return;
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
    shape: data,
    onSelect: handleSelect,
  };
};

export interface ItemProps<K extends record.Key = string>
  extends List.ItemRenderProps<K> {
  depth: number;
}

export interface TreeProps<K extends record.Key, E extends record.Keyed<K>>
  extends Omit<
    Select.FrameProps<K, E>,
    "children" | "ref" | "virtualizer" | "data" | "onChange"
  > {
  selected: Select.UseMultipleProps<K>["value"];
  onSelect: Select.UseMultipleProps<K>["onChange"];
  children: Component.RenderProp<ItemProps<K>>;
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
}: TreeProps<K, E>): ReactElement => {
  const { keys, depths } = shape;
  return (
    <Select.Frame
      multiple
      value={selected}
      data={keys}
      onChange={onSelect}
      getItem={getItem}
      subscribe={subscribe}
    >
      <List.Items<K, E>>
        {({ index, ...rest }) => children({ index, depth: depths[index], ...rest })}
      </List.Items>
    </Select.Frame>
  );
};
