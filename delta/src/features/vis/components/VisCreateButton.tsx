import { Button, ButtonProps } from "@synnaxlabs/pluto";

import { VisIcon } from "./VisToolbar";

export interface VisCreateButtonProps
  extends Omit<ButtonProps, "startIcon" | "endIcon" | "children"> {}

export const VisCreateButton = (props: VisCreateButtonProps): JSX.Element => (
  <Button {...props} startIcon={<VisIcon />}>
    Create a Visualization
  </Button>
);
