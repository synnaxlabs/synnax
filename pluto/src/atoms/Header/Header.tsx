import clsx from "clsx";
import { Fragment, ReactElement, cloneElement, isValidElement } from "react";
import { Button, ButtonIconOnlyProps } from "@/atoms/Button";
import { Divider } from "@/atoms/Divider";
import { Space, SpaceProps } from "@/atoms/Space";
import {
	Text,
	TextProps,
	TypographyLevel,
	TypographyLevelComponentSizes,
} from "@/atoms/Typography";
import "./Header.css";

export interface HeaderProps extends Omit<SpaceProps, "children">, TextProps {
	divided?: boolean;
	icon?: ReactElement;
	actions?: (ButtonIconOnlyProps | ReactElement)[];
}

export const Header = ({
	icon,
	level = "h1",
	divided = false,
	children,
	className,
	actions,
	...props
}: HeaderProps) => {
	return (
		<Space
			direction="horizontal"
			justify="spaceBetween"
			className={clsx(`pluto-header pluto-bordered--bottom`, className)}
			empty
			{...props}
		>
			<Text.WithIcon
				level={level}
				startIcon={icon}
				divided={divided}
				className="pluto-header__text"
			>
				{children}
			</Text.WithIcon>
			<Space
				direction="horizontal"
				size="small"
				align="center"
				className="pluto-header__actions"
			>
				{actions && actions.map((action, i) => renderAction(i, level, action, divided))}
			</Space>
		</Space>
	);
};

export interface HeaderButtonProps extends Omit<HeaderProps, "onClick"> {
	onClick?: (e: React.MouseEvent<HTMLButtonElement>) => void;
}

export const HeaderButton = ({
	icon,
	level = "h1",
	divided = false,
	children = "",
	className,
	actions,
	onClick,
	...props
}: HeaderButtonProps) => (
	<Space
		direction="horizontal"
		justify="spaceBetween"
		className={clsx(`pluto-header pluto-bordered--bottom`, className)}
		empty
		{...props}
	>
		<Button
			variant="text"
			size={TypographyLevelComponentSizes[level]}
			startIcon={icon}
			style={{ flexGrow: 1 }}
			onClick={onClick}
		>
			{children}
		</Button>
		{actions && (
			<Space
				direction="horizontal"
				size="small"
				align="center"
				className="pluto-header__actions"
			>
				{actions.map((action, i) => renderAction(i, level, action, divided))}
			</Space>
		)}
	</Space>
);

const renderAction = (
	index: number,
	level: TypographyLevel,
	action: ReactElement | ButtonIconOnlyProps,
	divided: boolean
) => {
	const content = isValidElement(action) ? (
		cloneElement(action, { key: action.key })
	) : (
		<Button.IconOnly
			onClick={(e) => {
				e.stopPropagation();
				e.preventDefault();
				action.onClick && action.onClick(e);
			}}
			key={index}
			size={TypographyLevelComponentSizes[level]}
			{...action}
		>
			{action.children}
		</Button.IconOnly>
	);
	return (
		<Fragment key={index}>
			{divided && <Divider direction="vertical" />}
			{content}
		</Fragment>
	);
};
