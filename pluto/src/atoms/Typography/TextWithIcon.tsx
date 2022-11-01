import { Children, ReactElement, cloneElement } from "react";
import { Theming } from "@/theming";
import { Divider } from "@/atoms/Divider";
import { Space, SpaceProps } from "@/atoms/Space";
import { CoreTextProps, Text } from "./Text";
import { TypographyLevel } from "./types";

export interface TextWithIconProps
  extends Omit<SpaceProps, "children">,
    CoreTextProps {
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
}: TextWithIconProps) => {
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
};

const useFormattedIcons = (
  icon: ReactElement | ReactElement[],
  level: TypographyLevel,
  color?: string
): ReactElement[] => {
  const { theme } = Theming.useContext();
  const size = Number(theme.typography[level]?.lineHeight) * theme.sizes.base;
  color = color || theme.colors.text;
  return toArray(icon).map((icon) =>
    cloneElement(icon, { size, color, ...icon.props })
  );
};

const toArray = (children: ReactElement | ReactElement[]): ReactElement[] => {
  return Children.toArray(children) as ReactElement[];
};
