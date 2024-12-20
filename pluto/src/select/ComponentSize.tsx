import { type ReactElement } from "react";
import { ComponentSize as BaseComponentSize } from "@/util/component";
import { Button, type ButtonProps } from "@/select/Button";

interface SizeEntry {
  key: BaseComponentSize;
  label: string;
}

const SIZE_DATA: SizeEntry[] = [
  { key: "large", label: "L" },
  { key: "medium", label: "M" },
  { key: "small", label: "S" },
];

export interface SelectComponentSizeProps
  extends Omit<ButtonProps<BaseComponentSize, SizeEntry>, "data" | "entryRenderKey"> {}

export const ComponentSize = ({
  children,
  ...props
}: SelectComponentSizeProps): ReactElement => (
  <Button {...props} data={SIZE_DATA} entryRenderKey="label">
    {children}
  </Button>
);
