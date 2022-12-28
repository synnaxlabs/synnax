import { Children, createContext, ReactNode, useContext } from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "../Space";
import { TypographyLevel } from "../Typography";

export interface HeaderProps extends Omit<SpaceProps, "children"> {
  level?: TypographyLevel;
  divided?: boolean;
  children: ReactNode | [ReactNode, ReactNode];
}

export interface HeaderContextValue {
  divided: boolean;
  level: TypographyLevel;
}

const HeaderContext = createContext<HeaderContextValue>({
  divided: false,
  level: "h1",
});

export const useHeaderContext = (): HeaderContextValue => useContext(HeaderContext);

export const Header = ({
  children,
  className,
  level = "h1",
  divided = false,
  ...props
}: HeaderProps): JSX.Element => {
  const _children = Children.toArray(children) as JSX.Element[];
  return (
    <HeaderContext.Provider value={{ level, divided }}>
      <Space
        direction="horizontal"
        justify="spaceBetween"
        className={clsx(`pluto-header pluto-bordered--bottom`, className)}
        {...props}
      >
        {_children[0]}
        {_children.length > 1 && _children[1]}
      </Space>
    </HeaderContext.Provider>
  );
};
