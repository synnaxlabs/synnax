import { Handle, type HandleProps } from "@xyflow/react";

export interface SinkProps extends Omit<HandleProps, "type"> {}

export const Sink = ({ position, ...props }: SinkProps) => (
  <Handle type="target" position={position} {...props} />
);
