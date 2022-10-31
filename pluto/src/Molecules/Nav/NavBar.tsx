import clsx from "clsx";
import React, {
  CSSProperties,
  ComponentType,
  HTMLAttributes,
  useContext,
} from "react";
import { Space, SpaceProps } from "@/atoms";
import {
  Direction,
  Location,
  Position,
  getDirection,
  swapLocation,
} from "@/util";
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

export const NavBar = ({
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

export interface NavbarContentProps extends SpaceProps {
  bordered?: boolean;
}

const contentFactory = (
  pos: Position | ""
): ComponentType<NavbarContentProps> => {
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

NavBar.Start = contentFactory("start");
NavBar.Center = contentFactory("center");
NavBar.End = contentFactory("end");
NavBar.Content = contentFactory("");
NavBar.Context = NavbarContext;

NavBar.useContext = () => useContext(NavbarContext);
