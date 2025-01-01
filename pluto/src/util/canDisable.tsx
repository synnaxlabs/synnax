import { type FC, type ReactNode } from "react";

export type CanDisabledProps<T extends object & { children?: ReactNode }> = T & {
  disabled?: boolean;
};

export const canDisable = <T extends object & { children?: ReactNode }>(
  C: FC<T>,
): FC<CanDisabledProps<T>> => {
  const O: FC<CanDisabledProps<T>> = ({ disabled = false, ...props }) =>
    disabled ? props.children : <C {...(props as T)} />;
  O.displayName = C.displayName;
  return O;
};
