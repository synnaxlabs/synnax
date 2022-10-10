import { cloneElement } from "react";
import { useThemeContext } from "../../Theme/ThemeContext";
import Space, { SpaceProps } from "../Space/Space";
import Text, { BaseTextProps } from "./Text";

export interface BaseIconTextProps
  extends Omit<SpaceProps, "children">,
    BaseTextProps {
  startIcon?: React.ReactElement;
  endIcon?: React.ReactElement;
  children?: string | number;
}

export interface IconTextProps extends BaseIconTextProps {}

export default function IconText({
  level = "h1",
  startIcon,
  endIcon,
  children,
  ...props
}: IconTextProps) {
  const { theme } = useThemeContext();
  const size = theme.typography[level].size;
  const color = theme.colors.text;
  return (
    <Space direction="horizontal" size="small" align="center" {...props}>
      {startIcon && cloneElement(startIcon, { size, color })}
      {children && <Text level={level}>{children}</Text>}
      {endIcon && cloneElement(endIcon, { size, color })}
    </Space>
  );
}
