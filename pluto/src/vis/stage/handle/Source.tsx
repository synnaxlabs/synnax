import { Handle, type HandleProps } from "@xyflow/react";

export interface SourceProps extends Omit<HandleProps, "type"> {}

export const Source = ({ position, ...props }: SourceProps) => (
  <Handle type="source" position={position} {...props} />
);
