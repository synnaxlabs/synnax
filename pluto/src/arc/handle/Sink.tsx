import { Core, type CoreProps } from "@/arc/handle/Core";
import { CSS } from "@/css";

export interface SinkProps extends Omit<CoreProps, "type"> {}

export const Sink = ({ location, ...props }: SinkProps) => (
  <Core type="target" className={CSS.M("sink")} location={location} {...props} />
);
