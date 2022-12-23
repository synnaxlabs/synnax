import { ReactElement } from "react";

import {
  AiFillInfoCircle,
  AiFillWarning,
  AiOutlineCheck,
  AiOutlineClose,
  AiOutlineWarning,
} from "react-icons/ai";

import { Text, TextProps, TypographyLevel } from "../Typography";

import { StatusVariant } from "./types";

export interface StatusBadgeProps extends Omit<TextProps, "level"> {
  level?: TypographyLevel;
  variant: StatusVariant;
}

const statusVariantIcons: Record<StatusVariant, ReactElement> = {
  info: <AiFillInfoCircle />,
  warning: <AiFillWarning />,
  error: <AiOutlineClose />,
  success: <AiOutlineCheck />,
  loading: <AiOutlineWarning />,
};

const statusVariantColors: Record<StatusVariant, string> = {
  info: "var(--pluto-text-color)",
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-text-color)",
  success: "var(--pluto-primary-z)",
  loading: "var(--pluto-text-color)",
};

export const StatusText = ({
  variant = "error",
  level = "p",
  ...props
}: StatusBadgeProps): JSX.Element => {
  return (
    <Text.WithIcon
      color={statusVariantColors[variant]}
      level={level}
      startIcon={statusVariantIcons[variant]}
      {...props}
    />
  );
};
