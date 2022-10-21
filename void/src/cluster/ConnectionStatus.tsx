import { ConnectionState } from "./slice";
import {
  AiFillInfoCircle,
  AiFillWarning,
  AiOutlineCheck,
  AiOutlineClose,
  AiOutlineWarning,
} from "react-icons/ai";
import { Text, TextProps, TypographyLevel } from "@synnaxlabs/pluto";
import { ReactElement } from "react";
import { Connectivity } from "@synnaxlabs/client";

export interface ConnectionStatusProps {
  state: ConnectionState;
}

const connectionStatusVariants: Record<Connectivity, StatusVariant> = {
  [Connectivity.CONNECTED]: "success",
  [Connectivity.FAILED]: "error",
  [Connectivity.CONNECTING]: "info",
  [Connectivity.DISCNNECTED]: "warning",
};

export default function ConnectionStatus({ state }: ConnectionStatusProps) {
  return (
    <StatusBadge
      variant={connectionStatusVariants[state.status]}
      message={state.message}
    />
  );
}

type StatusVariant =
  | "success"
  | "error"
  | "warning"
  | "info"
  | "warning"
  | "loading";

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
