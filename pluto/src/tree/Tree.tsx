// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DetailedHTMLProps, HtmlHTMLAttributes, ReactElement, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Input } from "@/input";
import { ComponentSize } from "@/util/component";
import { RenderProp } from "@/util/renderProp";

import "@/tree/Tree.css";

export interface TreeProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Partial<Input.Control<readonly string[], string>>,
    Omit<
      DetailedHTMLProps<HtmlHTMLAttributes<HTMLUListElement>, HTMLUListElement>,
      "children" | "onChange"
    > {
  data: Array<Node<K, E>>;
  onExpand?: (key: string) => void;
  children?: RenderProp<TreeLeafCProps<K, E>>;
  size?: ComponentSize;
}

export const Tree = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  data,
  value = [],
  onChange,
  onExpand,
  children = ButtonLeaf,
  size,
  ...props
}: TreeProps<K, E>): JSX.Element => {
  const _nextSiblingsHaveChildren = nextSiblingsHaveChildren(data);
  return (
    <ul className={CSS(CSS.BE("tree", "list"), CSS.BE("tree", "container"))} {...props}>
      {data.map((entry) => (
        <TreeLeafParent
          {...entry}
          key={entry.key}
          prevPaddingLeft={-1.5}
          siblingsHaveChildren={_nextSiblingsHaveChildren}
          selected={value}
          nodeKey={entry.key}
          onSelect={onChange}
          onExpand={onExpand}
          render={children}
          size={size}
        />
      ))}
    </ul>
  );
};

export type Node<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> = {
  hasChildren?: boolean;
  icon?: ReactElement;
  children?: Array<Node<K, E>>;
  url?: string;
} & RenderableNode<K, E>;

type RenderableNode<K extends Key, E extends KeyedRenderableRecord<K, E>> = {
  key: string;
  name: string;
} & Omit<E, "name" | "key">;

type LeafProps<K extends Key, E extends KeyedRenderableRecord<K, E>> = Node<K, E> & {
  selected: readonly string[];
  nodeKey: string;
  hasChildren?: boolean;
  prevPaddingLeft: number;
  onExpand?: (key: string) => void;
  onSelect?: (key: string) => void;
  render: RenderProp<TreeLeafCProps<K, E>>;
  siblingsHaveChildren: boolean;
  size?: ComponentSize;
};

const TreeLeafParent = <K extends Key, E extends KeyedRenderableRecord<K>>({
  nodeKey,
  name,
  icon,
  onSelect,
  selected,
  children = [],
  hasChildren = false,
  onExpand,
  prevPaddingLeft,
  render,
  siblingsHaveChildren,
  size,
  ...rest
}: LeafProps<K, E>): JSX.Element => {
  const [expanded, setExpanded] = useState(recursiveSelected(children, selected));
  const handleExpand = (key: string): void => {
    onExpand?.(key);
    setExpanded(!expanded);
  };
  hasChildren = children.length > 0 || hasChildren;
  let paddingLeft = prevPaddingLeft + 2.5;
  if (!hasChildren && siblingsHaveChildren) paddingLeft += 3.25;
  const _nextSiblingsHaveChildren = nextSiblingsHaveChildren(children);
  return (
    <li className={CSS.BE("tree-node", "container")}>
      {render({
        nodeKey,
        style: {
          paddingLeft: `${paddingLeft}rem`,
        },
        selected: selected.includes(nodeKey),
        name,
        icon,
        expanded,
        hasChildren,
        onExpand: handleExpand,
        onSelect,
        size,
        ...rest,
      } as const as TreeLeafCProps<K, E>)}
      {expanded && children.length > 0 && (
        <ul className={CSS.BE("tree", "list")}>
          {children.map((child) => (
            <TreeLeafParent
              {...child}
              key={child.key}
              nodeKey={child.key}
              siblingsHaveChildren={_nextSiblingsHaveChildren}
              onSelect={onSelect}
              prevPaddingLeft={paddingLeft}
              selected={selected}
              onExpand={onExpand}
              render={render}
            />
          ))}
        </ul>
      )}
    </li>
  );
};

type TreeLeafCProps<K extends Key, E extends KeyedRenderableRecord<K>> = Omit<
  RenderableNode<K, E>,
  "key"
> & {
  nodeKey: string;
  name: string;
  expanded: boolean;
  selected: boolean;
  hasChildren: boolean;
  icon?: ReactElement;
  url?: string;
  style: React.CSSProperties;
  onExpand: (key: string) => void;
  onSelect?: (key: string) => void;
};

export const ButtonLeaf = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  name,
  icon,
  nodeKey,
  selected,
  expanded,
  hasChildren = true,
  onSelect,
  onExpand,
  url,
  ...props
}: TreeLeafCProps<K, E>): JSX.Element => {
  const icons: ReactElement[] = [];
  if (hasChildren) icons.push(expanded ? <Icon.Caret.Down /> : <Icon.Caret.Right />);
  if (icon != null) icons.push(icon);

  const handleClick = (): void => {
    onSelect?.(nodeKey);
    onExpand(nodeKey);
  };

  const baseProps: Button.LinkProps | Button.ButtonExtensionProps = {
    variant: "text",
    onClick: handleClick,
    className: CSS(CSS.BE("tree-leaf", "button"), CSS.selected(selected)),
    startIcon: icons,
    iconSpacing: "small",
    noWrap: true,
    ...props,
  };

  return url != null ? (
    // @ts-expect-error
    <Button.Link href={url} {...baseProps} level="p">
      {name}
    </Button.Link>
  ) : (
    <Button.Button {...baseProps}>{name}</Button.Button>
  );
};

const recursiveSelected = (data: Node[], selected: readonly string[]): boolean => {
  for (const entry of data) {
    if (selected.includes(entry.key)) return true;
    if (entry.children != null && recursiveSelected(entry.children, selected))
      return true;
  }
  return false;
};

const nextSiblingsHaveChildren = (data: Node[]): boolean =>
  data.some(
    (child) =>
      child.hasChildren === true ||
      (child.children != null && child.children.length > 0)
  );
