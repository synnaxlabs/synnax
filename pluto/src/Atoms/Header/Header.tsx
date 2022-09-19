import Space, { SpaceProps } from "../Space/Space";
import Text, {TextLevel} from "../Typography/Text";
import "./Header.css";
import { cloneElement, ReactElement } from "react";
import { useThemeContext } from "../../Theme/ThemeContext";
import { classList } from "../../util/css";

export interface HeadingProps extends Omit<SpaceProps, "children" | "size"> {
  level: TextLevel;
  text: string;
  icon?: ReactElement;
  textColor?: string;
}

const Header = ({
  level,
  text,
  icon,
  style,
  className,
  textColor,
  ...props
}: HeadingProps) => {
  const { theme } = useThemeContext();
  const sizeVal = theme.typography[level].size;
  return (
    <Space
      direction="horizontal"
      className={classList("pluto-header__container", className)}
      align="center"
      size="medium"
      style={{ height: sizeVal, ...style }}
      {...props}
    >
      {icon && cloneElement(icon, { size: sizeVal, style: {color: textColor }})}
      <Text level={level} style={{ color: textColor }}>{text}</Text>
    </Space>
  );
};

export default Header;
