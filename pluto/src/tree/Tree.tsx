// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tree/Tree.css";

import { Icon } from "@synnaxlabs/media";
import {
  type FC,
  memo,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useCombinedStateAndRef,useSyncedRef } from "@/hooks";
import { List } from "@/list";
import {
  UseSelectMultipleProps,
  type UseSelectOnChangeExtra,
  type UseSelectProps,
} from "@/list/useSelect";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";
import { state } from "@/state";
import { Text } from "@/text";
import { flatten, type FlattenedNode,type Node } from "@/tree/core";
import { Triggers } from "@/triggers";
import { componentRenderProp,type RenderProp } from "@/util/renderProp";

export const HAUL_TYPE = "tree-item";

export interface HandleExpandProps {
  current: string[];
  action: "expand" | "contract";
  clicked: string;
}

export interface UseProps {
  onExpand?: (props: HandleExpandProps) => void;
  selected?: string[];
  onSelectedChange?: state.Set<string[]>;
  initialExpanded?: string[];
  nodes: Node[];
  sort?: boolean;
}

export interface UseReturn {
  selected: string[];
  expanded: string[];
  onSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"];
  nodes: FlattenedNode[];
}

const SHIFT_TRIGGERS: Triggers.Trigger[] = [["Shift"]];

export const use = (props: UseProps): UseReturn => {
  const {
    onExpand,
    nodes,
    initialExpanded = [],
    sort = true,
    selected: propsSelected,
    onSelectedChange,
  } = props ?? {};
  const [expanded, setExpanded, ref] =
    useCombinedStateAndRef<string[]>(initialExpanded);
  const [selected, setSelected] = state.usePassthrough<string[]>({
    initial: [],
    value: propsSelected,
    onChange: onSelectedChange,
  });
  const flat = useMemo(
    () => flatten({ nodes, expanded, sort }),
    [nodes, expanded, sort],
  );
  const flatRef = useSyncedRef(flat);

  const shiftRef = Triggers.useHeldRef({ triggers: SHIFT_TRIGGERS });

  const handleSelect: UseSelectProps<string, FlattenedNode>["onChange"] = useCallback(
    (
      keys: string[],
      { clicked }: UseSelectOnChangeExtra<string, FlattenedNode>,
    ): void => {
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
    expanded,
    nodes: flat,
  };
};

export interface ItemProps extends List.ItemProps<string, FlattenedNode> {
  onDrop?: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onSuccessfulDrop?: (key: string, props: Haul.OnSuccessfulDropProps) => void;
  onRename?: (key: string, name: string) => void;
  onDoubleClick?: (key: string, e: React.MouseEvent) => void;
  loading?: boolean;
  useMargin?: boolean;
}

type TreePropsInheritedFromItem = Pick<
  ItemProps,
  "onDrop" | "onRename" | "onSuccessfulDrop" | "onDoubleClick" | "useMargin"
>;

type TreePropsInheritedFromList = Omit<
  List.VirtualCoreProps<string, FlattenedNode>,
  "onDrop" | "onSelect" | "children" | "onDoubleClick" | "itemHeight"
> & {
  itemHeight?: number;
};

export interface TreeProps
  extends TreePropsInheritedFromItem,
    TreePropsInheritedFromList {
  nodes: FlattenedNode[];
  selected?: string[];
  onSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"];
  children?: RenderProp<ItemProps>;
  virtual?: boolean;
}

const expandedCaret = <Icon.Caret.Down className={CSS.B("caret")} />;
const collapsedCaret = <Icon.Caret.Right className={CSS.B("caret")} />;

export type Item = FC<ItemProps>;

export const DefaultItem = memo(
  ({
    entry,
    selected,
    onSelect,
    onDrop,
    onRename,
    onSuccessfulDrop,
    onDoubleClick,
    loading = false,
    useMargin = false,
    translate,
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

    const { getSelected } = List.useSelectionUtils<string>();
    const { getSourceData } = List.useDataUtilContext<string, FlattenedNode>();

    const actuallyHasChildren =
      hasChildren || (children != null && children.length > 0);

    // Expand, contract, and loading items.
    const startIcons: ReactElement[] = [];
    if (actuallyHasChildren) startIcons.push(expanded ? expandedCaret : collapsedCaret);
    if (icon != null) startIcons.push(icon);
    const endIcons: ReactElement[] = [];
    if (loading) endIcons.push(<Icon.Loading className={CSS.B("loading-indicator")} />);

    const [draggingOver, setDraggingOver] = useState(false);

    // Drag and Drop
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
      const selectedItemKeys = getSelected();
      const selectedItems = getSourceData().filter((item) =>
        selectedItemKeys.includes(item.key),
      );
      if (selectedItemKeys.includes(key)) {
        const selectedHaulItems = selectedItems
          .map(({ key, haulItems }) => [{ type: HAUL_TYPE, key }, ...(haulItems ?? [])])
          .flat();
        return startDrag(selectedHaulItems, (props) => onSuccessfulDrop?.(key, props));
      }
      startDrag([{ type: HAUL_TYPE, key }, ...haulItems], (props) =>
        onSuccessfulDrop?.(key, props),
      );
    };

    const offsetKey = useMargin ? "marginLeft" : "paddingLeft";

    const baseProps: Button.LinkProps | Button.ButtonProps = {
      id: key,
      variant: "text",
      draggable: true,
      className: CSS(
        CSS.BE("list", "item"),
        CONTEXT_TARGET,
        draggingOver && CSS.M("dragging-over"),
        selected && CONTEXT_SELECTED,
        CSS.selected(selected),
        actuallyHasChildren && CSS.M("has-children"),
      ),
      onDragLeave: () => setDraggingOver(false),
      onDragStart: handleDragStart,
      onClick: () => onSelect?.(key),
      style: {
        position: translate != null ? "absolute" : "relative",
        transform: `translateY(${translate}px)`,
        [offsetKey]: `${depth * 1.5 + 1}rem`,
      },
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
      <Base className={CSS.BE("list", "item")} {...baseProps}>
        <Text.MaybeEditable
          id={`text-${key}`}
          level="p"
          allowDoubleClick={false}
          value={name}
          disabled={!allowRename}
          onChange={(name) => onRename?.(key, name)}
        />
      </Base>
    );
  },
);
DefaultItem.displayName = "Tree.Item";

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
  virtual = true,
  ...props
}: TreeProps): ReactElement => {
  const Core = virtual ? List.Core.Virtual : List.Core;

  return (
    <List.List<string, FlattenedNode> data={nodes}>
      <List.Selector value={selected} onChange={onSelect} allowMultiple replaceOnSingle>
        <Core<string, FlattenedNode>
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
              onDoubleClick,
            })
          }
        </Core>
      </List.Selector>
    </List.List>
  );
};

export const startRenaming = (key: string): void => Text.edit(`text-${key}`);
