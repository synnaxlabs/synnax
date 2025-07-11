// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tree/Tree.css";

import { type record } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { type Component } from "@/component";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { List } from "@/list";
import { Select } from "@/select";
import { type state } from "@/state";
import { flatten, type Node } from "@/tree/core";
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

const SHIFT_TRIGGERS: Triggers.Trigger[] = [["Shift"]];

export interface ItemProps<K extends record.Key = string>
  extends List.ItemRenderProps<K> {
  depth: number;
}

export interface TreeProps<K extends record.Key, E extends record.Keyed<K>>
  extends Omit<
      Select.MultipleFrameProps<K, E>,
      "children" | "ref" | "virtualizer" | "data"
    >,
    Omit<List.ItemsProps<K>, "children" | "onChange"> {
  children: Component.RenderProp<ItemProps<K>>;
  showRules?: boolean;
  initialExpanded?: K[];
  onExpand?: (props: HandleExpandProps<K>) => void;
  nodes: Node<K>[];
}

export const Tree = <K extends record.Key, E extends record.Keyed<K>>({
  children,
  nodes,
  initialExpanded = [],
  onExpand,
  useListItem,
}: TreeProps<K, E>): ReactElement => {
  const nodesRef = useSyncedRef(nodes);
  const shiftRef = Triggers.useHeldRef({ triggers: SHIFT_TRIGGERS });

  const [expanded, setExpanded, expandedRef] = useCombinedStateAndRef(initialExpanded);
  const [selected, setSelected] = useCombinedStateAndRef<K[]>([]);

  const handleSelect: Select.UseMultipleProps<K>["onChange"] = useCallback(
    (keys: K[], { clicked }: Select.UseOnChangeExtra<K>): void => {
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

  const { keys, depths } = useMemo(
    () => flatten<K>({ nodes, expanded }),
    [nodes, expanded],
  );

  return (
    <Select.Frame<K, E>
      multiple
      data={keys}
      value={selected}
      onChange={handleSelect}
      useListItem={useListItem}
    >
      <List.Items<K, E>>
        {({ index, ...rest }) => children({ index, depth: depths[index], ...rest })}
      </List.Items>
    </Select.Frame>
  );
};
