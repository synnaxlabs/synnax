import {
  DetailedHTMLProps,
  HtmlHTMLAttributes,
  ReactElement,
  useState,
} from "react";
import { Text } from "../Typography";
import {
  AiFillCaretDown,
  AiFillCaretRight,
  AiFillCaretUp,
} from "react-icons/ai";
import "./Tree.css";
import { Button } from "../Button";
import { ButtonProps } from "../Button/Button";
import { useMultiSelect, useMultiSelectProps } from "../List/useMultiSelect";
import clsx from "clsx";

export interface TreeProps
  extends useMultiSelectProps<TreeEntry>,
    Omit<
      DetailedHTMLProps<HtmlHTMLAttributes<HTMLUListElement>, HTMLUListElement>,
      "onSelect"
    > {
  data: TreeEntry[];
  selected?: string[];
  onExpand?: (key: string) => void;
}

const Tree = ({
  data,
  selected: propsSelected,
  onSelect: propsOnSelect,
  onExpand,
  className,
  ...props
}: TreeProps) => {
  const { selected, onSelect } = useMultiSelect<TreeEntry>({
    selectMultiple: false,
    selected: propsSelected,
    onSelect: propsOnSelect,
    data: data,
  });

  return (
    <ul className={clsx("pluto-tree__list pluto-tree__container")} {...props}>
      {data.map((entry) => (
        <TreeNode
          {...entry}
          nodeKey={entry.key}
          selected={selected}
          onSelect={onSelect}
          onExpand={onExpand}
        />
      ))}
    </ul>
  );
};

export interface TreeEntry {
  key: string;
  title: string;
  hasChildren?: boolean;
  icon?: ReactElement;
  children?: TreeEntry[];
}

export interface TreeNodeProps extends Omit<TreeEntry, "key"> {
  onSelect: (key: string) => void;
  selected: string[];
  nodeKey: string;
  hasChildren?: boolean;
  onExpand?: (key: string) => void;
}

const TreeNode = ({
  nodeKey,
  title,
  icon,
  onSelect,
  selected,
  children = [],
  hasChildren,
  onExpand,
}: TreeNodeProps) => {
  const [expanded, setExpanded] = useState(false);
  return (
    <li className="tree-node__container">
      <TreeNodeButton
        selected={selected.includes(nodeKey)}
        nodeKey={nodeKey}
        title={title}
        icon={icon}
        expanded={expanded}
        showExpandIcon={children.length > 0 || hasChildren}
        onClick={() => {
          onExpand && onExpand(nodeKey);
          setExpanded(!expanded);
          onSelect(nodeKey);
        }}
      />
      {expanded && children.length > 0 && (
        <ul className="pluto-tree__list">
          {children.map((child) => (
            <TreeNode
              {...child}
              nodeKey={child.key}
              onSelect={onSelect}
              selected={selected}
              onExpand={onExpand}
            />
          ))}
        </ul>
      )}
    </li>
  );
};

export interface TreeNodeButtonProps
  extends Omit<ButtonProps, "children" | "level"> {
  nodeKey: string;
  title: string;
  expanded: boolean;
  selected: boolean;
  showExpandIcon?: boolean;
  icon?: ReactElement;
}

const TreeNodeButton = ({
  nodeKey,
  title,
  icon,
  selected,
  expanded,
  showExpandIcon,
  ...props
}: TreeNodeButtonProps) => {
  let icons: ReactElement[] = [];
  if (showExpandIcon)
    icons.push(expanded ? <AiFillCaretDown /> : <AiFillCaretRight />);
  if (icon) icons.push(icon);
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

export default Tree;
