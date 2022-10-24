import { thresholdFreedmanDiaconis } from "d3";
import React, { Children, cloneElement, ReactElement } from "react";
import { Theme } from "../../Theme";
import { Divider } from "../Divider";
import Space, { SpaceProps } from "../Space/Space";
import Text, { BaseTextProps } from "./Text";
import { TypographyLevel } from "./Types";

export interface BaseTextWithIconProps
  extends Omit<SpaceProps, "children">,
    BaseTextProps {
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
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
  color,
  ...props
}: TextWithIconProps) {
  const endIcons = startIcon && useFormattedIcons(startIcon, level, color);
  const startIcons = endIcon && useFormattedIcons(endIcon, level, color);
  return (
    <Space direction="horizontal" size="small" align="center" {...props}>
      {endIcons && endIcons.map((i) => i)}
      {divided && <Divider direction="vertical" />}
      {children && (
        <Text color={color} level={level}>
          {children}
        </Text>
      )}
      {divided && startIcons && <Divider direction="vertical" />}
      {startIcons && startIcons.map((i) => i)}
    </Space>
  );
}

const useFormattedIcons = (
  icon: ReactElement | ReactElement[],
  level: TypographyLevel,
  color?: string
): ReactElement[] => {
  const { theme } = Theme.useContext();
  const size = Number(theme.typography[level]?.lineHeight) * theme.sizes.base;
  color = color || theme.colors.text;
  return toArray(icon).map((icon) =>
    cloneElement(icon, { size, color, ...icon.props })
  );
};

const toArray = (children: ReactElement | ReactElement[]): ReactElement[] => {
  return Children.toArray(children) as ReactElement[];
};
