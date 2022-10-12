import { cloneElement } from "react";
import { Theme } from "../../Theme";
import { Divider } from "../Divider";
import Space, { SpaceProps } from "../Space/Space";
import Text, { BaseTextProps } from "./Text";
import { TypographyLevel } from "./Types";

export interface BaseTextWithIconProps
  extends Omit<SpaceProps, "children">,
    BaseTextProps {
  startIcon?: React.ReactElement;
  endIcon?: React.ReactElement;
  children?: string | number;
  divided?: boolean;
}

export interface TextWithIconProps extends BaseTextWithIconProps {}

export default function TextWithIcon({
  level = "h1",
  divided = false,
  startIcon,
  endIcon,
  children,
  ...props
}: TextWithIconProps) {
  const formattedStartIcon = startIcon && useFormattedIcon(startIcon, level);
  const formattedEndIcon = endIcon && useFormattedIcon(endIcon, level);
  return (
    <Space direction="horizontal" size="small" align="center" {...props}>
      {formattedStartIcon && formattedStartIcon}
      {divided && <Divider direction="vertical" />}
      {children && <Text level={level}>{children}</Text>}
      {divided && formattedEndIcon && <Divider direction="vertical" />}
      {formattedEndIcon && formattedEndIcon}
    </Space>
  );
}

const useFormattedIcon = (icon: React.ReactElement, level: TypographyLevel) => {
  const { theme } = Theme.useContext();
  const size = theme.typography[level]?.size;
  console.log(size);
  if (!size) return <h1>Hello</h1>;
  const color = theme.colors.text;
  return cloneElement(icon, { size, color });
};
