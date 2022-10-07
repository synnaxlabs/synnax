import Space, { SpaceProps } from "../Space/Space";
import "./Header.css";
import { cloneElement, ReactElement } from "react";
import { useThemeContext } from "../../Theme/ThemeContext";
import clsx from "clsx";

export type FontSize = "h1" | "h2" | "h3" | "h4" | "h5" | "p" | "small";

export interface HeadingProps extends Omit<SpaceProps, "children" | "size"> {
  size: FontSize;
  text: string;
  icon?: ReactElement;
}

const fontSizeMap = (size: FontSize, children: string) => {
  const map = {
    h1: <h1>{children}</h1>,
    h2: <h2>{children}</h2>,
    h3: <h3>{children}</h3>,
    h4: <h4>{children}</h4>,
    h5: <h5>{children}</h5>,
    p: <p>{children}</p>,
    small: <h6>{children}</h6>,
  };
  return map[size];
};

const Header = ({
  size,
  text,
  icon,
  style,
  className,
  ...props
}: HeadingProps) => {
  const { theme } = useThemeContext();
  const sizeVal = theme.typography[size].size;
  return (
    <Space
      direction="horizontal"
      className={clsx("pluto-header__container", className)}
      align="center"
      size="medium"
      style={{ height: sizeVal, ...style }}
      {...props}
    >
      {icon && cloneElement(icon, { size: sizeVal })}
      {fontSizeMap(size, text)}
    </Space>
  );
};

export default Header;
