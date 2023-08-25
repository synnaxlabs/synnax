// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback, useMemo, useState } from "react";

import { Icon } from "@synnaxlabs/media";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useCombinedStateAndRef } from "@/hooks/useCombinedStateAndRef";
import { UseSelectMultipleProps } from "@/hooks/useSelectMultiple";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";
import { state } from "@/state";
import { Text } from "@/text";

import "@/tree/Tree.css";

export const HAUL_TYPE = "tree-item";

export interface Node {
  key: string;
  name: string;
  icon?: ReactElement;
  editable?: boolean;
  hasChildren?: boolean;
  children?: Node[];
  haulItems?: Haul.Item[];
  canDrop?: (items: Haul.Item[]) => boolean;
}

export interface FlattenedNode extends Node {
  index: number;
  depth: number;
  expanded: boolean;
}

export interface HandleExpandProps {
  current: string[];
  action: "expand" | "contract";
  clicked: string;
}

export interface UseProps {
  onExpand?: (props: HandleExpandProps) => void;
}

export interface UseReturn {
  selected: string[];
  expanded: string[];
  onSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"];
}

export const use = (props?: UseProps): UseReturn => {
  const { onExpand } = props ?? {};
  const [expanded, setExpanded, ref] = useCombinedStateAndRef<string[]>([]);
  const [selected, setSelected] = useState<string[]>([]);

  const handleSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"] =
    useCallback(
      (keys: string[], { clicked }): void => {
        setSelected(keys);
        if (clicked == null) return;
        const currentlyExpanded = ref.current;
        const action = currentlyExpanded.some((key) => key === clicked)
          ? "contract"
          : "expand";
        let nextExpanded = currentlyExpanded;
        if (action === "contract")
          nextExpanded = currentlyExpanded.filter((key) => key !== clicked);
        else nextExpanded = [...currentlyExpanded, clicked];
        setExpanded(nextExpanded);
        onExpand?.({ current: nextExpanded, action, clicked });
      },
      [onExpand]
    );

  return {
    onSelect: handleSelect,
    expanded,
    selected,
  };
};

export interface TreeProps
  extends Pick<ItemProps, "onDrop" | "onRename" | "onSuccessfulDrop"> {
  nodes: Node[];
  selected?: string[];
  expanded?: string[];
  onSelect: (key: string[]) => void;
}

export const Tree = ({
  nodes,
  selected = [],
  expanded = [],
  onSelect,
  onDrop,
  onRename,
  onSuccessfulDrop,
}: TreeProps): ReactElement => {
  const flat = useMemo(() => flatten(nodes, expanded), [nodes, expanded]);
  return (
    <List.List<string, FlattenedNode> data={flat}>
      <List.Selector
        value={selected}
        onChange={onSelect}
        allowMultiple
        replaceOnSingle
      />
      <List.Core.Virtual<string, FlattenedNode>
        itemHeight={27}
        className={CSS.B("tree")}
      >
        {(props) => (
          <Item
            {...props}
            onDrop={onDrop}
            onRename={onRename}
            onSuccessfulDrop={onSuccessfulDrop}
            selectedItems={flat.filter((item) => selected.includes(item.key))}
          />
        )}
      </List.Core.Virtual>
    </List.List>
  );
};

interface ItemProps extends List.ItemProps<string, FlattenedNode> {
  onDrop?: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onSuccessfulDrop?: (key: string, props: Haul.OnSuccessfulDropProps) => void;
  onRename?: (key: string, name: string) => void;
  selectedItems: FlattenedNode[];
}

const expandedCaret = <Icon.Caret.Down className={CSS.B("caret")} />;
const collapsedCaret = <Icon.Caret.Right className={CSS.B("caret")} />;

const Item = ({
  entry,
  selected,
  onSelect,
  style,
  onDrop,
  onRename,
  onSuccessfulDrop,
  selectedItems,
}: ItemProps): ReactElement => {
  const {
    key,
    hasChildren = false,
    haulItems = [],
    children,
    icon,
    name,
    depth,
    expanded,
  } = entry;

  const icons: ReactElement[] = [];
  if (hasChildren || (children != null && children.length > 0))
    icons.push(expanded ? expandedCaret : collapsedCaret);
  if (icon != null) icons.push(icon);

  const [draggingOver, setDraggingOver] = useState(false);

  const { startDrag, ...dropProps } = Haul.useDragAndDrop({
    type: "Tree.Item",
    key,
    canDrop: ({ items: entities, source }) => {
      const keys = entities.map((item) => item.key);
      setDraggingOver(false);
      return source.type === "Tree.Item" && !keys.includes(key);
    },
    onDrop: (props) => onDrop?.(key, props) ?? [],
    onDragOver: () => setDraggingOver(true),
  });

  const [editable, setEditable] = useState(entry.editable ?? false);

  return (
    <Button.Button
      id={key}
      variant="text"
      draggable
      className={CSS(
        CONTEXT_TARGET,
        draggingOver && CSS.M("dragging-over"),
        selected && CONTEXT_SELECTED,
        CSS.selected(selected)
      )}
      onDragLeave={() => setDraggingOver(false)}
      onDragStart={() =>
        startDrag(
          selectedItems
            .map(({ key, haulItems }) => [
              { type: HAUL_TYPE, key },
              ...(haulItems ?? []),
            ])
            .flat(),
          (props) => onSuccessfulDrop?.(key, props)
        )
      }
      onClick={() => onSelect?.(key)}
      style={{ ...style, paddingLeft: `${depth * 1.5 + 1}rem` }}
      startIcon={icons}
      noWrap
      iconSpacing="small"
      {...dropProps}
    >
      <Text.MaybeEditable
        level="p"
        useEditableState={useCallback(
          (): state.PureUseReturn<boolean> => [editable, setEditable],
          [editable, setEditable]
        )}
        value={name}
        onChange={onRename != null ? (name) => onRename(key, name) : undefined}
      />
    </Button.Button>
  );
};

export const shouldExpand = (node: Node, expanded: string[]): boolean =>
  expanded.includes(node.key);

export const flatten = (
  nodes: Node[],
  expanded: string[],
  depth: number = 0
): FlattenedNode[] => {
  const flattened: FlattenedNode[] = [];
  nodes.forEach((node, index) => {
    const e = shouldExpand(node, expanded);
    flattened.push({ ...node, depth, expanded: e, index });
    if (e && node.children != null)
      flattened.push(...flatten(node.children, expanded, depth + 1));
  });
  return flattened;
};

export const moveNode = (
  tree: Node[],
  destination: string,
  ...keys: string[]
): Node[] => {
  keys.forEach((key) => {
    const node = findNode(tree, key);
    if (node == null) return;
    removeNode(tree, key);
    addNode(tree, destination, node);
  });
  return tree;
};

export const removeNode = (tree: Node[], ...keys: string[]): Node[] => {
  const treeKeys = tree.map((node) => node.key);
  keys.forEach((key) => {
    const index = treeKeys.indexOf(key);
    if (index !== -1) tree.splice(index, 1);
    else {
      const parent = findNodeParent(tree, key);
      if (parent != null)
        parent.children = parent.children?.filter((child) => child.key !== key);
    }
  });
  return tree;
};

export const addNode = (
  tree: Node[],
  destination: string,
  ...nodes: Node[]
): Node[] => {
  const node = findNode(tree, destination);
  if (node == null) throw new Error(`Could not find node with key ${destination}`);
  if (node.children == null) node.children = [];
  const keys = nodes.map((node) => node.key);
  node.children = [
    ...nodes,
    ...node.children.filter((child) => !keys.includes(child.key)),
  ];
  return tree;
};

export const updateNode = (
  tree: Node[],
  key: string,
  updater: (node: Node) => Node
): Node[] => {
  const node = findNode(tree, key);
  if (node == null) throw new Error(`Could not find node with key ${key}`);
  const parent = findNodeParent(tree, key);
  if (parent != null) {
    // splice the updated node into the parent's children
    const index = parent.children?.findIndex((child) => child.key === key);
    if (index != null && index !== -1) parent.children?.splice(index, 1, updater(node));
  } else {
    // we're in the root, so just update the node
    tree.splice(
      tree.findIndex((node) => node.key === key),
      1,
      updater(node)
    );
  }
  return tree;
};

export const findNode = (tree: Node[], key: string): Node | null => {
  for (const node of tree) {
    if (node.key === key) return node;
    if (node.children != null) {
      const found = findNode(node.children, key);
      if (found != null) return found;
    }
  }
  return null;
};

export const findNodes = (tree: Node[], keys: string[]): Node[] => {
  const nodes: Node[] = [];
  for (const key of keys) {
    const node = findNode(tree, key);
    if (node != null) nodes.push(node);
  }
  return nodes;
};

export const findNodeParent = (tree: Node[], key: string): Node | null => {
  for (const node of tree) {
    if (node.children != null) {
      if (node.children.some((child) => child.key === key)) return node;
      const found = findNodeParent(node.children, key);
      if (found != null) return found;
    }
  }
  return null;
};
