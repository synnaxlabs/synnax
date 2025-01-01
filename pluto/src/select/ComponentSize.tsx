import { type ReactElement } from "react";

import { Button, type ButtonProps } from "@/select/Button";
import { type ComponentSize as BaseComponentSize } from "@/util/component";

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
