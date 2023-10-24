// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback, useMemo, useState, type FC } from "react";

import { Icon } from "@synnaxlabs/media";
import { type Optional } from "@synnaxlabs/x";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useSyncedRef } from "@/hooks";
import { useCombinedStateAndRef } from "@/hooks/useCombinedStateAndRef";
import { type UseSelectMultipleProps } from "@/hooks/useSelectMultiple";
import { List } from "@/list";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { type RenderProp, componentRenderProp } from "@/util/renderProp";

import "@/tree/Tree.css";

export const HAUL_TYPE = "tree-item";

export interface Node {
  key: string;
  name: string;
  icon?: ReactElement;
  allowRename?: boolean;
  hasChildren?: boolean;
  children?: Node[];
  haulItems?: Haul.Item[];
  canDrop?: (items: Haul.Item[]) => boolean;
  href?: string;
}

export interface NodeWithDepth extends Node {
  depth: number;
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
  initialExpanded?: string[];
  nodes: Node[];
}

export interface UseReturn {
  selected: string[];
  expanded: string[];
  onSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"];
  flat: FlattenedNode[];
}

export const use = (props: UseProps): UseReturn => {
  const { onExpand, nodes, initialExpanded = [] } = props ?? {};
  const [expanded, setExpanded, ref] =
    useCombinedStateAndRef<string[]>(initialExpanded);
  const [selected, setSelected] = useState<string[]>([]);
  const flat = useMemo(() => flatten(nodes, expanded), [nodes, expanded]);
  const flatRef = useSyncedRef(flat);

  const shiftRef = Triggers.useHeldRef({ triggers: [["Shift"]] });

  const handleSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"] =
    useCallback(
      (keys: string[], { clicked }): void => {
        setSelected(keys);
        const n = flatRef.current.find((node) => node.key === clicked);
        if (n?.hasChildren === false) return;
        if (clicked == null || shiftRef.current.held) return;
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
      [onExpand, flatRef, setExpanded, setSelected],
    );

  return {
    onSelect: handleSelect,
    selected,
    nodes: flat,
  };
};

export interface ItemProps extends List.ItemProps<string, FlattenedNode> {
  onDrop?: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onSuccessfulDrop?: (key: string, props: Haul.OnSuccessfulDropProps) => void;
  onRename?: (key: string, name: string) => void;
  onDoubleClick?: (key: string, e: React.MouseEvent) => void;
  loading?: boolean;
  selectedItems: FlattenedNode[];
  useMargin?: boolean;
}

export interface TreeProps
  extends Pick<
      ItemProps,
      "onDrop" | "onRename" | "onSuccessfulDrop" | "onDoubleClick" | "useMargin"
    >,
    Optional<
      Omit<
        List.VirtualCoreProps<string, FlattenedNode>,
        "onDrop" | "onSelect" | "children" | "onDoubleClick"
      >,
      "itemHeight"
    > {
  nodes: FlattenedNode[];
  selected?: string[];
  onSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"];
  children?: RenderProp<ItemProps>;
}

const expandedCaret = <Icon.Caret.Down className={CSS.B("caret")} />;
const collapsedCaret = <Icon.Caret.Right className={CSS.B("caret")} />;

export type Item = FC<ItemProps>;

export const DefaultItem = ({
  entry,
  selected,
  onSelect,
  style,
  onDrop,
  onRename,
  onSuccessfulDrop,
  selectedItems,
  onDoubleClick,
  loading = false,
  useMargin = false,
}: ItemProps): ReactElement => {
  const {
    key,
    hasChildren = false,
    allowRename = false,
    children,
    icon,
    name,
    depth,
    expanded,
    href,
    haulItems = [],
  } = entry;

  const actuallyHasChildren = hasChildren || (children != null && children.length > 0);

  const startIcons: ReactElement[] = [];
  if (actuallyHasChildren) startIcons.push(expanded ? expandedCaret : collapsedCaret);
  if (icon != null) startIcons.push(icon);
  const endIcons: ReactElement[] = [];
  if (loading) endIcons.push(<Icon.Loading className={CSS.B("loading-indicator")} />);

  const [draggingOver, setDraggingOver] = useState(false);

  const { startDrag, ...dropProps } = Haul.useDragAndDrop({
    type: "Tree.Item",
    key,
    canDrop: useCallback(({ items: entities, source }) => {
      const keys = entities.map((item) => item.key);
      setDraggingOver(false);
      return source.type === "Tree.Item" && !keys.includes(key);
    }, []),
    onDrop: useCallback((props) => onDrop?.(key, props) ?? [], [key, onDrop]),
    onDragOver: useCallback(() => setDraggingOver(true), []),
  });

  const handleDragStart = (): void => {
    const selectedItemKeys = selectedItems.map(({ key }) => key);
    if (selectedItemKeys.includes(key)) {
      const selectedHaulItems = selectedItems
        .map(({ key, haulItems }) => [{ type: HAUL_TYPE, key }, ...(haulItems ?? [])])
        .flat();
      return startDrag(selectedHaulItems, (props) => onSuccessfulDrop?.(key, props));
    }
    startDrag(
      [{ type: HAUL_TYPE, key }, ...haulItems],
      (props) => onSuccessfulDrop?.(key, props),
    );
  };

  const offsetKey = useMargin ? "marginLeft" : "paddingLeft";

  const baseProps: Button.LinkProps | Button.ButtonProps = {
    id: key,
    variant: "text",
    draggable: true,
    className: CSS(
      CONTEXT_TARGET,
      draggingOver && CSS.M("dragging-over"),
      selected && CONTEXT_SELECTED,
      CSS.selected(selected),
      actuallyHasChildren && CSS.M("has-children"),
    ),
    onDragLeave: () => setDraggingOver(false),
    onDragStart: handleDragStart,
    onClick: () => onSelect?.(key),
    style: { ...style, [offsetKey]: `${depth * 1.5 + 1}rem` },
    startIcon: startIcons,
    iconSpacing: "small",
    noWrap: true,
    endIcon: endIcons,
    onDoubleClick: (e) => onDoubleClick?.(key, e),
    href,
    ...dropProps,
  };

  const Base = href != null ? Button.Link : Button.Button;

  return (
    <Base {...baseProps}>
      <Text.MaybeEditable
        id={`text-${key}`}
        level="p"
        allowDoubleClick={false}
        value={name}
        onChange={
          onRename != null && allowRename ? (name) => onRename(key, name) : undefined
        }
      />
    </Base>
  );
};

const defaultChild = componentRenderProp(DefaultItem);
export const Tree = ({
  nodes,
  selected = [],
  onSelect,
  onDrop,
  onRename,
  onSuccessfulDrop,
  onDoubleClick,
  className,
  children = defaultChild,
  itemHeight = 27,
  useMargin = false,
  ...props
}: TreeProps): ReactElement => {
  return (
    <List.List<string, FlattenedNode> data={nodes}>
      <List.Selector
        value={selected}
        onChange={onSelect}
        allowMultiple
        replaceOnSingle
      />
      <List.Core.Virtual<string, FlattenedNode>
        itemHeight={itemHeight}
        className={CSS(className, CSS.B("tree"))}
        {...props}
      >
        {(props) =>
          children({
            ...props,
            useMargin,
            onDrop,
            onRename,
            onSuccessfulDrop,
            selectedItems: nodes.filter((item) => selected.includes(item.key)),
            onDoubleClick,
          })
        }
      </List.Core.Virtual>
    </List.List>
  );
};

export const startRenaming = (key: string): void => Text.edit(`text-${key}`);

export const shouldExpand = (node: Node, expanded: string[]): boolean =>
  expanded.includes(node.key);

export const flatten = (
  nodes: Node[],
  expanded: string[],
  depth: number = 0,
): FlattenedNode[] => {
  // Sort the first level of the tree independently of the rest
  if (depth === 0) nodes = nodes.sort((a, b) => a.name.localeCompare(b.name));
  const flattened: FlattenedNode[] = [];
  nodes.forEach((node, index) => {
    const expand = shouldExpand(node, expanded);
    flattened.push({ ...node, depth, expanded: expand, index });
    if (expand && node.children != null) {
      node.children = node.children.sort((a, b) => a.name.localeCompare(b.name));
      flattened.push(...flatten(node.children, expanded, depth + 1));
    }
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
    setNode(tree, destination, node);
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

export const setNode = (
  tree: Node[],
  destination: string,
  ...additions: Node[]
): Node[] => {
  const node = findNode(tree, destination);
  if (node == null) throw new Error(`Could not find node with key ${destination}`);
  if (node.children == null) node.children = [];
  const addedKeys = additions.map((node) => node.key);
  node.children = [
    ...additions,
    ...node.children.filter((child) => !addedKeys.includes(child.key)),
  ];
  return tree;
};

export const updateNode = (
  tree: Node[],
  key: string,
  updater: (node: Node) => Node,
  throwOnMissing: boolean = true,
): Node[] => {
  const node = findNode(tree, key);
  if (node == null) {
    if (throwOnMissing) throw new Error(`Could not find node with key ${key}`);
    return tree;
  }
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
      updater(node),
    );
  }
  return tree;
};

export const findNode = (
  tree: Node[],
  key: string,
  depth: number = 0,
): NodeWithDepth | null => {
  for (const node of tree) {
    if (node.key === key) {
      const n = node as NodeWithDepth;
      n.depth = depth;
      return n;
    }
    if (node.children != null) {
      const found = findNode(node.children, key, depth + 1);
      if (found != null) return found;
    }
  }
  return null;
};

export const findNodes = (tree: Node[], keys: string[]): NodeWithDepth[] => {
  const nodes: NodeWithDepth[] = [];
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
