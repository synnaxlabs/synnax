import { Children, ReactElement, cloneElement } from "react";

import { CoreTextProps, Text } from "./Text";
import { TypographyLevel } from "./types";

import { Divider } from "@/atoms/Divider";
import { Space, SpaceProps } from "@/atoms/Space";
import { Theming } from "@/theming";

export interface TextWithIconProps extends Omit<SpaceProps, "children">, CoreTextProps {
  startIcon?: ReactElement | ReactElement[];
  endIcon?: ReactElement | ReactElement[];
  children?: string | number;
  divided?: boolean;
}

export const TextWithIcon = ({
  level = "h1",
  divided = false,
  startIcon,
  endIcon,
  children,
  color,
  ...props
}: TextWithIconProps): JSX.Element => {
  const startIcons = startIcon != null && useFormattedIcons(startIcon, level, color);
  const endIcons = endIcon != null && useFormattedIcons(endIcon, level, color);
  return (
    <Space direction="horizontal" size="small" align="center" {...props}>
      {startIcons}
      {divided && startIcon != null && <Divider direction="vertical" />}
      {children != null && (
        <Text color={color} level={level}>
          {children}
        </Text>
      )}
      {divided && endIcon != null && <Divider direction="vertical" />}
      {endIcons}
    </Space>
  );
};

const useFormattedIcons = (
  icon: ReactElement | ReactElement[],
  level: TypographyLevel,
  color?: string
): ReactElement[] => {
  const { theme } = Theming.useContext();
  const size = Number(theme.typography[level]?.lineHeight) * theme.sizes.base;
  color ??= theme.colors.text;
  return (Children.toArray(icon) as ReactElement[]).map((icon) =>
    cloneElement(icon, {
      size,
      color,
      style: { minWidth: size, ...icon.props.style },
      ...icon.props,
    })
  );
};
