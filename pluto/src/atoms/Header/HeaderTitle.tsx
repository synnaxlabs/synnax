import clsx from "clsx";

import { TextWithIconProps } from "../Typography/TextWithIcon";

import { useHeaderContext } from "./Header";

import { Text } from "@/atoms/Typography";

import "./Header.css";

export interface HeaderTitleProps
  extends Omit<TextWithIconProps, "level" | "divided"> {}

export const HeaderTitle = ({ className, ...props }: HeaderTitleProps): JSX.Element => {
  const { level, divided } = useHeaderContext();
  return (
    <Text.WithIcon
      className={clsx("pluto-header__text", className)}
      level={level}
      divided={divided}
      {...props}
    />
  );
};
