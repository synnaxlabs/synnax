import { ReactElement } from "react";
import { Text, TextProps, TypographyLevel } from "../Typography";
import {
  AiFillInfoCircle,
  AiFillWarning,
  AiOutlineClose,
  AiOutlineCheck,
  AiOutlineWarning,
} from "react-icons/ai";
import { StatusVariant } from "./types";

export interface StatusBadgeProps
  extends Omit<TextProps, "children" | "level"> {
  level?: TypographyLevel;
  message: string;
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
  warning: "var(--pluto-warning-z)",
  success: "var(--pluto-primary-z)",
  loading: "var(--pluto-text-color)",
};

export const StatusBadge = ({
  variant = "error",
  message,
  level = "p",
  ...props
}: StatusBadgeProps) => {
  return (
    <Text.WithIcon
      color={statusVariantColors[variant]}
      level={level}
      startIcon={statusVariantIcons[variant]}
      {...props}
    >
      {message}
    </Text.WithIcon>
  );
};
