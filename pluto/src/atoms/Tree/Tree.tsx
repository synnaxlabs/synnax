import { DetailedHTMLProps, HtmlHTMLAttributes, ReactElement, useState } from "react";

import clsx from "clsx";
import { AiFillCaretDown, AiFillCaretRight } from "react-icons/ai";

import "./Tree.css";
import {
  useSelectMultiple,
  UseSelectMultipleProps,
} from "../../hooks/useSelectMultiple";

import { Button, ButtonProps } from "@/atoms/Button";

export interface TreeProps
  extends Omit<UseSelectMultipleProps<RenderableTreeLeaf>, "allowMultiple">,
    Omit<
      DetailedHTMLProps<HtmlHTMLAttributes<HTMLUListElement>, HTMLUListElement>,
      "onChange"
    > {
  data: TreeLeaf[];
  value: readonly string[];
  onExpand?: (key: string) => void;
}

export const Tree = ({
  data,
  value,
  onChange,
  onExpand,
  ...props
}: TreeProps): JSX.Element => {
  const { onSelect } = useSelectMultiple<RenderableTreeLeaf>({
    allowMultiple: false,
    data,
    value,
    onChange,
  });

  return (
    <ul className={clsx("pluto-tree__list pluto-tree__container")} {...props}>
      {data.map((entry) => (
        <TreeLeafC
          {...entry}
          key={entry.key}
          depth={1}
          selected={value}
          nodeKey={entry.key}
          onSelect={onSelect}
          onExpand={onExpand}
        />
      ))}
    </ul>
  );
};

export interface TreeLeaf {
  key: string;
  title: string;
  hasChildren?: boolean;
  icon?: ReactElement;
  children?: TreeLeaf[];
}

type RenderableTreeLeaf = Omit<TreeLeaf, "icon" | "children" | "hasChildren">;

interface TreeLeafProps extends Omit<TreeLeaf, "key"> {
  onSelect: (key: string) => void;
  selected: readonly string[];
  nodeKey: string;
  hasChildren?: boolean;
  onExpand?: (key: string) => void;
  depth: number;
}

const TreeLeafC = ({
  nodeKey,
  title,
  icon,
  onSelect,
  selected,
  children = [],
  hasChildren,
  onExpand,
  depth,
}: TreeLeafProps): JSX.Element => {
  const [expanded, setExpanded] = useState(false);
  return (
    <li className="tree-node__container">
      <TreeNodeButton
        style={{ paddingLeft: `${depth * 2}rem` }}
        selected={selected.includes(nodeKey)}
        title={title}
        icon={icon}
        expanded={expanded}
        showExpandIcon={children.length > 0 || hasChildren}
        onClick={() => {
          onExpand?.(nodeKey);
          setExpanded(!expanded);
          onSelect(nodeKey);
        }}
      />
      {expanded && children.length > 0 && (
        <ul className="pluto-tree__list">
          {children.map((child) => (
            <TreeLeafC
              {...child}
              key={child.key}
              nodeKey={child.key}
              onSelect={onSelect}
              depth={depth + 1}
              selected={selected}
              onExpand={onExpand}
            />
          ))}
        </ul>
      )}
    </li>
  );
};

export interface TreeNodeButtonProps extends Omit<ButtonProps, "children" | "level"> {
  title: string;
  expanded: boolean;
  selected: boolean;
  showExpandIcon?: boolean;
  icon?: ReactElement;
}

const TreeNodeButton = ({
  title,
  icon,
  selected,
  expanded,
  showExpandIcon = true,
  ...props
}: TreeNodeButtonProps): JSX.Element => {
  const icons: ReactElement[] = [];
  if (showExpandIcon) icons.push(expanded ? <AiFillCaretDown /> : <AiFillCaretRight />);
  if (icon != null) icons.push(icon);
  return (
    <Button
      variant="text"
      className={clsx(
        "pluto-tree__node__button",
        selected && "pluto-tree__node__button--selected"
      )}
      startIcon={icons}
      {...props}
    >
      {title}
    </Button>
  );
};
