import clsx from "clsx";
import React, {
	CSSProperties,
	HTMLAttributes,
	PropsWithChildren,
	useContext,
} from "react";
import { Space, SpaceProps } from "@/atoms";
import { Direction, Location, Position, getDirection, swapLocation } from "@/util";
import "./NavBar.css";

export interface NavBarProps extends HTMLAttributes<HTMLDivElement> {
	location: Location;
	size?: string | number;
	withContext?: boolean;
}

const NavbarContext = React.createContext<{
	location?: Location;
	direction?: Direction;
}>({});

export const useNavBar = ({
	location,
	size,
}: NavBarProps): {
	style: CSSProperties;
	direction: Direction;
} => {
	const style: CSSProperties = {};
	const direction = getDirection(location);
	if (direction === "horizontal") {
		style.height = size;
	} else {
		style.width = size;
	}
	return { style, direction };
};

const CoreNavBar = ({
	location,
	size = 60,
	withContext = true,
	children,
	...props
}: NavBarProps) => {
	const { style, direction } = useNavBar({ location, size });
	const content = withContext ? (
		<NavbarContext.Provider value={{ location, direction }}>
			{children}
		</NavbarContext.Provider>
	) : (
		children
	);
	return (
		<Space
			className={clsx(
				"pluto-navbar",
				`pluto-bordered--${swapLocation(location)}`,
				`pluto-navbar--${getDirection(location)}`
			)}
			direction={direction}
			style={style}
			align="center"
			empty
			{...props}
		>
			{content}
		</Space>
	);
};

export interface NavbarContentProps
	extends PropsWithChildren<HTMLAttributes<HTMLDivElement>> {
	bordered?: boolean;
	className?: string;
	children: React.ReactNode;
}

const contentFactory = (pos: Position | "") => {
	const Content = ({
		children,
		bordered = true,
		className,
		...props
	}: NavbarContentProps) => {
		const { direction } = useContext(NavbarContext);
		return (
			<Space
				className={clsx(
					"pluto-navbar__content",
					`pluto-navbar__content--${pos}`,
					bordered && "pluto-navbar__content--bordered",
					className
				)}
				direction={direction}
				align="center"
				{...props}
			>
				{children}
			</Space>
		);
	};
	return Content;
};

type CoreNavBarType = typeof CoreNavBar;

const useNavBarContext = () => useContext(NavbarContext);

const NavBarStart = contentFactory("start");
const NavBarEnd = contentFactory("end");
const NavBarCenter = contentFactory("center");
const NavBarContent = contentFactory("");

export interface NavBarType extends CoreNavBarType {
	Start: typeof NavBarStart;
	Center: typeof NavBarCenter;
	End: typeof NavBarEnd;
	Content: typeof NavBarContent;
	Context: typeof NavbarContext;
	useContext: typeof useNavBarContext;
}

export const NavBar = CoreNavBar as NavBarType;

NavBar.Start = NavBarStart;
NavBar.Center = NavBarCenter;
NavBar.End = NavBarEnd;
NavBar.Content = NavBarContent;
NavBar.Context = NavbarContext;
NavBar.useContext = useNavBarContext;
