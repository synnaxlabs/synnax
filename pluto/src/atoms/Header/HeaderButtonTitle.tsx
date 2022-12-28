import { Button, ButtonProps } from "../Button";
import { Typography } from "../Typography";

import { useHeaderContext } from "./Header";

export interface HeaderButtonProps extends Omit<ButtonProps, "variant" | "size"> {}

export const HeaderButtonTitle = ({
  children = "",
  className,
  onClick,
  style,
  ...props
}: HeaderButtonProps): JSX.Element => {
  const { level } = useHeaderContext();
  return (
    <Button
      variant="text"
      size={Typography.LevelComponentSizes[level]}
      style={{ flexGrow: 1, ...style }}
      onClick={onClick}
      {...props}
    >
      {children}
    </Button>
  );
};
