import clsx from "clsx";
import React, {
  ComponentType,
  CSSProperties,
  HTMLAttributes,
  ReactElement,
  ReactNode,
  useContext,
} from "react";
import Space, { SpaceProps } from "../../Atoms/Space/Space";
import "./Navbar.css";

type Location = "top" | "bottom" | "left" | "right";
type Direction = "horizontal" | "vertical";

export interface NavbarProps extends HTMLAttributes<HTMLDivElement> {
  location: Location;
  size?: string | number;
  context?: boolean;
}

const getDirection = (location: Location) => {
  return location === "top" || location === "bottom"
    ? "horizontal"
    : "vertical";
};

const NavbarContext = React.createContext<{
  location?: Location;
  direction?: Direction;
}>({});

const useNavbarContext = () => useContext(NavbarContext);

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

export default function Navbar({
  location,
  size = 60,
  context = true,
  children,
  ...props
}: NavbarProps) {
  const { style, direction } = useNavbar({ location, size });
  let content: ReactNode;
  if (context) {
    content = (
      <NavbarContext.Provider value={{ location, direction }}>
        {children}
      </NavbarContext.Provider>
    );
  } else {
    content = children;
  }
  return (
    <Space
      className={clsx(
        "pluto-navbar",
        `pluto-navbar--${location}`,
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
}

export interface NavbarContentProps extends SpaceProps {
  bordered?: boolean;
}

const contentFactory = (
  position: string
): ComponentType<NavbarContentProps> => {
  return ({
    children,
    bordered = true,
    className,
    ...props
  }: NavbarContentProps) => {
    const { direction } = useNavbarContext();
    return (
      <Space
        className={clsx(
          "pluto-navbar__content",
          `pluto-navbar__content--${position}`,
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
};

Navbar.Start = contentFactory("start");
Navbar.Center = contentFactory("center");
Navbar.End = contentFactory("end");
Navbar.Content = contentFactory("");
Navbar.Context = NavbarContext;
