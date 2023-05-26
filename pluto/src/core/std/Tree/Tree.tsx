// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DetailedHTMLProps,
  HtmlHTMLAttributes,
  ReactElement,
  useEffect,
  useState,
} from "react";

import { Icon } from "@synnaxlabs/media";
import { KeyedRenderableRecord } from "@synnaxlabs/x";

import "./Tree.css";

import { CSS } from "@/core/css";
import { Haul } from "@/core/haul";
import { Button, ButtonLinkProps, ButtonProps } from "@/core/std/Button";
import { InputControl } from "@/core/std/Input";
import { List, ListItemProps } from "@/core/std/List";
import { ComponentSize } from "@/util/component";
import { componentRenderProp } from "@/util/renderProp";

export type TreeLeaf<E extends KeyedRenderableRecord<E> = KeyedRenderableRecord> = {
  hasChildren?: boolean;
  icon?: ReactElement;
  children?: Array<TreeLeaf<E>>;
  url?: string;
  expanded?: boolean;
} & RenderableTreeLeaf<E>;

type RenderableTreeLeaf<E extends KeyedRenderableRecord<E> = KeyedRenderableRecord> = {
  key: string;
  name: string;
} & Omit<E, "name" | "key">;

export interface TreeProps<E extends KeyedRenderableRecord<E> = KeyedRenderableRecord>
  extends Partial<InputControl<readonly string[], string>>,
    Omit<
      DetailedHTMLProps<HtmlHTMLAttributes<HTMLDivElement>, HTMLDivElement>,
      "children" | "onChange"
    > {
  data: Array<TreeLeaf<E>>;
  onExpand?: (key: string) => void;
  size?: ComponentSize;
}

export const Tree = <E extends KeyedRenderableRecord<E> = KeyedRenderableRecord>({
  data,
  value = [],
  onChange,
  onExpand,
  size,
  ...props
}: TreeProps<E>): ReactElement => {
  const [tree, setTree] = useState<TreeLeaf<E>>({
    key: "root",
    name: "Root",
  } as TreeLeaf<E>);

  const normalized = tree.children?.map((t) => normalize(t, 1)).flat() ?? [];

  useEffect(() => {
    setTree(merge(tree, { key: "root", name: "Root", children: data } as TreeLeaf<E>));
  }, [data]);

  const handleSelect = ([key]: readonly string[]) => {
    const next = update(
      tree,
      key,
      ({ expanded, ...prev }) => ({ ...prev, expanded: !expanded } as TreeLeaf<E>)
    );
    setTree(next);
  };

  return (
    <List<TreeLeafListItem<E>> data={normalized}>
      <List.Selector onChange={handleSelect} value={[]} />
      <List.Core.Virtual<TreeLeafListItem<E>> itemHeight={24} {...props}>
        {componentRenderProp(TreeLeafListItemC<E>)}
      </List.Core.Virtual>
    </List>
  );
};

export type TreeLeafListItem<
  E extends KeyedRenderableRecord<E> = KeyedRenderableRecord
> = {
  padding: number;
} & TreeLeaf<E>;

export const normalize = <E extends KeyedRenderableRecord<E>>(
  tree: TreeLeaf<E>,
  padding: number
): Array<TreeLeafListItem<E>> => {
  tree.hasChildren =
    tree.hasChildren || (tree.children != null && tree.children.length) > 0;
  const items: Array<TreeLeafListItem<E>> = [
    {
      ...tree,
      padding,
    },
  ];
  const nextPadding = padding + 2;
  if (tree.hasChildren && tree.expanded === true) {
    tree.children?.forEach((child) => {
      items.push(...normalize(child, nextPadding));
    });
  }
  return items;
};

export const update = <E extends KeyedRenderableRecord<E>>(
  tree: TreeLeaf<E>,
  key: string,
  set: (leaf: TreeLeaf<E>) => TreeLeaf<E>
): TreeLeaf<E> => {
  if (tree.key == key) return set(tree);
  else if (tree.children?.length ?? 0 > 0)
    return {
      ...tree,
      children: tree.children?.map((child) => update(child, key, set)),
    };
  return tree;
};

export const merge = <E extends KeyedRenderableRecord<E>>(
  prev: TreeLeaf<E>,
  next: TreeLeaf<E>
): TreeLeaf<E> => {
  const merged = { ...prev, ...next };
  if (prev.children == null || next.children == null) return merged;
  merged.children = prev.children.map((child) => {
    const nextChild = next.children?.find((c) => c.key === child.key);
    return nextChild != null ? merge(child, nextChild) : child;
  });
  const keys = merged.children.map((c) => c.key);
  merged.children.push(...next.children.filter((c) => !keys.includes(c.key)));
  return merged;
};

const DRAGGING_TYPE = "pluto-tree-list-item";

export const TreeLeafListItemC = <E extends KeyedRenderableRecord<E>>({
  entry: { padding, hasChildren, icon, name, url, expanded, key: nodeKey },
  onSelect,
  selected,
  ...props
}: ListItemProps<TreeLeafListItem<E>>): ReactElement => {
  const icons: ReactElement[] = [];
  if (hasChildren) icons.push(expanded ? <Icon.Caret.Down /> : <Icon.Caret.Right />);
  if (icon != null) icons.push(icon);

  const handleClick = (): void => onSelect?.(nodeKey);

  const { startDrag, endDrag, dragging } = Haul.useRef();

  const [draggedOver, setDraggedOver] = useState<boolean>(false);

  const handleDragOver = () =>
    dragging.current.some((v) => v.type === DRAGGING_TYPE) && setDraggedOver(true);

  const handleDragLeave = () => setDraggedOver(false);

  const handleDragStart = () => startDrag([{ key: nodeKey, type: DRAGGING_TYPE }]);

  const baseProps: ButtonLinkProps | ButtonProps = {
    variant: "text",
    onClick: handleClick,
    className: CSS(CSS.BE("tree-leaf", "button"), CSS.selected(selected)),
    startIcon: icons,
    iconSpacing: "small",
    noWrap: true,
    draggable: true,
    onDragLeave: handleDragLeave,
    onDragOver: handleDragOver,
    onDragStart: handleDragStart,
    onDragEnd: endDrag,
    ...props,
    style: {
      paddingLeft: `${padding}rem`,
      ...props.style,
      backgroundColor: draggedOver ? "red" : undefined,
    },
  };

  return url != null ? (
    <Button.Link href={url} {...baseProps}>
      {name}
    </Button.Link>
  ) : (
    <Button {...baseProps}>{name}</Button>
  );
};
