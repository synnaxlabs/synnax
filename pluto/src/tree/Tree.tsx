// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tree/Tree.css";

import { Icon } from "@synnaxlabs/media";
import { type Optional, unique } from "@synnaxlabs/x";
import {
  type FC,
  memo,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { type Icon as PIcon } from "@/icon";
import { List } from "@/list";
import {
  type UseSelectMultipleProps,
  type UseSelectOnChangeExtra,
  type UseSelectProps,
} from "@/list/useSelect";
import { CONTEXT_SELECTED, CONTEXT_TARGET } from "@/menu/ContextMenu";
import { state } from "@/state";
import { Text } from "@/text";
import { flatten, type FlattenedNode, type Node, type SortOption } from "@/tree/core";
import { Triggers } from "@/triggers";
import { componentRenderProp, type RenderProp } from "@/util/renderProp";

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
  sort?: SortOption;
}

export interface UseReturn {
  selected: string[];
  expanded: string[];
  onSelect: UseSelectMultipleProps<string, FlattenedNode>["onChange"];
  expand: (key: string) => void;
  contract: (...keys: string[]) => void;
  clearExpanded: () => void;
  nodes: FlattenedNode[];
}

const SHIFT_TRIGGERS: Triggers.Trigger[] = [["Shift"]];

export const use = ({
  onExpand,
  nodes,
  initialExpanded = [],
  sort,
  selected: propsSelected,
  onSelectedChange,
}: UseProps): UseReturn => {
  const [expanded, setExpanded, expandedRef] =
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
      const currentlyExpanded = expandedRef.current;
      const action = currentlyExpanded.some((key) => key === clicked)
        ? "contract"
        : "expand";
      let nextExpanded: string[];
      if (action === "contract")
        nextExpanded = currentlyExpanded.filter((key) => key !== clicked);
      else nextExpanded = [...currentlyExpanded, clicked];
      setExpanded(nextExpanded);
      onExpand?.({ current: nextExpanded, action, clicked });
    },
    [onExpand, flatRef, setExpanded, setSelected],
  );

  const handleExpand = useCallback(
    (key: string): void => {
      setExpanded((expanded) => unique.unique([...expanded, key]));
      onExpand?.({ current: expanded, action: "expand", clicked: key });
    },
    [setExpanded],
  );

  const handleContract = useCallback(
    (...keys: string[]): void => {
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
    onSelect: handleSelect,
    selected,
    expanded,
    contract: handleContract,
    expand: handleExpand,
    nodes: flat,
    clearExpanded,
  };
};

export interface ItemProps extends List.ItemProps<string, FlattenedNode> {
  key?: string;
  onDrop?: (key: string, props: Haul.OnDropProps) => Haul.Item[];
  onSuccessfulDrop?: (key: string, props: Haul.OnSuccessfulDropProps) => void;
  onRename?: (key: string, name: string) => void;
  onDoubleClick?: (key: string, e: React.MouseEvent) => void;
  loading: boolean;
  useMargin?: boolean;
  children?: RenderProp<ItemProps>;
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
    TreePropsInheritedFromList,
    Optional<UseReturn, "selected" | "expand" | "contract">,
    Pick<List.ListProps, "emptyContent"> {
  nodes: FlattenedNode[];
  children?: RenderProp<ItemProps>;
  virtual?: boolean;
  showRules?: boolean;
  loading?: string | null | false;
}

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
    useMargin = true,
    translate,
    children: childrenProp,
    index,
    sourceIndex,
    className,
    hovered,
    ...rest
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
    const { getSourceData } = List.useDataUtils<string, FlattenedNode>();

    const actuallyHasChildren =
      hasChildren || (children != null && children.length > 0);

    // Expand, contract, and loading items.
    const startIcons: PIcon.Element[] = [];
    if (actuallyHasChildren)
      startIcons.push(
        <Caret.Animated
          key="caret"
          enabled={expanded}
          enabledLoc="bottom"
          disabledLoc="right"
        />,
      );
    if (icon != null) startIcons.push(icon);
    const endIcons: PIcon.Element[] = [];
    if (loading)
      endIcons.push(
        <Icon.Loading key="loading-indicator" className={CSS.B("loading-indicator")} />,
      );

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
      const selectedItemKeys = getSelected();
      const selectedItems = getSourceData().filter((item) =>
        selectedItemKeys.includes(item.key),
      );
      if (selectedItemKeys.includes(key)) {
        const selectedHaulItems = selectedItems
          .map(({ key, haulItems, depth }) => [
            { type: HAUL_TYPE, key, data: { depth } },
            ...(haulItems?.map((item) => ({
              ...item,
              data: { ...item.data, depth },
            })) ?? []),
          ])
          .flat();
        return startDrag(selectedHaulItems, (props) => onSuccessfulDrop?.(key, props));
      }
      startDrag(
        [
          { type: HAUL_TYPE, key, data: { depth } },
          ...haulItems.map((item) => ({ ...item, data: { ...item.data, depth } })),
        ],
        (props) => onSuccessfulDrop?.(key, props),
      );
    };

    const offsetKey = useMargin ? "marginLeft" : "paddingLeft";

    let offset = depth * 2.5 + 1.5;
    if (actuallyHasChildren) offset -= 0.5;

    const baseProps: Button.LinkProps | Button.ButtonProps = {
      id: key,
      variant: "text",
      draggable: haulItems.length > 0,
      className: CSS(
        CSS.BE("list", "item"),
        CONTEXT_TARGET,
        draggingOver && CSS.M("dragging-over"),
        selected && CONTEXT_SELECTED,
        CSS.selected(selected),
        actuallyHasChildren && CSS.M("has-children"),
        CSS.BM("depth", depth.toString()),
        className,
      ),
      onDragLeave: () => setDraggingOver(false),
      onDragStart: handleDragStart,
      onClick: () => onSelect?.(key),
      style: {
        border: "none",
        position: translate != null ? "absolute" : "relative",
        transform: `translateY(${translate}px)`,
        [offsetKey]: `${offset}rem`,
        [CSS.var("tree-indicator-offset")]: `${offset - 1.5}rem`,
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
      <Base {...baseProps} align="center" {...rest}>
        {childrenProp != null ? (
          childrenProp({
            key,
            loading,
            useMargin,
            onDrop,
            onRename,
            onSuccessfulDrop,
            onDoubleClick,
            entry,
            selected,
            onSelect,
            index,
            sourceIndex,
            hovered,
          })
        ) : (
          <Text.MaybeEditable
            id={`text-${key}`}
            level="p"
            allowDoubleClick={false}
            value={name}
            disabled={!allowRename}
            onChange={(name) => onRename?.(key, name)}
          />
        )}
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
  showRules = false,
  virtual = true,
  clearExpanded,
  expand,
  contract,
  emptyContent,
  loading,
  ...rest
}: TreeProps): ReactElement => {
  const Core = virtual ? List.Core.Virtual : List.Core;
  const child: List.ItemRenderProp<string, FlattenedNode> = useCallback(
    ({ key, ...rest }) =>
      children({
        ...rest,
        key,
        loading: loading === key,
        useMargin,
        onDrop,
        onRename,
        onSuccessfulDrop,
        onDoubleClick,
      }),
    [children, loading, onDrop, onDoubleClick, onRename, onSuccessfulDrop],
  );
  return (
    <List.List<string, FlattenedNode> data={nodes} emptyContent={emptyContent}>
      <List.Selector<string, FlattenedNode>
        value={selected}
        onChange={onSelect}
        allowMultiple
        replaceOnSingle
      >
        <Core<string, FlattenedNode>
          itemHeight={itemHeight}
          className={CSS(className, CSS.B("tree"), showRules && CSS.M("rules"))}
          {...rest}
        >
          {child}
        </Core>
      </List.Selector>
    </List.List>
  );
};

export const startRenaming = (
  key: string,
  onChange?: (value: string, renamed: boolean) => void,
): void => Text.edit(`text-${key}`, onChange);

export const asyncRename = (key: string): Promise<[string, boolean]> =>
  Text.asyncEdit(`text-${key}`);
