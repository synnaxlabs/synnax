import { Button, type Icon } from "@synnaxlabs/pluto";

import { CSS } from "@/css";

export interface ItemProps extends Omit<Button.ButtonProps, "children"> {
  title: string;
  icon: Icon.ReactElement;
}

export const Item = ({ title, icon, className, ...rest }: ItemProps) => (
  <Button.Button
    variant="outlined"
    className={CSS(CSS.BE("vis-layout-selector", "item"), className)}
    square={false}
    {...rest}
  >
    {icon}
    {title}
  </Button.Button>
);
