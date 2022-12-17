import React, {
  ComponentType,
  CSSProperties,
  HTMLAttributes,
  PropsWithChildren,
  useContext,
} from "react";

import clsx from "clsx";

import { Space } from "@/atoms";
import { Direction, Location, Position, getDirection, swapLocation } from "@/util";
import "./Navbar.css";

export interface NavbarProps extends HTMLAttributes<HTMLDivElement> {
  location: Location;
  size?: string | number;
  withContext?: boolean;
}

export interface NavbarContextValue {
  location?: Location;
  direction?: Direction;
}

const NavbarContext = React.createContext<NavbarContextValue>({});

export const useNavbar = ({
  location,
  size,
}: NavbarProps): {
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

const CoreNavbar = ({
  location,
  size = 60,
  withContext = true,
  children,
  ...props
}: NavbarProps): JSX.Element => {
  const { style, direction } = useNavbar({ location, size });
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

const contentFactory = (pos: Position | ""): ComponentType<NavbarContentProps> => {
  const Content = ({
    children,
    bordered = true,
    className,
    ...props
  }: NavbarContentProps): JSX.Element => {
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

type CoreNavbarType = typeof CoreNavbar;

const useNavbarContext = (): NavbarContextValue => useContext(NavbarContext);

const NavbarStart = contentFactory("start");
const NavbarEnd = contentFactory("end");
const NavbarCenter = contentFactory("center");
const NavbarContent = contentFactory("");

export interface NavbarType extends CoreNavbarType {
  Start: typeof NavbarStart;
  Center: typeof NavbarCenter;
  End: typeof NavbarEnd;
  Content: typeof NavbarContent;
  Context: typeof NavbarContext;
  useContext: typeof useNavbarContext;
}

export const Navbar = CoreNavbar as NavbarType;

Navbar.Start = NavbarStart;
Navbar.Center = NavbarCenter;
Navbar.End = NavbarEnd;
Navbar.Content = NavbarContent;
Navbar.Context = NavbarContext;
Navbar.useContext = useNavbarContext;
