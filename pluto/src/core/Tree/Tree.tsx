// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DetailedHTMLProps, HtmlHTMLAttributes, ReactElement, useState } from "react";

import { RenderableRecord } from "@synnaxlabs/x";
import clsx from "clsx";
import { AiFillCaretDown, AiFillCaretRight } from "react-icons/ai";

import { Button } from "@/core/Button";

import { InputControlProps } from "../Input";

import { ComponentSize } from "@/util/component";
import { RenderProp } from "@/util/renderProp";

import "./Tree.css";

export interface TreeProps<E extends RenderableRecord<E> = RenderableRecord>
  extends Partial<InputControlProps<readonly string[], string>>,
    Omit<
      DetailedHTMLProps<HtmlHTMLAttributes<HTMLUListElement>, HTMLUListElement>,
      "children" | "onChange"
    > {
  data: Array<TreeLeaf<E>>;
  onExpand?: (key: string) => void;
  children?: RenderProp<TreeLeafCProps<E>>;
  size?: ComponentSize;
}

export const Tree = <E extends RenderableRecord<E> = RenderableRecord>({
  data,
  value = [],
  onChange,
  onExpand,
  children = ButtonLeaf,
  size,
  ...props
}: TreeProps<E>): JSX.Element => {
  const _nextSiblingsHaveChildren = nextSiblingsHaveChildren(data);
  return (
    <ul className={clsx("pluto-tree__list pluto-tree__container")} {...props}>
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

export type TreeLeaf<E extends RenderableRecord<E> = RenderableRecord> = {
  hasChildren?: boolean;
  icon?: ReactElement;
  children?: Array<TreeLeaf<E>>;
  url?: string;
} & RenderableTreeLeaf<E>;

type RenderableTreeLeaf<E extends RenderableRecord<E> = RenderableRecord> = {
  key: string;
  name: string;
} & Omit<E, "name" | "key">;

type TreeLeafProps<E extends RenderableRecord<E>> = TreeLeaf<E> & {
  selected: readonly string[];
  nodeKey: string;
  hasChildren?: boolean;
  prevPaddingLeft: number;
  onExpand?: (key: string) => void;
  onSelect?: (key: string) => void;
  render: RenderProp<TreeLeafCProps<E>>;
  siblingsHaveChildren: boolean;
  size?: ComponentSize;
};

const TreeLeafParent = <E extends RenderableRecord>({
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
}: TreeLeafProps<E>): JSX.Element => {
  const [expanded, setExpanded] = useState(recursiveSelected(children, selected));
  const handleExpand = (key: string): void => {
    onExpand?.(key);
    setExpanded(!expanded);
  };
  hasChildren = children.length > 0 || hasChildren;
  let paddingLeft = prevPaddingLeft + 3.5;
  if (!hasChildren && siblingsHaveChildren) paddingLeft += 3;
  const _nextSiblingsHaveChildren = nextSiblingsHaveChildren(children);
  return (
    <li className="tree-node__container">
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
      } as const as TreeLeafCProps<E>)}
      {expanded && children.length > 0 && (
        <ul className="pluto-tree__list">
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

type TreeLeafCProps<E extends RenderableRecord<E> = RenderableRecord> = Omit<
  RenderableTreeLeaf<E>,
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

export const ButtonLeaf = <E extends RenderableRecord<E>>({
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
}: TreeLeafCProps<E>): JSX.Element => {
  const icons: ReactElement[] = [];
  if (hasChildren) icons.push(expanded ? <AiFillCaretDown /> : <AiFillCaretRight />);
  if (icon != null) icons.push(icon);

  const handleClick = (): void => {
    onSelect?.(nodeKey);
    onExpand(nodeKey);
  };

  const _Button = url != null ? Button.Link : Button;

  return (
    <_Button
      href={url}
      variant="text"
      className={clsx(
        "pluto-tree-leaf__button",
        selected && "pluto-tree-leaf__button--selected"
      )}
      startIcon={icons}
      onClick={handleClick}
      {...props}
    >
      {name}
    </_Button>
  );
};

const recursiveSelected = (data: TreeLeaf[], selected: readonly string[]): boolean => {
  for (const entry of data) {
    if (selected.includes(entry.key)) return true;
    if (entry.children != null && recursiveSelected(entry.children, selected))
      return true;
  }
  return false;
};

const nextSiblingsHaveChildren = (data: TreeLeaf[]): boolean =>
  data.some(
    (child) =>
      child.hasChildren === true ||
      (child.children != null && child.children.length > 0)
  );
