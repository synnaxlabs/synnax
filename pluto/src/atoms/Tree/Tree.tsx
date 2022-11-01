import { DetailedHTMLProps, HtmlHTMLAttributes, ReactElement, useState } from "react";
import { AiFillCaretDown, AiFillCaretRight } from "react-icons/ai";
import "./Tree.css";
import { Button, ButtonProps } from "@/atoms/Button";
import { useMultiSelect, useMultiSelectProps } from "../List/useMultiSelect";
import clsx from "clsx";

export interface TreeProps
	extends useMultiSelectProps<TreeLeaf>,
		Omit<
			DetailedHTMLProps<HtmlHTMLAttributes<HTMLUListElement>, HTMLUListElement>,
			"onSelect"
		> {
	data: TreeLeaf[];
	selected?: string[];
	onExpand?: (key: string) => void;
}

export const Tree = ({
	data,
	selected: propsSelected,
	onSelect: propsOnSelect,
	onExpand,
	...props
}: TreeProps) => {
	const { selected, onSelect } = useMultiSelect<TreeLeaf>({
		selectMultiple: false,
		selected: propsSelected,
		onSelect: propsOnSelect,
		data: data,
	});

	return (
		<ul className={clsx("pluto-tree__list pluto-tree__container")} {...props}>
			{data.map((entry) => (
				<TreeLeafC
					{...entry}
					key={entry.key}
					nodeKey={entry.key}
					selected={selected}
					onSelect={onSelect}
					onExpand={onExpand}
				/>
			))}
		</ul>
	);
};

export type TreeLeaf = {
	key: string;
	title: string;
	hasChildren?: boolean;
	icon?: ReactElement;
	children?: TreeLeaf[];
};

interface TreeLeafProps extends Omit<TreeLeaf, "key"> {
	onSelect: (key: string) => void;
	selected: string[];
	nodeKey: string;
	hasChildren?: boolean;
	onExpand?: (key: string) => void;
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
}: TreeLeafProps) => {
	const [expanded, setExpanded] = useState(false);
	return (
		<li className="tree-node__container">
			<TreeNodeButton
				selected={selected.includes(nodeKey)}
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
						<TreeLeafC
							{...child}
							key={child.key}
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
	showExpandIcon,
	...props
}: TreeNodeButtonProps) => {
	const icons: ReactElement[] = [];
	if (showExpandIcon) icons.push(expanded ? <AiFillCaretDown /> : <AiFillCaretRight />);
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
