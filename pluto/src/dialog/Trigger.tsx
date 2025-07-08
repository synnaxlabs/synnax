import { type ReactElement } from "react";

import { Button } from "@/button";
import { useContext } from "@/dialog/Dialog";

export interface TriggerProps extends Button.ButtonProps {}

export const Trigger = ({ onClick, ...rest }: TriggerProps): ReactElement => {
  const { open } = useContext();
  return (
    <Button.Button
      onClick={(e) => {
        onClick?.(e);
        open();
      }}
      {...rest}
    />
  );
};
